"""
Apache Parquet connector for ARC-HAWK-DD scanner.
Requires: pyarrow
Also handles ORC files (same dependency).
"""

import os
from pathlib import Path

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()


def scan_parquet_file(args, file_path, profile_name, limit_rows=None):
    try:
        import pyarrow.parquet as pq
    except ImportError:
        system.print_error(args, "pyarrow not installed. Run: pip install pyarrow")
        return []

    results = []
    try:
        table = pq.read_table(file_path)
        if limit_rows:
            table = table.slice(0, limit_rows)

        schema_names = table.schema.names
        for batch in table.to_batches(max_chunksize=1000):
            for col_name in schema_names:
                col = batch.column(col_name)
                for val in col:
                    val_py = val.as_py()
                    if val_py is None:
                        continue
                    value_str = str(val_py)
                    matches = system.match_strings(args, value_str)
                    if matches:
                        validated = validate_findings(matches, args)
                        if validated:
                            for match in validated:
                                results.append({
                                    'host': str(Path(file_path).parent),
                                    'file_path': file_path,
                                    'column': col_name,
                                    'pattern_name': match['pattern_name'],
                                    'matches': match['matches'],
                                    'sample_text': match['sample_text'],
                                    'profile': profile_name,
                                    'data_source': 'parquet',
                                })
    except Exception as e:
        system.print_error(args, f"Error reading Parquet {file_path}: {e}")

    return results


def scan_orc_file(args, file_path, profile_name, limit_rows=None):
    try:
        import pyarrow.orc as orc
    except ImportError:
        system.print_error(args, "pyarrow not installed. Run: pip install pyarrow")
        return []

    results = []
    try:
        table = orc.read_table(file_path)
        if limit_rows:
            table = table.slice(0, limit_rows)

        schema_names = table.schema.names
        for batch in table.to_batches(max_chunksize=1000):
            for col_name in schema_names:
                col = batch.column(col_name)
                for val in col:
                    val_py = val.as_py()
                    if val_py is None:
                        continue
                    value_str = str(val_py)
                    matches = system.match_strings(args, value_str)
                    if matches:
                        validated = validate_findings(matches, args)
                        if validated:
                            for match in validated:
                                results.append({
                                    'host': str(Path(file_path).parent),
                                    'file_path': file_path,
                                    'column': col_name,
                                    'pattern_name': match['pattern_name'],
                                    'matches': match['matches'],
                                    'sample_text': match['sample_text'],
                                    'profile': profile_name,
                                    'data_source': 'orc',
                                })
    except Exception as e:
        system.print_error(args, f"Error reading ORC {file_path}: {e}")

    return results


def _scan_directory(args, base_path, profile_name, limit_rows, recursive, extension, scan_fn):
    results = []
    if os.path.isfile(base_path):
        return scan_fn(args, base_path, profile_name, limit_rows)

    glob = f'**/*.{extension}' if recursive else f'*.{extension}'
    for f in Path(base_path).glob(glob):
        system.print_info(args, f"Scanning: {f}")
        results += scan_fn(args, str(f), profile_name, limit_rows)
    return results


def execute(args):
    results = []
    connections = system.get_connection(args)
    sources_config = connections.get('sources', {})

    # Parquet sources
    parquet_config = sources_config.get('parquet')
    if parquet_config:
        system.print_info(args, "Running checks for Parquet sources")
        for key, config in parquet_config.items():
            for base_path in config.get('paths', []):
                results += _scan_directory(
                    args, base_path, key,
                    config.get('limit_rows'),
                    config.get('recursive', True),
                    'parquet', scan_parquet_file,
                )

    # ORC sources (same file, same dependency)
    orc_config = sources_config.get('orc')
    if orc_config:
        system.print_info(args, "Running checks for ORC sources")
        for key, config in orc_config.items():
            for base_path in config.get('paths', []):
                results += _scan_directory(
                    args, base_path, key,
                    config.get('limit_rows'),
                    config.get('recursive', True),
                    'orc', scan_orc_file,
                )

    return results
