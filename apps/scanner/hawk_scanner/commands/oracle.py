"""
Oracle Database connector for ARC-HAWK-DD scanner.
Requires: cx_Oracle (or oracledb) — uses thin mode so no Oracle Instant Client needed.

Install: pip install oracledb
Note: oracledb is the modern drop-in successor to cx_Oracle.
"""

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

# Oracle system tables to skip (don't scan internal Oracle catalog data)
EXCLUDED_SCHEMAS = {
    'SYS', 'SYSTEM', 'OUTLN', 'DBSNMP', 'APPQOSSYS', 'DBSFWUSER',
    'GGSYS', 'ANONYMOUS', 'CTXSYS', 'DVSYS', 'DVF', 'GSMADMIN_INTERNAL',
    'MDSYS', 'OLAPSYS', 'ORDDATA', 'ORDSYS', 'REMOTE_SCHEDULER_AGENT',
    'SI_INFORMTN_SCHEMA', 'SPATIAL_CSW_ADMIN_USR', 'SPATIAL_WFS_ADMIN_USR',
    'SYS$UMF', 'SYSBACKUP', 'SYSDG', 'SYSKM', 'SYSRAC', 'WMSYS', 'XDB', 'XS$NULL',
}

EXCLUDED_TABLES = {
    'PATTERNS', 'FINDINGS', 'ASSETS', 'CLASSIFICATIONS',
    'ASSET_RELATIONSHIPS', 'REVIEW_STATES', 'SCAN_RUNS',
}


def connect_oracle(args, host, port, user, password, service_name):
    """
    Connect to an Oracle database using oracledb thin mode.

    Args:
        args: Parsed CLI arguments
        host: Oracle host
        port: Oracle listener port (default 1521)
        user: Username
        password: Password
        service_name: Oracle service name (e.g. ORCL, XE)

    Returns:
        Connection object or None on failure
    """
    try:
        import oracledb
    except ImportError:
        system.print_error(
            args,
            "oracledb not installed. Run: pip install oracledb"
        )
        return None

    try:
        # Use thin mode — no Oracle Instant Client required
        oracledb.init_oracle_client()  # no-op in thin mode; safe to call
    except Exception:
        pass  # thin mode doesn't need this

    dsn = f"{host}:{port}/{service_name}"
    try:
        conn = oracledb.connect(user=user, password=password, dsn=dsn)
        system.print_info(args, f"Connected to Oracle at {host}:{port}/{service_name}")
        return conn
    except Exception as e:
        system.print_error(
            args,
            f"Failed to connect to Oracle at {host}:{port}/{service_name}: {e}"
        )
        return None


def check_data_patterns(args, conn, patterns, profile_name, database_name,
                        limit_start=0, limit_end=None,
                        whitelisted_tables=None, whitelisted_schemas=None):
    """
    Scan Oracle tables for PII patterns.

    Args:
        args: Parsed CLI arguments
        conn: Active Oracle connection
        patterns: Fingerprint patterns dict
        profile_name: Profile key from connection.yml
        database_name: Oracle service name (used as logical DB name)
        limit_start: Row offset
        limit_end: Maximum rows per table (None = unlimited)
        whitelisted_tables: If set, only scan these tables
        whitelisted_schemas: If set, only scan these schemas

    Returns:
        List of finding dicts
    """
    cursor = conn.cursor()
    results = []
    total_rows_scanned = 0

    # Build schema filter
    if whitelisted_schemas:
        schema_list = [s.upper() for s in whitelisted_schemas]
        schema_in = ', '.join(f"'{s}'" for s in schema_list)
        schema_clause = f"AND OWNER IN ({schema_in})"
    else:
        excluded_in = ', '.join(f"'{s}'" for s in EXCLUDED_SCHEMAS)
        schema_clause = f"AND OWNER NOT IN ({excluded_in})"

    query = f"""
        SELECT OWNER, TABLE_NAME
        FROM ALL_TABLES
        WHERE 1=1 {schema_clause}
        ORDER BY OWNER, TABLE_NAME
    """
    cursor.execute(query)
    all_tables = [(row[0], row[1]) for row in cursor.fetchall()]

    # Filter out ARC-Hawk internal tables
    all_tables = [
        (schema, table) for schema, table in all_tables
        if table.upper() not in EXCLUDED_TABLES
    ]

    # Apply whitelist filter
    if whitelisted_tables:
        upper_whitelist = {t.upper() for t in whitelisted_tables}
        tables_to_scan = [
            (schema, table) for schema, table in all_tables
            if table.upper() in upper_whitelist
        ]
    else:
        tables_to_scan = all_tables

    system.print_info(args, f"Oracle: scanning {len(tables_to_scan)} tables")

    for schema, table in tables_to_scan:
        qualified = f'"{schema}"."{table}"'

        # Get row count
        try:
            cursor.execute(f"SELECT COUNT(*) FROM {qualified}")
            table_row_count = cursor.fetchone()[0]
        except Exception as e:
            system.print_error(args, f"Cannot count rows in {qualified}: {e}")
            continue

        if table_row_count > 10000 and limit_end is None:
            system.print_info(
                args,
                f"Large table {qualified} ({table_row_count:,} rows) — consider setting limit_end"
            )

        # Oracle uses FETCH FIRST / OFFSET for pagination
        try:
            if limit_end is not None:
                row_query = (
                    f"SELECT * FROM {qualified} "
                    f"OFFSET {limit_start} ROWS FETCH NEXT {limit_end} ROWS ONLY"
                )
            else:
                row_query = f"SELECT * FROM {qualified}"

            cursor.execute(row_query)
        except Exception as e:
            system.print_error(args, f"Cannot query {qualified}: {e}")
            continue

        columns = [col[0] for col in cursor.description]

        # Reservoir sampling for large tables
        if _PROFILING_AVAILABLE and should_reservoir_sample(table_row_count):
            system.print_info(
                args,
                f"  Large table — reservoir sampling {table_row_count:,} rows"
            )
            sampler = ReservoirSampler()
            for row in cursor:
                sampler.add(row)
            rows_to_scan = sampler.get_sample()
            row_count = sampler.items_seen
        else:
            rows_to_scan = cursor.fetchall()
            row_count = len(rows_to_scan)

        # Column profiling
        table_profiling = {}
        if _PROFILING_AVAILABLE:
            table_profiling = profile_table(columns, rows_to_scan)

        pii_count_by_col = {col: 0 for col in columns}

        for row in rows_to_scan:
            for column, value in zip(columns, row):
                if value is None:
                    continue
                value_str = str(value)
                matches = system.match_strings(args, value_str)
                if matches:
                    validated = validate_findings(matches, args)
                    if validated:
                        pii_count_by_col[column] = pii_count_by_col.get(column, 0) + 1
                        for match in validated:
                            finding = {
                                'host': f"{conn.dsn}",
                                'database': database_name,
                                'schema': schema,
                                'table': table,
                                'column': column,
                                'pattern_name': match['pattern_name'],
                                'matches': match['matches'],
                                'sample_text': match['sample_text'],
                                'profile': profile_name,
                                'data_source': 'oracle',
                            }
                            if _PROFILING_AVAILABLE and column in table_profiling:
                                attach_profiling(finding, table_profiling[column], pii_count_by_col)
                            results.append(finding)

        total_rows_scanned += row_count

    cursor.close()
    system.print_success(
        args,
        f"Oracle: scanned {total_rows_scanned:,} rows across {len(tables_to_scan)} tables"
    )
    return results


def execute(args):
    """
    Entry point — called by all.py and hawk_scanner CLI.

    Reads 'oracle' section from connection.yml sources:

        sources:
          oracle:
            prod:
              host: db.example.com
              port: 1521
              user: scott
              password: tiger
              service_name: ORCL
              limit_end: 5000       # optional
              schemas: [MYAPP]      # optional
              tables: [CUSTOMERS]   # optional
    """
    results = []
    system.print_info(args, "Running checks for Oracle sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    oracle_config = sources_config.get('oracle')

    if not oracle_config:
        system.print_error(args, "No Oracle connection details found in connection.yml")
        return results

    patterns = system.get_fingerprint_file(args)

    for key, config in oracle_config.items():
        host = config.get('host')
        port = int(config.get('port', 1521))
        user = config.get('user')
        password = config.get('password')
        service_name = config.get('service_name', config.get('database', ''))
        limit_start = config.get('limit_start', 0)
        limit_end = config.get('limit_end', None)
        tables = config.get('tables', [])
        schemas = config.get('schemas', None)

        if not all([host, user, password, service_name]):
            system.print_error(
                args,
                f"Incomplete Oracle config for '{key}'. "
                f"Required: host, user, password, service_name"
            )
            continue

        system.print_info(args, f"Checking Oracle profile '{key}', service {service_name}")

        if limit_end is not None:
            system.print_info(
                args, f"Row limit active: scanning up to {limit_end} rows per table"
            )

        conn = connect_oracle(args, host, port, user, password, service_name)
        if conn is None:
            system.print_error(args, f"Skipping profile '{key}': connection failed")
            continue

        try:
            results += check_data_patterns(
                args, conn, patterns, key, service_name,
                limit_start=limit_start,
                limit_end=limit_end,
                whitelisted_tables=tables,
                whitelisted_schemas=schemas,
            )
        finally:
            conn.close()

    return results


if __name__ == "__main__":
    execute(None)
