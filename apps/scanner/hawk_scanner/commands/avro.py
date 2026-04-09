"""
Apache Avro connector for ARC-HAWK-DD scanner.
Requires: fastavro
"""

import os
from pathlib import Path

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()


def scan_avro_file(args, file_path, profile_name, limit_rows=None):
    try:
        import fastavro
    except ImportError:
        system.print_error(args, "fastavro not installed. Run: pip install fastavro")
        return []

    results = []
    try:
        with open(file_path, 'rb') as f:
            reader = fastavro.reader(f)
            for i, record in enumerate(reader):
                if limit_rows and i >= limit_rows:
                    break
                for key, value in record.items():
                    if value is None:
                        continue
                    value_str = str(value)
                    matches = system.match_strings(args, value_str)
                    if matches:
                        validated = validate_findings(matches, args)
                        if validated:
                            for match in validated:
                                results.append({
                                    'host': str(Path(file_path).parent),
                                    'file_path': file_path,
                                    'column': key,
                                    'pattern_name': match['pattern_name'],
                                    'matches': match['matches'],
                                    'sample_text': match['sample_text'],
                                    'profile': profile_name,
                                    'data_source': 'avro',
                                })
    except Exception as e:
        system.print_error(args, f"Error reading Avro {file_path}: {e}")

    return results


def execute(args):
    results = []
    system.print_info(args, "Running checks for Avro sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    avro_config = sources_config.get('avro')
    if not avro_config:
        system.print_error(args, "No Avro connection details found in connection.yml")
        return results

    for key, config in avro_config.items():
        limit_rows = config.get('limit_rows')
        recursive = config.get('recursive', True)
        for base_path in config.get('paths', []):
            if os.path.isfile(base_path):
                results += scan_avro_file(args, base_path, key, limit_rows)
            elif os.path.isdir(base_path):
                glob = '**/*.avro' if recursive else '*.avro'
                for f in Path(base_path).glob(glob):
                    results += scan_avro_file(args, str(f), key, limit_rows)

    return results
