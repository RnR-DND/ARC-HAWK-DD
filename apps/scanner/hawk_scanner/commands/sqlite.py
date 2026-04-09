"""
SQLite connector for ARC-HAWK-DD scanner.
Uses the stdlib sqlite3 module — no extra dependencies.
"""

import sqlite3

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()


def check_data_patterns(args, conn, profile_name, db_path, limit_end=None, whitelisted_tables=None):
    cursor = conn.cursor()
    cursor.execute("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
    all_tables = [row[0] for row in cursor.fetchall()]

    if whitelisted_tables:
        all_tables = [t for t in all_tables if t in whitelisted_tables]

    results = []
    total_rows = 0

    for table in all_tables:
        try:
            if limit_end:
                cursor.execute(f'SELECT * FROM "{table}" LIMIT {int(limit_end)}')
            else:
                cursor.execute(f'SELECT * FROM "{table}"')
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
                                        'host': db_path,
                                        'file_path': db_path,
                                        'table': table,
                                        'column': col,
                                        'pattern_name': match['pattern_name'],
                                        'matches': match['matches'],
                                        'sample_text': match['sample_text'],
                                        'profile': profile_name,
                                        'data_source': 'sqlite',
                                    })
        except Exception as e:
            system.print_error(args, f"Error scanning table {table}: {e}")

    cursor.close()
    system.print_success(args, f"Scanned {total_rows:,} rows across {len(all_tables)} tables in {db_path}")
    return results


def execute(args):
    results = []
    system.print_info(args, "Running checks for SQLite sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    sqlite_config = sources_config.get('sqlite')
    if not sqlite_config:
        system.print_error(args, "No SQLite connection details found in connection.yml")
        return results

    for key, config in sqlite_config.items():
        db_path = config.get('path')
        limit_end = config.get('limit_end')
        tables = config.get('tables', [])

        if not db_path:
            system.print_error(args, f"Missing 'path' for SQLite key: {key}")
            continue

        try:
            conn = sqlite3.connect(db_path)
            system.print_info(args, f"Opened SQLite database at {db_path}")
            results += check_data_patterns(
                args, conn, key, db_path,
                limit_end=limit_end,
                whitelisted_tables=tables,
            )
            conn.close()
        except Exception as e:
            system.print_error(args, f"Failed to open SQLite database {db_path}: {e}")

    return results
