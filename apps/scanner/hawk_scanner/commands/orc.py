"""
Apache ORC file format connector for ARC-HAWK-DD scanner.
Requires: pyarrow (already in requirements.txt)

ORC (Optimized Row Columnar) is a self-describing, type-aware columnar file
format designed for Hadoop workloads. Common in data lakes and Hive tables.

Configuration in connection.yml:

    sources:
      orc:
        datalake_orc:
          paths:
            - /data/hive/warehouse/customers/
            - /data/exports/events.orc
          recursive: true
          limit_rows: 10000   # optional; omit for unlimited
"""

import os
from pathlib import Path

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()


def connect_orc(args, path):
    """
    Validate that an ORC path exists and pyarrow is available.

    Args:
        args: Parsed CLI arguments
        path: File or directory path

    Returns:
        str path if valid, None otherwise
    """
    try:
        import pyarrow.orc  # noqa: F401 — validate import only
    except ImportError:
        system.print_error(
            args,
            "pyarrow not installed or missing ORC support. Run: pip install 'pyarrow>=15.0.0'"
        )
        return None

    if not os.path.exists(path):
        system.print_error(args, f"ORC path does not exist: {path}")
        return None

    system.print_info(args, f"ORC source validated: {path}")
    return path


def scan_orc_file(args, file_path, profile_name, limit_rows=None):
    """
    Scan a single ORC file for PII patterns.

    Args:
        args: Parsed CLI arguments
        file_path: Absolute path to the .orc file
        profile_name: Profile key from connection.yml
        limit_rows: Maximum rows to scan (None = all rows)

    Returns:
        List of finding dicts
    """
    try:
        import pyarrow.orc as orc
    except ImportError:
        system.print_error(args, "pyarrow not installed. Run: pip install pyarrow")
        return []

    results = []

    # Validate magic bytes — ORC files start with b'ORC'
    try:
        with open(file_path, 'rb') as fh:
            magic = fh.read(3)
        if magic != b'ORC':
            system.print_error(
                args,
                f"File does not appear to be a valid ORC file (bad magic bytes): {file_path}"
            )
            return []
    except OSError as e:
        system.print_error(args, f"Cannot read file {file_path}: {e}")
        return []

    try:
        table = orc.read_table(file_path)
    except Exception as e:
        system.print_error(args, f"Failed to parse ORC file {file_path}: {e}")
        return []

    if limit_rows:
        table = table.slice(0, limit_rows)

    schema_names = table.schema.names
    total_rows = table.num_rows
    system.print_info(
        args,
        f"Scanning ORC file: {file_path} ({total_rows:,} rows, {len(schema_names)} columns)"
    )

    try:
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
        system.print_error(args, f"Error scanning ORC file {file_path}: {e}")

    return results


def _scan_path(args, base_path, profile_name, limit_rows, recursive):
    """
    Scan a single file or a directory of .orc files.

    Args:
        args: Parsed CLI arguments
        base_path: File or directory path
        profile_name: Profile key from connection.yml
        limit_rows: Row limit per file
        recursive: Whether to recurse into subdirectories

    Returns:
        List of finding dicts
    """
    results = []

    if os.path.isfile(base_path):
        return scan_orc_file(args, base_path, profile_name, limit_rows)

    if not os.path.isdir(base_path):
        system.print_error(args, f"ORC path is neither a file nor directory: {base_path}")
        return []

    glob_pattern = '**/*.orc' if recursive else '*.orc'
    orc_files = list(Path(base_path).glob(glob_pattern))

    if not orc_files:
        system.print_info(args, f"No .orc files found in: {base_path}")
        return []

    system.print_info(args, f"Found {len(orc_files)} ORC file(s) in {base_path}")
    for f in orc_files:
        system.print_info(args, f"  Scanning: {f}")
        results += scan_orc_file(args, str(f), profile_name, limit_rows)

    return results


def execute(args):
    """
    Entry point — called by all.py and hawk_scanner CLI.

    Reads 'orc' section from connection.yml sources:

        sources:
          orc:
            datalake_orc:
              paths:
                - /data/hive/warehouse/events/
              recursive: true
              limit_rows: 5000
    """
    results = []
    system.print_info(args, "Running checks for ORC file sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    orc_config = sources_config.get('orc')

    if not orc_config:
        system.print_error(args, "No ORC source configuration found in connection.yml")
        return results

    for key, config in orc_config.items():
        paths = config.get('paths', [])
        limit_rows = config.get('limit_rows', None)
        recursive = config.get('recursive', True)

        if not paths:
            system.print_error(args, f"No paths defined for ORC profile '{key}'")
            continue

        system.print_info(args, f"Checking ORC profile '{key}'")

        for base_path in paths:
            validated_path = connect_orc(args, base_path)
            if validated_path is None:
                continue
            results += _scan_path(args, validated_path, key, limit_rows, recursive)

    system.print_success(args, f"ORC scan complete: {len(results)} finding(s)")
    return results


if __name__ == "__main__":
    execute(None)
