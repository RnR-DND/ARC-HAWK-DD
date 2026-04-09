"""
HTML file connector for ARC-HAWK-DD scanner.
Requires: beautifulsoup4
Extracts visible text and scans for PII patterns.
"""

import os
from pathlib import Path

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()


def scan_html_file(args, file_path, profile_name):
    try:
        from bs4 import BeautifulSoup
    except ImportError:
        system.print_error(args, "beautifulsoup4 not installed. Run: pip install beautifulsoup4")
        return []

    results = []
    try:
        with open(file_path, 'r', encoding='utf-8', errors='replace') as f:
            content = f.read()

        soup = BeautifulSoup(content, 'html.parser')

        # Remove script and style elements — not user-visible
        for tag in soup(['script', 'style']):
            tag.decompose()

        # Extract visible text, split by whitespace runs
        texts = soup.get_text(separator='\n').splitlines()
        for i, line in enumerate(texts):
            line = line.strip()
            if not line:
                continue
            matches = system.match_strings(args, line)
            if matches:
                validated = validate_findings(matches, args)
                if validated:
                    for match in validated:
                        results.append({
                            'host': str(Path(file_path).parent),
                            'file_path': file_path,
                            'line_number': i + 1,
                            'pattern_name': match['pattern_name'],
                            'matches': match['matches'],
                            'sample_text': match['sample_text'],
                            'profile': profile_name,
                            'data_source': 'html',
                        })
    except Exception as e:
        system.print_error(args, f"Error reading HTML {file_path}: {e}")

    return results


def execute(args):
    results = []
    system.print_info(args, "Running checks for HTML file sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    html_config = sources_config.get('html')
    if not html_config:
        system.print_error(args, "No HTML file connection details found in connection.yml")
        return results

    for key, config in html_config.items():
        paths = config.get('paths', [])
        recursive = config.get('recursive', True)

        for base_path in paths:
            if not os.path.exists(base_path):
                system.print_error(args, f"Path does not exist: {base_path}")
                continue

            if os.path.isfile(base_path):
                results += scan_html_file(args, base_path, key)
            elif os.path.isdir(base_path):
                pattern = '**/*.htm*' if recursive else '*.htm*'
                for html_file in Path(base_path).glob(pattern):
                    results += scan_html_file(args, str(html_file), key)

    return results
