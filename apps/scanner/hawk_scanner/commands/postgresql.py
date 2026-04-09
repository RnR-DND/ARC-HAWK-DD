import psycopg2
from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

try:
    from sdk.sampling import should_reservoir_sample, ReservoirSampler
    from sdk.field_profiler import profile_table, attach_profiling
    _PROFILING_AVAILABLE = True
except ImportError:
    _PROFILING_AVAILABLE = False

console = Console()


def connect_postgresql(args, host, port, user, password, database):
    try:
        conn = psycopg2.connect(
            host=host,
            port=port,
            user=user,
            password=password,
            database=database
        )
        if conn:
            system.print_info(
                args, f"Connected to PostgreSQL database at {host}")
            return conn
    except Exception as e:
        system.print_error(
            args, f"Failed to connect to PostgreSQL database at {host} with error: {e}")
        return None


def check_data_patterns(args, conn, patterns, profile_name, database_name, limit_start=0, limit_end=None, whitelisted_tables=None, schemas=None):
    cursor = conn.cursor()

    # Determine schemas to scan
    if schemas:
        # User specified schemas
        schema_list = schemas if isinstance(schemas, list) else [schemas]
        schema_clause = f"table_schema IN ({','.join(['%s'] * len(schema_list))})"
        cursor.execute(
            f"SELECT table_schema, table_name FROM information_schema.tables WHERE {schema_clause}", schema_list)
    else:
        # Scan all user schemas (exclude system schemas)
        cursor.execute("""
            SELECT table_schema, table_name
            FROM information_schema.tables
            WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
        """)

    all_tables = [(row[0], row[1]) for row in cursor.fetchall()]

    # AUTO-EXCLUDE: ARC-Hawk system tables and common framework tables
    # This prevents scanning the platform's own metadata tables
    EXCLUDED_TABLES = {
        # ARC-Hawk platform tables (don't scan our own scan results!)
        'patterns', 'findings', 'assets', 'classifications',
        'asset_relationships', 'review_states', 'scan_runs',
        # Common migration/framework tables
        'schema_migrations', 'goose_db_version', 'flyway_schema_history',
        'knex_migrations', 'knex_migrations_lock',
        # PostgreSQL extension tables
        'pg_stat_statements', 'spatial_ref_sys',
    }

    # Filter out excluded tables
    filtered_tables = [
        (schema, table) for schema, table in all_tables
        if table.lower() not in EXCLUDED_TABLES
    ]

    excluded_count = len(all_tables) - len(filtered_tables)
    if excluded_count > 0:
        system.print_info(
            args, f"ℹ️  Skipped {excluded_count} system/framework tables")

    # Use filtered tables for subsequent operations
    all_tables = filtered_tables

    # Filter by whitelisted tables if specified
    if whitelisted_tables:
        tables_to_scan = [(schema, table) for schema,
                          table in all_tables if table in whitelisted_tables]
    else:
        tables_to_scan = all_tables

    results = []
    total_rows_scanned = 0

    for schema, table in tables_to_scan:
        qualified_table = f'"{schema}"."{table}"'

        # Check row count for this table
        cursor.execute(f"SELECT COUNT(*) FROM {qualified_table}")
        table_row_count = cursor.fetchone()[0]

        # Warn if table is large and no limit set
        if table_row_count > 10000 and limit_end is None:
            system.print_info(
                args, f"⚠️  Scanning large table {qualified_table} ({table_row_count:,} rows) - this may take time")

        # Build query with optional limit
        if limit_end is not None:
            query = f"SELECT * FROM {qualified_table} LIMIT {limit_end} OFFSET {limit_start}"
            system.print_info(
                args, f"Scanning {qualified_table} (LIMIT {limit_end}, OFFSET {limit_start})")
        else:
            query = f"SELECT * FROM {qualified_table}"
            system.print_info(
                args, f"Scanning {qualified_table} (all {table_row_count:,} rows)")

        cursor.execute(query)
        columns = [column[0] for column in cursor.description]

        # Reservoir sampling for large tables (P4-2)
        if _PROFILING_AVAILABLE and should_reservoir_sample(table_row_count):
            system.print_info(
                args, f"  Large table — reservoir sampling {table_row_count:,} rows → 100k sample")
            sampler = ReservoirSampler()
            for row in cursor:
                sampler.add(row)
            rows_to_scan = sampler.get_sample()
            row_count = sampler.items_seen
        else:
            rows_to_scan = cursor.fetchall()
            row_count = len(rows_to_scan)

        # Column profiling (P4-1)
        table_profiling = {}
        if _PROFILING_AVAILABLE:
            table_profiling = profile_table(columns, rows_to_scan)

        # PII match count per column for pattern_frequency
        pii_count_by_col = {col: 0 for col in columns}

        for row in rows_to_scan:
            for column, value in zip(columns, row):
                if value:
                    value_str = str(value)
                    matches = system.match_strings(args, value_str)
                    if matches:
                        validated_matches = validate_findings(matches, args)
                        if validated_matches:
                            pii_count_by_col[column] = pii_count_by_col.get(column, 0) + 1
                            for match in validated_matches:
                                finding = {
                                    'host': conn.dsn,
                                    'database': database_name,
                                    'schema': schema,
                                    'table': table,
                                    'column': column,
                                    'pattern_name': match['pattern_name'],
                                    'matches': match['matches'],
                                    'sample_text': match['sample_text'],
                                    'profile': profile_name,
                                    'data_source': 'postgresql',
                                }
                                if _PROFILING_AVAILABLE and column in table_profiling:
                                    attach_profiling(finding, table_profiling[column], pii_count_by_col)
                                results.append(finding)

        total_rows_scanned += row_count

    cursor.close()
    system.print_success(
        args, f"✅ Scanned {total_rows_scanned:,} rows across {len(tables_to_scan)} tables")
    return results


def execute(args):
    results = []
    system.print_info(args, "Running Checks for PostgreSQL Sources")
    connections = system.get_connection(args)

    if 'sources' in connections:
        sources_config = connections['sources']
        postgresql_config = sources_config.get('postgresql')

        if postgresql_config:
            patterns = system.get_fingerprint_file(args)

            for key, config in postgresql_config.items():
                host = config.get('host')
                user = config.get('user')
                port = int(config.get('port', 5432))
                password = config.get('password')
                database = config.get('database')
                limit_start = config.get('limit_start', 0)
                # NEW: Default to None (unlimited)
                limit_end = config.get('limit_end', None)
                tables = config.get('tables', [])
                # NEW: Optional schema list
                schemas = config.get('schemas', None)

                if host and user and password and database:
                    system.print_info(
                        args, f"Checking PostgreSQL Profile {key}, database {database}")

                    # Warn if limit is set
                    if limit_end is not None:
                        system.print_info(
                            args, f"⚠️  Row limit active: scanning only {limit_end} rows per table")

                    conn = connect_postgresql(
                        args, host, port, user, password, database)
                    if conn is None:
                        system.print_error(args, f"Skipping profile {key}: connection failed")
                        continue
                    if conn:
                        results += check_data_patterns(
                            args, conn, patterns, key, database,
                            limit_start=limit_start,
                            limit_end=limit_end,
                            whitelisted_tables=tables,
                            schemas=schemas  # NEW: Pass schemas
                        )
                        conn.close()
                else:
                    system.print_error(
                        args, f"Incomplete PostgreSQL configuration for key: {key}")
        else:
            system.print_error(
                args, "No PostgreSQL connection details found in connection.yml")
    else:
        system.print_error(
            args, "No 'sources' section found in connection.yml")
    return results


# Example usage
if __name__ == "__main__":
    execute(None)
