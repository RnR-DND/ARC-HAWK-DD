"""
All Sources Scanner Command
===========================

Scans all configured data sources in parallel or sequential mode.
This command aggregates results from multiple data source types.

Supported sources:
- filesystem (fs)
- postgresql, mysql, mongodb, redis, couchdb
- mssql, sqlite, snowflake, redshift, bigquery
- s3, gcs, azure_blob
- kafka, kinesis (streaming snapshot)
- slack, ms_teams
- firebase
- pptx, html, parquet, orc, avro (file formats)
- email (eml/msg)
- salesforce, jira, hubspot (SaaS)
"""

import sys
import concurrent.futures
from typing import List, Dict, Any
from hawk_scanner.internals import system
from rich.console import Console
from rich.table import Table

console = Console()

# Define supported commands and their execution order
SUPPORTED_COMMANDS = [
    # Filesystems
    'fs',
    # Relational DBs
    'postgresql',
    'mysql',
    'mssql',
    'sqlite',
    'snowflake',
    'redshift',
    # NoSQL
    'mongodb',
    'redis',
    'couchdb',
    # Cloud object stores
    's3',
    'gcs',
    'azure_blob',
    # Big data / analytics
    'bigquery',
    'parquet',
    'avro',
    # Streaming (snapshot scans)
    'kafka',
    'kinesis',
    # Collaboration / messaging
    'slack',
    'ms_teams',
    # Other SaaS
    'firebase',
    'salesforce',
    'jira',
    'hubspot',
    # File formats
    'pptx',
    'html_files',
    'email_files',
]


def execute_single_command(command: str, args) -> List[Dict[str, Any]]:
    """Execute a single scanner command and return results."""
    try:
        system.print_info(args, f"Starting {command} scan...")

        # Import the command module dynamically
        module = __import__(
            f"hawk_scanner.commands.{command}", fromlist=[command])

        # Execute the command
        results = module.execute(args)

        system.print_success(
            args, f"{command} scan completed: {len(results)} findings")
        return results

    except Exception as e:
        system.print_error(args, f"Failed to execute {command}: {str(e)}")
        import traceback
        system.print_error(args, traceback.format_exc())
        return []


def execute_parallel(args) -> List[Dict[str, Any]]:
    """Execute all commands in parallel using ThreadPoolExecutor."""
    all_results = []

    system.print_info(
        args, f"Starting parallel scan of all {len(SUPPORTED_COMMANDS)} data sources...")

    with concurrent.futures.ThreadPoolExecutor(max_workers=len(SUPPORTED_COMMANDS)) as executor:
        # Submit all tasks
        future_to_command = {
            executor.submit(execute_single_command, command, args): command
            for command in SUPPORTED_COMMANDS
        }

        # Collect results as they complete
        for future in concurrent.futures.as_completed(future_to_command):
            command = future_to_command[future]
            try:
                results = future.result()
                all_results.extend(results)
            except Exception as exc:
                system.print_error(
                    args, f"{command} generated an exception: {exc}")

    return all_results


def execute_sequential(args) -> List[Dict[str, Any]]:
    """Execute all commands sequentially."""
    all_results = []

    system.print_info(
        args, f"Starting sequential scan of all {len(SUPPORTED_COMMANDS)} data sources...")

    for command in SUPPORTED_COMMANDS:
        results = execute_single_command(command, args)
        all_results.extend(results)

    return all_results


def display_summary_table(results: List[Dict[str, Any]]):
    """Display a summary table of scan results by data source."""
    from collections import defaultdict

    # Group results by data source
    source_counts = defaultdict(int)
    severity_counts = defaultdict(lambda: defaultdict(int))

    for result in results:
        source = result.get('data_source', 'unknown')
        severity = result.get('severity', 'unknown')

        source_counts[source] += 1
        severity_counts[source][severity] += 1

    # Create and display table
    table = Table(title="Scan Summary by Data Source")
    table.add_column("Data Source", style="cyan", no_wrap=True)
    table.add_column("Total Findings", style="magenta")
    table.add_column("Critical", style="red")
    table.add_column("High", style="orange3")
    table.add_column("Medium", style="yellow")
    table.add_column("Low", style="green")

    for source in sorted(source_counts.keys()):
        counts = severity_counts[source]
        table.add_row(
            source,
            str(source_counts[source]),
            str(counts.get('Critical', 0)),
            str(counts.get('High', 0)),
            str(counts.get('Medium', 0)),
            str(counts.get('Low', 0))
        )

    console.print("\n")
    console.print(table)


def execute(args):
    """
    Execute scan across all configured data sources.

    Args:
        args: Parsed command line arguments containing:
            - connection: Path to connection configuration
            - Other scanner-specific arguments

    Returns:
        List of findings from all data sources
    """
    connections = system.get_connection(args)
    options = connections.get('options', {})
    execution_mode = options.get('execution_mode', 'sequential')

    system.print_info(args, "ARC-Hawk Multi-Source Scanner")
    system.print_info(args, f"Execution Mode: {execution_mode}")
    system.print_info(
        args, f"Supported Sources: {', '.join(SUPPORTED_COMMANDS)}")

    # Validate execution mode
    if execution_mode not in ['sequential', 'parallel']:
        system.print_error(
            args, f"Invalid execution_mode: {execution_mode}. Must be 'sequential' or 'parallel'")
        sys.exit(1)

    # Execute scans based on mode
    if execution_mode == 'parallel':
        results = execute_parallel(args)
    else:
        results = execute_sequential(args)

    # Display summary
    if results:
        display_summary_table(results)
        system.print_success(
            args, f"All scans completed: {len(results)} total findings across {len(SUPPORTED_COMMANDS)} data sources")
    else:
        system.print_info(args, "No findings detected across all data sources")

    return results
