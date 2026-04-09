"""
Azure Blob Storage connector for ARC-HAWK-DD scanner.
Requires: azure-storage-blob
Scans text/CSV/JSON blobs for PII patterns.
"""

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()

# Extensions we attempt to decode as text for scanning
TEXT_EXTENSIONS = {
    '.txt', '.csv', '.json', '.jsonl', '.log', '.xml', '.yaml', '.yml', '.tsv', '.md',
}


def connect_azure_blob(args, connection_string=None, account_name=None, account_key=None, sas_token=None):
    try:
        from azure.storage.blob import BlobServiceClient
        if connection_string:
            client = BlobServiceClient.from_connection_string(connection_string)
        elif account_name and account_key:
            client = BlobServiceClient(
                account_url=f"https://{account_name}.blob.core.windows.net",
                credential=account_key,
            )
        elif account_name and sas_token:
            client = BlobServiceClient(
                account_url=f"https://{account_name}.blob.core.windows.net",
                credential=sas_token,
            )
        else:
            raise ValueError("Provide connection_string or (account_name + account_key/sas_token)")
        system.print_info(args, "Connected to Azure Blob Storage")
        return client
    except ImportError:
        system.print_error(args, "azure-storage-blob not installed. Run: pip install azure-storage-blob")
        return None
    except Exception as e:
        system.print_error(args, f"Azure Blob connection failed: {e}")
        return None


def scan_container(args, client, container_name, profile_name, max_blobs=None):
    results = []
    try:
        container_client = client.get_container_client(container_name)
        blobs = list(container_client.list_blobs())
        if max_blobs:
            blobs = blobs[:max_blobs]

        for blob in blobs:
            name = blob.name
            ext = '.' + name.rsplit('.', 1)[-1].lower() if '.' in name else ''
            if ext not in TEXT_EXTENSIONS:
                continue

            try:
                blob_client = container_client.get_blob_client(blob)
                content = blob_client.download_blob().readall().decode('utf-8', errors='replace')
                for i, line in enumerate(content.splitlines()):
                    if not line.strip():
                        continue
                    matches = system.match_strings(args, line)
                    if matches:
                        validated = validate_findings(matches, args)
                        if validated:
                            for match in validated:
                                results.append({
                                    'host': f"azure://{container_name}",
                                    'file_path': f"{container_name}/{name}",
                                    'line_number': i + 1,
                                    'pattern_name': match['pattern_name'],
                                    'matches': match['matches'],
                                    'sample_text': match['sample_text'],
                                    'profile': profile_name,
                                    'data_source': 'azure_blob',
                                })
            except Exception as e:
                system.print_error(args, f"Error reading blob {container_name}/{name}: {e}")

    except Exception as e:
        system.print_error(args, f"Error listing blobs in {container_name}: {e}")

    return results


def execute(args):
    results = []
    system.print_info(args, "Running checks for Azure Blob Storage sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    azure_config = sources_config.get('azure_blob')
    if not azure_config:
        system.print_error(args, "No Azure Blob connection details found in connection.yml")
        return results

    for key, config in azure_config.items():
        connection_string = config.get('connection_string')
        account_name = config.get('account_name')
        account_key = config.get('account_key')
        sas_token = config.get('sas_token')
        containers = config.get('containers', [])
        max_blobs = config.get('max_blobs')

        client = connect_azure_blob(
            args,
            connection_string=connection_string,
            account_name=account_name,
            account_key=account_key,
            sas_token=sas_token,
        )
        if not client:
            continue

        if not containers:
            system.print_error(args, f"No containers specified for Azure Blob key: {key}")
            continue

        for container in containers:
            system.print_info(args, f"Scanning Azure Blob container: {container}")
            results += scan_container(args, client, container, key, max_blobs=max_blobs)

    return results
