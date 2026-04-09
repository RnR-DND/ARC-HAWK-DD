"""
Jira connector for ARC-HAWK-DD scanner.
Requires: jira
Scans issue summaries, descriptions, and comments for PII patterns.
"""

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()


def connect_jira(args, server, email, api_token):
    try:
        from jira import JIRA
        client = JIRA(server=server, basic_auth=(email, api_token))
        system.print_info(args, f"Connected to Jira at {server}")
        return client
    except ImportError:
        system.print_error(args, "jira not installed. Run: pip install jira")
        return None
    except Exception as e:
        system.print_error(args, f"Jira connection failed: {e}")
        return None


def _scan_text(args, text, issue_key, location, profile_name):
    results = []
    if not text:
        return results
    for line in text.splitlines():
        line = line.strip()
        if not line:
            continue
        matches = system.match_strings(args, line)
        if matches:
            validated = validate_findings(matches, args)
            if validated:
                for match in validated:
                    results.append({
                        'host': 'jira',
                        'file_path': f"jira://{issue_key}",
                        'location': location,
                        'pattern_name': match['pattern_name'],
                        'matches': match['matches'],
                        'sample_text': match['sample_text'],
                        'profile': profile_name,
                        'data_source': 'jira',
                    })
    return results


def scan_project(args, client, project_key, profile_name, max_issues=500, include_comments=True):
    results = []
    start = 0
    batch = 50

    while start < max_issues:
        try:
            issues = client.search_issues(
                f"project={project_key}",
                startAt=start,
                maxResults=min(batch, max_issues - start),
                fields=['summary', 'description', 'comment'],
            )
        except Exception as e:
            system.print_error(args, f"Error fetching Jira issues for {project_key}: {e}")
            break

        if not issues:
            break

        for issue in issues:
            key = issue.key
            results += _scan_text(args, issue.fields.summary, key, 'summary', profile_name)
            results += _scan_text(args, issue.fields.description or '', key, 'description', profile_name)

            if include_comments:
                try:
                    for comment in client.comments(issue):
                        results += _scan_text(args, comment.body, key, f'comment:{comment.id}', profile_name)
                except Exception:
                    pass

        start += len(issues)
        if len(issues) < batch:
            break

    return results


def execute(args):
    results = []
    system.print_info(args, "Running checks for Jira sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    jira_config = sources_config.get('jira')
    if not jira_config:
        system.print_error(args, "No Jira connection details found in connection.yml")
        return results

    for key, config in jira_config.items():
        server = config.get('server')
        email = config.get('email')
        api_token = config.get('api_token')
        projects = config.get('projects', [])
        max_issues = int(config.get('max_issues', 500))
        include_comments = config.get('include_comments', True)

        if not all([server, email, api_token]):
            system.print_error(args, f"Incomplete Jira config for key: {key}")
            continue

        client = connect_jira(args, server, email, api_token)
        if not client:
            continue

        for project in projects:
            system.print_info(args, f"Scanning Jira project: {project}")
            results += scan_project(args, client, project, key, max_issues, include_comments)

    return results
