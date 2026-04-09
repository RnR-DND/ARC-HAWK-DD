"""
Email (EML/MSG) connector for ARC-HAWK-DD scanner.
EML: stdlib `email` module.
MSG: `extract-msg` library (optional, falls back gracefully).
Scans headers, subject, body, and text attachments.
"""

import email
import email.policy
import os
from pathlib import Path

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()


def _scan_text(args, text, file_path, location, profile_name, source_key):
    results = []
    for i, line in enumerate(text.splitlines()):
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
                        'location': location,
                        'line_number': i + 1,
                        'pattern_name': match['pattern_name'],
                        'matches': match['matches'],
                        'sample_text': match['sample_text'],
                        'profile': profile_name,
                        'data_source': source_key,
                    })
    return results


def scan_eml_file(args, file_path, profile_name):
    results = []
    try:
        with open(file_path, 'rb') as f:
            msg = email.message_from_binary_file(f, policy=email.policy.default)

        # Headers
        for header in ['From', 'To', 'Cc', 'Subject', 'Reply-To']:
            value = msg.get(header, '')
            if value:
                results += _scan_text(args, value, file_path, f'header:{header}', profile_name, 'eml')

        # Body + text attachments
        for part in msg.walk():
            content_type = part.get_content_type()
            disposition = str(part.get('Content-Disposition', ''))

            if content_type in ('text/plain', 'text/html') and 'attachment' not in disposition:
                try:
                    body = part.get_content()
                    if body:
                        results += _scan_text(args, str(body), file_path, content_type, profile_name, 'eml')
                except Exception:
                    pass
            elif 'attachment' in disposition and content_type == 'text/plain':
                try:
                    body = part.get_payload(decode=True).decode('utf-8', errors='replace')
                    results += _scan_text(args, body, file_path, 'attachment', profile_name, 'eml')
                except Exception:
                    pass

    except Exception as e:
        system.print_error(args, f"Error reading EML {file_path}: {e}")

    return results


def scan_msg_file(args, file_path, profile_name):
    try:
        import extract_msg
    except ImportError:
        system.print_error(args, "extract-msg not installed. Run: pip install extract-msg")
        return []

    results = []
    try:
        msg = extract_msg.openMsg(file_path)
        for field, value in [('From', msg.sender), ('To', msg.to), ('Subject', msg.subject)]:
            if value:
                results += _scan_text(args, value, file_path, f'header:{field}', profile_name, 'msg')
        if msg.body:
            results += _scan_text(args, msg.body, file_path, 'body', profile_name, 'msg')
        msg.close()
    except Exception as e:
        system.print_error(args, f"Error reading MSG {file_path}: {e}")

    return results


def execute(args):
    results = []
    system.print_info(args, "Running checks for Email (EML/MSG) sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    email_config = sources_config.get('email')
    if not email_config:
        system.print_error(args, "No email file connection details found in connection.yml")
        return results

    for key, config in email_config.items():
        paths = config.get('paths', [])
        recursive = config.get('recursive', True)

        for base_path in paths:
            if not os.path.exists(base_path):
                system.print_error(args, f"Path does not exist: {base_path}")
                continue

            search_dirs = [base_path] if os.path.isfile(base_path) else [base_path]
            for base in search_dirs:
                if os.path.isfile(base):
                    ext = Path(base).suffix.lower()
                    if ext == '.eml':
                        results += scan_eml_file(args, base, key)
                    elif ext == '.msg':
                        results += scan_msg_file(args, base, key)
                    continue

                glob = '**/*' if recursive else '*'
                for f in Path(base).glob(glob):
                    if f.suffix.lower() == '.eml':
                        results += scan_eml_file(args, str(f), key)
                    elif f.suffix.lower() == '.msg':
                        results += scan_msg_file(args, str(f), key)

    return results
