"""
Amazon Redshift connector for ARC-HAWK-DD scanner.
Redshift is Postgres-compatible — uses psycopg2 with Redshift endpoint.
"""

import psycopg2

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()


def connect_redshift(args, host, port, user, password, database):
    try:
        conn = psycopg2.connect(
            host=host,
            port=port,
            user=user,
            password=password,
            database=database,
            sslmode='require',
        )
        system.print_info(args, f"Connected to Redshift at {host}:{port}/{database}")
        return conn
    except Exception as e:
        system.print_error(args, f"Redshift connection failed: {e}")
        return None


def check_data_patterns(args, conn, database, profile_name,
                        limit_end=None, whitelisted_tables=None, schemas=None):
    cursor = conn.cursor()

    schema_clause = ""
    schema_params = []
    if schemas:
        placeholders = ','.join(['%s'] * len(schemas))
        schema_clause = f"AND schemaname IN ({placeholders})"
        schema_params = schemas
    else:
        # Exclude Redshift system schemas
        schema_clause = "AND schemaname NOT IN ('pg_catalog', 'information_schema', 'pg_internal')"

    cursor.execute(
        f"SELECT schemaname, tablename FROM pg_tables WHERE 1=1 {schema_clause}",
        schema_params,
    )
    all_tables = [(row[0], row[1]) for row in cursor.fetchall()]
    if whitelisted_tables:
        all_tables = [(s, t) for s, t in all_tables if t in whitelisted_tables]

    results = []
    total_rows = 0

    for schema, table in all_tables:
        qualified = f'"{schema}"."{table}"'
        try:
            if limit_end:
                cursor.execute(f"SELECT * FROM {qualified} LIMIT {int(limit_end)}")
            else:
                cursor.execute(f"SELECT * FROM {qualified}")
            columns = [col[0] for col in cursor.description]
            for row in cursor.fetchall():
                total_rows += 1
                for col, value in zip(columns, row):
                    if value is None:
                        continue
                    value_str = str(value)
                    matches = system.match_strings(args, value_str)
                    if matches:
                        validated = validate_findings(matches, args)
                        if validated:
                            for match in validated:
                                results.append({
                                    'host': database,
                                    'database': database,
                                    'schema': schema,
                                    'table': table,
                                    'column': col,
                                    'pattern_name': match['pattern_name'],
                                    'matches': match['matches'],
                                    'sample_text': match['sample_text'],
                                    'profile': profile_name,
                                    'data_source': 'redshift',
                                })
        except Exception as e:
            system.print_error(args, f"Error scanning {qualified}: {e}")

    cursor.close()
    system.print_success(args, f"Scanned {total_rows:,} rows in {database}")
    return results


def execute(args):
    results = []
    system.print_info(args, "Running checks for Redshift sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    rs_config = sources_config.get('redshift')
    if not rs_config:
        system.print_error(args, "No Redshift connection details found in connection.yml")
        return results

    for key, config in rs_config.items():
        host = config.get('host')
        port = int(config.get('port', 5439))
        user = config.get('user')
        password = config.get('password')
        database = config.get('database')
        limit_end = config.get('limit_end')
        tables = config.get('tables', [])
        schemas = config.get('schemas')

        if not all([host, user, password, database]):
            system.print_error(args, f"Incomplete Redshift config for key: {key}")
            continue

        conn = connect_redshift(args, host, port, user, password, database)
        if conn:
            results += check_data_patterns(
                args, conn, database, key,
                limit_end=limit_end,
                whitelisted_tables=tables,
                schemas=schemas,
            )
            conn.close()

    return results
