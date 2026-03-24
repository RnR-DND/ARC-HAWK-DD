import os
import yaml
import json
import subprocess
import requests
import sys
import time
try:
    from rich.console import Console
    from rich.table import Table
    from rich.panel import Panel
    from rich import print as rprint
    rich_available = True
except ImportError:
    rich_available = False


# Resolve paths relative to the script location (assuming script is in scripts/automation)
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "../../"))

CONNECTION_FILE = os.path.join(PROJECT_ROOT, "apps/scanner/config/connection.yml")
SCANNER_CMD = "hawk_scanner"  # Assumes it's in PATH
BACKEND_URL = "http://localhost:8080/api/v1/scans/ingest"

def print_table(json_file):
    try:
        with open(json_file, 'r') as f:
            data = json.load(f)
            
        console = Console()
        
        # Data is grouped by source_type in the JSON output of hawk_scanner?
        # main.py: grouped_results[result['data_source']].append(result)
        # So data is { 'fs': [ ...list of results... ] }
        
        if not data:
            console.print(f"[yellow]⚠️ No findings found in {json_file}[/yellow]")
            return

        for group, items in data.items():
            if not items:
                console.print(f"[green]✅ No findings found for {group}[/green]")
                continue

            table = Table(show_header=True, header_style="bold magenta", show_lines=True, 
                          title=f"[bold blue]Total {len(items)} findings in {group}[/bold blue]")
            
            table.add_column("Sl. No.")
            table.add_column("Vulnerable Profile")
            table.add_column("Location") # Simplified generic column
            table.add_column("Pattern Name")
            table.add_column("Severity")
            table.add_column("Total Exposed")
            table.add_column("Exposed Values")
            table.add_column("Sample Text")
            
            for i, result in enumerate(items, 1):
                # Construct Location (simplified logic for common types)
                location = result.get('file_path', '')
                if group == 's3': location = f"{result.get('bucket')} > {result.get('file_path')}"
                if group in ['mysql', 'postgresql']: location = f"{result.get('host')} > {result.get('table')}.{result.get('column')}"
                
                matches = result.get('matches', [])
                records_mini = ', '.join(matches) if len(matches) < 5 else ', '.join(matches[:5]) + f" + {len(matches) - 5} more"
                
                sev = result.get('severity', 'Unknown')
                sev_styled = sev
                if sev.lower() in ['critical', 'high']:
                    sev_styled = f"[red]{sev}[/red]"
                elif sev.lower() == 'medium':
                    sev_styled = f"[yellow]{sev}[/yellow]"
                
                table.add_row(
                    str(i),
                    result.get('profile', ''),
                    location,
                    result.get('pattern_name', ''),
                    sev_styled,
                    str(len(matches)),
                    records_mini,
                    result.get('sample_text', '')
                )
            
            console.print(table)
            
    except Exception as e:
        print(f"⚠️ Failed to print table: {e}")


def load_connection(file_path):
    if not os.path.exists(file_path):
        print(f"❌ Connection file not found: {file_path}")
        sys.exit(1)
    with open(file_path, 'r') as f:
        return yaml.safe_load(f)

def run_scan(source_type, source_name):
    output_file = f"output_{source_name}.json"
    print(f"🔍 Scanning source: {source_name} ({source_type})...")
    
    # Construct command: hawk_scanner <type> --connection ... --json ...
    # Note: hawk_scanner takes the source type as command (e.g. 'fs', 's3')
    # Use config from the connection file
    
    cmd = [
        SCANNER_CMD,
        source_type,
        "--connection", CONNECTION_FILE,
        "--json", output_file,
        # "--quiet", # Disabled to show terminal output
        # "--shutup"
    ]
    
    try:
        subprocess.run(cmd, check=True)
        print(f"✅ Scan complete for {source_name}")
        
        # Print table if rich is available
        if rich_available and os.path.exists(output_file):
            print_table(output_file)
            
        return output_file
    except subprocess.CalledProcessError as e:
        print(f"❌ Scan failed for {source_name}: {e}")
        return None

def ingest_results(json_file):
    if not os.path.exists(json_file):
        print(f"❌ Output file not found: {json_file}")
        return False
        
    print(f"📤 Ingesting results from {json_file}...")
    with open(json_file, 'r') as f:
        data = json.load(f)
        
    try:
        response = requests.post(BACKEND_URL, json=data)
        if response.status_code in [200, 201]:
            print(f"✅ Ingestion successful!")
            return True
        else:
            print(f"❌ Ingestion failed: {response.status_code} - {response.text}")
            return False
    except requests.exceptions.RequestException as e:
        print(f"❌ Failed to contact backend: {e}")
        return False

def main():
    print("🚀 Starting Unified Scan & Ingest...")
    
    config = load_connection(CONNECTION_FILE)
    sources = config.get('sources', {})
    
    if not sources:
        print("⚠️ No sources found in connection.yml")
        sys.exit(0)
        
    success_count = 0
    total_count = 0
    
    for source_type, source_profiles in sources.items():
        for source_name, profile in source_profiles.items():
            total_count += 1
            json_file = run_scan(source_type, source_name)
            
            if json_file:
                if ingest_results(json_file):
                    success_count += 1
                
                # Cleanup
                if os.path.exists(json_file):
                    os.remove(json_file)
                    
    print(f"\n✨ Completed! Successfully processed {success_count}/{total_count} sources.")

if __name__ == "__main__":
    main()
