import sys
import os
import json
import importlib
import time
from rich.console import Console
from rich.table import Table
from rich.panel import Panel
from rich.text import Text
from collections import defaultdict
from hawk_scanner.internals import system
from rich import print as rprint
# SSL verification is enabled by default


def clear_screen():
    os.system('cls' if os.name == 'nt' else 'clear')

console = Console()


def load_command_module(command):
    try:
        module = importlib.import_module(f"hawk_scanner.commands.{command}")
        return module
    except Exception as e:
        rprint(f"Command '{command}' is not supported. {e}")
        sys.exit(1)


def execute_command(command, args):
    module = load_command_module(command)
    return module.execute(args)


def group_results(args, results):
    grouped_results = defaultdict(list)
    for result in results:
        connection = system.get_connection(args)
        result = system.evaluate_severity(result, connection)
        grouped_results[result['data_source']].append(result)
    return grouped_results


def format_slack_message(group, result, records_mini, mention):
    template_map = {
        's3': """
        *** PII Or Secret Found ***
        Data Source: S3 Bucket - {vulnerable_profile}
        Bucket: {bucket}
        File Path: {file_path}
        Pattern Name: {pattern_name}
        Total Exposed: {total_exposed}
        Exposed Values: {exposed_values}
        """,
        'mysql': """
        *** PII Or Secret Found ***
        Data Source: MySQL - {vulnerable_profile}
        Host: {host}
        Database: {database}
        Table: {table}
        Column: {column}
        Pattern Name: {pattern_name}
        Total Exposed: {total_exposed}
        Exposed Values: {exposed_values}
        """,
        'postgresql': """
        *** PII Or Secret Found ***
        Data Source: PostgreSQL - {vulnerable_profile}
        Host: {host}
        Database: {database}
        Table: {table}
        Column: {column}
        Pattern Name: {pattern_name}
        Total Exposed: {total_exposed}
        Exposed Values: {exposed_values}
        """,
        'mongodb': """
        *** PII Or Secret Found ***
        Data Source: MongoDB - {vulnerable_profile}
        Host: {host}
        Database: {database}
        Collection: {collection}
        Field: {field}
        Pattern Name: {pattern_name}
        Total Exposed: {total_exposed}
        Exposed Values: {exposed_values}
        """,
        'redis': """
        *** PII Or Secret Found ***
        Data Source: Redis - {vulnerable_profile}
        Host: {host}
        Key: {key}
        Pattern Name: {pattern_name}
        Total Exposed: {total_exposed}
        Exposed Values: {exposed_values}
        """,
        'firebase': """
        *** PII Or Secret Found ***
        Data Source: Firebase - {vulnerable_profile}
        Bucket: {bucket}
        File Path: {file_path}
        Pattern Name: {pattern_name}
        Total Exposed: {total_exposed}
        Exposed Values: {exposed_values}
        """,
        'gcs': """
        *** PII Or Secret Found ***
        Data Source: GCS - {vulnerable_profile}
        Bucket: {bucket}
        File Path: {file_path}
        Pattern Name: {pattern_name}
        Total Exposed: {total_exposed}
        Exposed Values: {exposed_values}
        """,
        'fs': """
        *** PII Or Secret Found ***
        Data Source: File System - {vulnerable_profile}
        File Path: {file_path}
        Pattern Name: {pattern_name}
        Total Exposed: {total_exposed}
        Exposed Values: {exposed_values}
        """,
        'slack': """
        *** PII Or Secret Found ***
        Data Source: Slack - {vulnerable_profile}
        Channel Name: {channel_name}
        Message Link: {message_link}
        Pattern Name: {pattern_name}
        Total Exposed: {total_exposed}
        Exposed Values: {exposed_values}
        """,
        'couchdb': """
        *** PII Or Secret Found ***
        Data Source: CouchDB - {vulnerable_profile}
        Host: {host}
        Database: {database}
        Document ID: {doc_id}
        Field: {field}
        Pattern Name: {pattern_name}
        Total Exposed: {total_exposed}
        Exposed Values: {exposed_values}
        """,
        'gdrive': """
        *** PII Or Secret Found ***
        Data Source: Google Drive - {vulnerable_profile}
        File Name: {file_name}
        Pattern Name: {pattern_name}
        Total Exposed: {total_exposed}
        Exposed Values: {exposed_values}
        """,
        'gdrive_workspace': """
        *** PII Or Secret Found ***
        Data Source: Google Drive Workspace - {vulnerable_profile}
        File Name: {file_name}
        User: {user}
        Pattern Name: {pattern_name}
        Total Exposed: {total_exposed}
        Exposed Values: {exposed_values}
        """,
        'text': """
        *** PII Or Secret Found ***
        Data Source: Text - {vulnerable_profile}
        Pattern Name: {pattern_name}
        Total Exposed: {total_exposed}
        Exposed Values: {exposed_values}
        """
    }
    return f"{mention} " + template_map.get(group, "").format(
        vulnerable_profile=result['profile'],
        bucket=result.get('bucket', ''),
        file_path=result.get('file_path', ''),
        host=result.get('host', ''),
        database=result.get('database', ''),
        table=result.get('table', ''),
        column=result.get('column', ''),
        doc_id=result.get('doc_id', ''),
        channel_name=result.get('channel_name', ''),
        message_link=result.get('message_link', ''),
        file_name=result.get('file_name', ''),
        user=result.get('user', ''),
        pattern_name=result['pattern_name'],
        total_exposed=str(len(result['matches'])),
        exposed_values=records_mini
    )


def add_columns_to_table(group, table):
    if group in ['s3', 'firebase', 'gcs']:
        table.add_column("Bucket > File Path")
    elif group in ['mysql', 'postgresql']:
        table.add_column("Host > Database > Table.Column")
    elif group == 'redis':
        table.add_column("Host > Key")
    elif group == 'mongodb':
        table.add_column("Host > Database > Collection > Field")
    elif group == 'slack':
        table.add_column("Channel Name > Message Link")
    elif group == 'gdrive':
        table.add_column("File Name")
    elif group == 'gdrive_workspace':
        table.add_column("File Name")
        table.add_column("User")
    elif group == 'couchdb':
        table.add_column("Host > Database > Document ID > Field")
    elif group == 'fs':
        table.add_column("File Path")
    table.add_column("Pattern Name")
    table.add_column("Total Exposed")
    table.add_column("Exposed Values")
    table.add_column("Sample Text")


def main():
    start_time = time.time()

    args = system.parse_args()

    # NEW: Read scan config file if provided (from Go backend)
    if hasattr(args, 'config') and args.config:
        try:
            with open(args.config, 'r') as f:
                scan_config = json.load(f)
                # Extract ingest_url if not provided via command line
                if hasattr(args, 'ingest_url') and args.ingest_url:
                    pass  # Use command line value
                elif 'ingest_url' in scan_config:
                    args.ingest_url = scan_config['ingest_url']
                    rprint(
                        f"[INFO] Using ingest_url from config: {args.ingest_url}")

                # Extract scan_id if present
                if 'scan_id' in scan_config:
                    args.scan_id = scan_config['scan_id']
                    rprint(f"[INFO] Using Scan ID from config: {args.scan_id}")

        except Exception as e:
            rprint(f"[WARNING] Failed to read scan config: {e}")

    system.print_banner(args)
    results = []

    if args.command:
        connections = system.get_connection(args)
        data_sources = connections.get('sources', {}).keys()
        commands = [args.command] if args.command != 'all' else data_sources
        for command in commands:
            results.extend(execute_command(command, args))
    else:
        system.print_error(args, "Please provide a command to execute")
        sys.exit(1)

    grouped_results = group_results(args, results)
    if args.json:
        if args.json:
            with open(args.json, 'w') as file:
                file.write(json.dumps(grouped_results, indent=4))
            system.print_success(args, f"Results saved to {args.json}")
        else:
            rprint(json.dumps(grouped_results, indent=4))
        sys.exit(0)

    if args.csv:
        import csv
        with open(args.csv, 'w', newline='', encoding='utf-8') as file:
            writer = csv.writer(file)
            writer.writerow(["Sl. No.", "Data Source", "Vulnerable Profile", "Location",
                            "Pattern Name", "Match Value", "Sample Text", "Severity", "Severity Description"])

            i = 1
            for group, group_data in grouped_results.items():
                for result in group_data:
                    # Construct Location string based on group type
                    location = ""
                    if group in ['s3', 'firebase', 'gcs']:
                        location = f"{result.get('bucket', '')} > {result.get('file_path', '')}"
                    elif group in ['mysql', 'postgresql']:
                        location = f"{result.get('host', '')} > {result.get('database', '')} > {result.get('table', '')}.{result.get('column', '')}"
                    elif group == 'mongodb':
                        location = f"{result.get('host', '')} > {result.get('database', '')} > {result.get('collection', '')} > {result.get('field', '')}"
                    elif group == 'redis':
                        location = f"{result.get('host', '')} > {result.get('key', '')}"
                    elif group == 'slack':
                        location = f"{result.get('channel_name', '')} > {result.get('message_link', '')}"
                    elif group == 'couchdb':
                        location = f"{result.get('host', '')} > {result.get('database', '')} > {result.get('doc_id', '')} > {result.get('field', '')}"
                    elif group == 'gdrive':
                        location = f"{result.get('file_name', '')}"
                    elif group == 'gdrive_workspace':
                        location = f"{result.get('file_name', '')} (User: {result.get('user', '')})"
                    elif group == 'fs':
                        location = result.get('file_path', '')
                    elif group == 'text':
                        location = "Text Input"

                    # Explode matches into individual rows
                    for match in result['matches']:
                        writer.writerow([
                            i,
                            group,
                            result.get('profile', ''),
                            location,
                            result.get('pattern_name', ''),
                            match,
                            result.get('sample_text', ''),
                            result.get('severity', 'unknown'),
                            result.get('severity_description', '')
                        ])
                        i += 1
        system.print_success(args, f"Results saved to {args.csv}")
        sys.exit(0)

    if args.stdout:
        rprint(json.dumps(grouped_results, indent=4))
        sys.exit(0)

    # Display results in the table format
    console.print(
        Panel(Text("Now, let's look at findings!", justify="center")))

    for group, group_data in grouped_results.items():
        table = Table(show_header=True, header_style="bold magenta", show_lines=True,
                      title=f"[bold blue]Total {len(group_data)} findings in {group}[/bold blue]")
        table.add_column("Sl. No.")
        table.add_column("Vulnerable Profile")
        add_columns_to_table(group, table)
        for i, result in enumerate(group_data, 1):
            records_mini = ', '.join(result['matches']) if len(result['matches']) < 25 else ', '.join(
                result['matches'][:25]) + f" + {len(result['matches']) - 25} more"
            connection = system.get_connection(args)
            mention = connection.get('notify', {}).get(
                'slack', {}).get('mention', '')
            slack_message = format_slack_message(
                group, result, records_mini, mention)
            if slack_message:
                system.create_jira_ticket(args, result, slack_message)
                system.SlackNotify(slack_message, args)

            if group == 's3':
                table.add_row(str(i), result['profile'], f"{result['bucket']} > {result['file_path']}",
                              result['pattern_name'], str(len(result['matches'])), records_mini, result['sample_text'])
            elif group in ['mysql', 'postgresql']:
                table.add_row(str(i), result['profile'],
                              f"{result['host']} > {result['database']} > {result['table']}.{result['column']}",
                              result['pattern_name'], str(len(result['matches'])), records_mini, result['sample_text'])
            elif group == 'mongodb':
                table.add_row(str(i), result['profile'],
                              f"{result['host']} > {result['database']} > {result['collection']} > {result['field']}",
                              result['pattern_name'], str(len(result['matches'])), records_mini, result['sample_text'])
            elif group == 'slack':
                table.add_row(str(i), result['profile'],
                              f"{result['channel_name']} > {result['message_link']}",
                              result['pattern_name'], str(len(result['matches'])), records_mini, result['sample_text'])
            elif group == 'redis':
                table.add_row(str(i), result['profile'], f"{result['host']} > {result['key']}",
                              result['pattern_name'], str(len(result['matches'])), records_mini, result['sample_text'])
            elif group in ['firebase', 'gcs']:
                table.add_row(str(i), result['profile'], f"{result['bucket']} > {result['file_path']}",
                              result['pattern_name'], str(len(result['matches'])), records_mini, result['sample_text'])
            elif group == 'fs':
                table.add_row(str(i), result['profile'], result['file_path'], result['pattern_name'],
                              str(len(result['matches'])), records_mini, result['sample_text'])
            elif group == 'couchdb':
                table.add_row(str(i), result['profile'],
                              f"{result['host']} > {result['database']} > {result['doc_id']} > {result['field']}",
                              result['pattern_name'], str(len(result['matches'])), records_mini, result['sample_text'])
            elif group == 'gdrive':
                table.add_row(str(i), result['profile'], result['file_name'], result['pattern_name'],
                              str(len(result['matches'])), records_mini, result['sample_text'])
            elif group == 'gdrive_workspace':
                table.add_row(str(i), result['profile'], result['file_name'], result['user'],
                              result['pattern_name'], str(len(result['matches'])), records_mini, result['sample_text'])
            elif group == 'text':
                table.add_row(str(i), result['profile'], result['pattern_name'],
                              str(len(result['matches'])), records_mini, result['sample_text'])

        console.print(table)

    # NEW: Auto-ingestion to backend API
    if hasattr(args, 'ingest_url') and args.ingest_url:
        from hawk_scanner.internals.auto_ingest import ingest_scan_results, validate_ingest_url

        if validate_ingest_url(args.ingest_url):
            scan_metadata = {
                "scanner_version": "hawk-eye-scanner",
                "scan_timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
                "command": args.command,
                "execution_time": time.time() - start_time,
                "total_findings": sum(len(results) for results in grouped_results.values())
            }
            ingest_scan_results(args, grouped_results, scan_metadata)
        else:
            console.print(
                f"[bold red]❌ Invalid ingestion URL: {args.ingest_url}[/bold red]")
            console.print(
                "[yellow]URL should end with /ingest, /api/v1/ingest, or /api/ingest[/yellow]")

    if args.hawk_thuu:
        console.print("Hawk thuuu, Spitting on that thang!....")
        import shutil
        if os.path.exists("data"):
            shutil.rmtree("data")
        time.sleep(2)
        console.print("Cleaned hawk data! 🧹")

    # Measure and print the total execution time
    end_time = time.time()
    execution_time = end_time - start_time
    rprint(
        f"[bold green]Execution completed in {execution_time:.2f} seconds.[/bold green]")


if __name__ == '__main__':
    main()
