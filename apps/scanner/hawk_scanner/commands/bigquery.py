"""
BigQuery connector for ARC-HAWK-DD scanner.
Requires: google-cloud-bigquery
"""

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()


def connect_bigquery(args, project_id, credentials_path=None):
    try:
        from google.cloud import bigquery
        if credentials_path:
            import os
            os.environ.setdefault('GOOGLE_APPLICATION_CREDENTIALS', credentials_path)
        client = bigquery.Client(project=project_id)
        system.print_info(args, f"Connected to BigQuery project: {project_id}")
        return client
    except ImportError:
        system.print_error(args, "google-cloud-bigquery not installed. Run: pip install google-cloud-bigquery")
        return None
    except Exception as e:
        system.print_error(args, f"BigQuery connection failed: {e}")
        return None


def check_data_patterns(args, client, project_id, dataset_id, profile_name,
                        limit_end=None, whitelisted_tables=None):
    from google.cloud import bigquery

    results = []
    total_rows = 0

    try:
        tables = list(client.list_tables(f"{project_id}.{dataset_id}"))
    except Exception as e:
        system.print_error(args, f"Failed to list tables in {dataset_id}: {e}")
        return results

    if whitelisted_tables:
        tables = [t for t in tables if t.table_id in whitelisted_tables]

    for table_ref in tables:
        table_id = f"{project_id}.{dataset_id}.{table_ref.table_id}"
        try:
            limit_clause = f"LIMIT {int(limit_end)}" if limit_end else ""
            query = f"SELECT * FROM `{table_id}` {limit_clause}"
            rows = client.query(query).result()
            schema_fields = [f.name for f in rows.schema]

            for row in rows:
                total_rows += 1
                for col in schema_fields:
                    value = row[col]
                    if value is not None:
                        value_str = str(value)
                        matches = system.match_strings(args, value_str)
                        if matches:
                            validated = validate_findings(matches, args)
                            if validated:
                                for match in validated:
                                    results.append({
                                        'host': project_id,
                                        'database': dataset_id,
                                        'table': table_ref.table_id,
                                        'column': col,
                                        'pattern_name': match['pattern_name'],
                                        'matches': match['matches'],
                                        'sample_text': match['sample_text'],
                                        'profile': profile_name,
                                        'data_source': 'bigquery',
                                    })
        except Exception as e:
            system.print_error(args, f"Error scanning {table_id}: {e}")

    system.print_success(args, f"Scanned {total_rows:,} rows in {dataset_id}")
    return results


def execute(args):
    results = []
    system.print_info(args, "Running checks for BigQuery sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    bq_config = sources_config.get('bigquery')
    if not bq_config:
        system.print_error(args, "No BigQuery connection details found in connection.yml")
        return results

    for key, config in bq_config.items():
        project_id = config.get('project_id')
        datasets = config.get('datasets', [])
        credentials_path = config.get('credentials_path')
        limit_end = config.get('limit_end')
        tables = config.get('tables', [])

        if not project_id:
            system.print_error(args, f"Missing project_id for BigQuery key: {key}")
            continue

        client = connect_bigquery(args, project_id, credentials_path)
        if not client:
            continue

        for dataset_id in datasets:
            system.print_info(args, f"Scanning BigQuery dataset: {project_id}.{dataset_id}")
            results += check_data_patterns(
                args, client, project_id, dataset_id, key,
                limit_end=limit_end,
                whitelisted_tables=tables,
            )

    return results
