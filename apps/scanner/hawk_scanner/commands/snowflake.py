"""
Snowflake connector for ARC-HAWK-DD scanner.
Requires: snowflake-connector-python
"""

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()


def connect_snowflake(args, account, user, password, warehouse=None, database=None, schema=None, role=None):
    try:
        import snowflake.connector
        conn = snowflake.connector.connect(
            account=account,
            user=user,
            password=password,
            warehouse=warehouse,
            database=database,
            schema=schema,
            role=role,
        )
        system.print_info(args, f"Connected to Snowflake account: {account}")
        return conn
    except ImportError:
        system.print_error(args, "snowflake-connector-python not installed. Run: pip install snowflake-connector-python")
        return None
    except Exception as e:
        system.print_error(args, f"Snowflake connection failed: {e}")
        return None


def check_data_patterns(args, conn, database, schema, profile_name,
                        limit_end=None, whitelisted_tables=None):
    cursor = conn.cursor()

    cursor.execute(f"""
        SELECT TABLE_NAME FROM {database}.INFORMATION_SCHEMA.TABLES
        WHERE TABLE_SCHEMA = '{schema.upper()}'
        AND TABLE_TYPE = 'BASE TABLE'
    """)
    all_tables = [row[0] for row in cursor.fetchall()]
    if whitelisted_tables:
        all_tables = [t for t in all_tables if t in whitelisted_tables]

    results = []
    total_rows = 0

    for table in all_tables:
        full_table = f'"{database}"."{schema}"."{table}"'
        try:
            limit_clause = f"LIMIT {int(limit_end)}" if limit_end else ""
            cursor.execute(f"SELECT * FROM {full_table} {limit_clause}")
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
                                    'data_source': 'snowflake',
                                })
        except Exception as e:
            system.print_error(args, f"Error scanning {full_table}: {e}")

    cursor.close()
    system.print_success(args, f"Scanned {total_rows:,} rows in {database}.{schema}")
    return results


def execute(args):
    results = []
    system.print_info(args, "Running checks for Snowflake sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    sf_config = sources_config.get('snowflake')
    if not sf_config:
        system.print_error(args, "No Snowflake connection details found in connection.yml")
        return results

    for key, config in sf_config.items():
        account = config.get('account')
        user = config.get('user')
        password = config.get('password')
        warehouse = config.get('warehouse')
        database = config.get('database')
        schema = config.get('schema', 'PUBLIC')
        role = config.get('role')
        limit_end = config.get('limit_end')
        tables = config.get('tables', [])

        if not all([account, user, password, database]):
            system.print_error(args, f"Incomplete Snowflake config for key: {key}")
            continue

        conn = connect_snowflake(args, account, user, password, warehouse, database, schema, role)
        if conn:
            results += check_data_patterns(
                args, conn, database, schema, key,
                limit_end=limit_end,
                whitelisted_tables=tables,
            )
            conn.close()

    return results
