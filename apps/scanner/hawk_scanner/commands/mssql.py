"""
MSSQL connector for ARC-HAWK-DD scanner.
Requires: pymssql (or pyodbc as fallback).
"""

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()


def connect_mssql(args, host, port, user, password, database):
    try:
        import pymssql
        conn = pymssql.connect(
            server=host,
            port=port,
            user=user,
            password=password,
            database=database,
            as_dict=False,
        )
        system.print_info(args, f"Connected to MSSQL at {host}:{port}/{database}")
        return conn
    except ImportError:
        pass  # fallback to pyodbc below
    except Exception as e:
        system.print_error(args, f"pymssql connection failed: {e}")
        return None

    try:
        import pyodbc
        dsn = (
            f"DRIVER={{ODBC Driver 17 for SQL Server}};"
            f"SERVER={host},{port};DATABASE={database};"
            f"UID={user};PWD={password}"
        )
        conn = pyodbc.connect(dsn)
        system.print_info(args, f"Connected to MSSQL (pyodbc) at {host}:{port}/{database}")
        return conn
    except Exception as e:
        system.print_error(args, f"pyodbc MSSQL connection failed: {e}")
        return None


def check_data_patterns(args, conn, patterns, profile_name, database_name,
                        limit_end=None, whitelisted_tables=None):
    cursor = conn.cursor()

    cursor.execute("""
        SELECT TABLE_SCHEMA, TABLE_NAME
        FROM INFORMATION_SCHEMA.TABLES
        WHERE TABLE_TYPE = 'BASE TABLE'
    """)
    all_tables = [(row[0], row[1]) for row in cursor.fetchall()]

    if whitelisted_tables:
        all_tables = [(s, t) for s, t in all_tables if t in whitelisted_tables]

    results = []
    total_rows = 0

    for schema, table in all_tables:
        qualified = f"[{schema}].[{table}]"
        try:
            if limit_end:
                cursor.execute(f"SELECT TOP {limit_end} * FROM {qualified}")
            else:
                cursor.execute(f"SELECT * FROM {qualified}")
            columns = [col[0] for col in cursor.description]
            for row in cursor.fetchall():
                total_rows += 1
                for col, value in zip(columns, row):
                    if value:
                        value_str = str(value)
                        matches = system.match_strings(args, value_str)
                        if matches:
                            validated = validate_findings(matches, args)
                            if validated:
                                for match in validated:
                                    results.append({
                                        'host': database_name,
                                        'database': database_name,
                                        'schema': schema,
                                        'table': table,
                                        'column': col,
                                        'pattern_name': match['pattern_name'],
                                        'matches': match['matches'],
                                        'sample_text': match['sample_text'],
                                        'profile': profile_name,
                                        'data_source': 'mssql',
                                    })
        except Exception as e:
            system.print_error(args, f"Error scanning {qualified}: {e}")

    cursor.close()
    system.print_success(args, f"Scanned {total_rows:,} rows across {len(all_tables)} tables")
    return results


def execute(args):
    results = []
    system.print_info(args, "Running checks for MSSQL sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    mssql_config = sources_config.get('mssql')
    if not mssql_config:
        system.print_error(args, "No MSSQL connection details found in connection.yml")
        return results

    patterns = system.get_fingerprint_file(args)
    for key, config in mssql_config.items():
        host = config.get('host')
        port = int(config.get('port', 1433))
        user = config.get('user')
        password = config.get('password')
        database = config.get('database')
        limit_end = config.get('limit_end')
        tables = config.get('tables', [])

        if not all([host, user, password, database]):
            system.print_error(args, f"Incomplete MSSQL config for key: {key}")
            continue

        conn = connect_mssql(args, host, port, user, password, database)
        if conn:
            results += check_data_patterns(
                args, conn, patterns, key, database,
                limit_end=limit_end,
                whitelisted_tables=tables,
            )
            conn.close()

    return results
