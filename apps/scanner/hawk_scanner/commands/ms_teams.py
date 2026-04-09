"""
Microsoft Teams connector for ARC-HAWK-DD scanner.
Uses the Microsoft Graph API via requests (no extra SDK required).
Scans channel messages and chat messages for PII patterns.

Auth: app-only (client credentials) or delegated (user token).
Required Graph scopes (app-only):
  - ChannelMessage.Read.All
  - Chat.Read.All
  - Team.ReadBasic.All
"""

import requests

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()

GRAPH_BASE = "https://graph.microsoft.com/v1.0"


def get_access_token(tenant_id, client_id, client_secret):
    url = f"https://login.microsoftonline.com/{tenant_id}/oauth2/v2.0/token"
    data = {
        "grant_type": "client_credentials",
        "client_id": client_id,
        "client_secret": client_secret,
        "scope": "https://graph.microsoft.com/.default",
    }
    resp = requests.post(url, data=data, timeout=10)
    resp.raise_for_status()
    return resp.json()["access_token"]


def graph_get(token, path, params=None):
    headers = {"Authorization": f"Bearer {token}"}
    resp = requests.get(f"{GRAPH_BASE}{path}", headers=headers, params=params, timeout=15)
    if resp.status_code == 200:
        return resp.json()
    return None


def _paginate(token, path, params=None):
    """Yield all items across @odata.nextLink pages."""
    params = params or {}
    data = graph_get(token, path, params)
    while data:
        for item in data.get('value', []):
            yield item
        next_link = data.get('@odata.nextLink')
        if not next_link:
            break
        # nextLink is a full URL
        headers = {"Authorization": f"Bearer {token}"}
        resp = requests.get(next_link, headers=headers, timeout=15)
        data = resp.json() if resp.status_code == 200 else None


def _scan_text(args, text, location, profile_name):
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
                        'host': 'ms_teams',
                        'file_path': location,
                        'pattern_name': match['pattern_name'],
                        'matches': match['matches'],
                        'sample_text': match['sample_text'],
                        'profile': profile_name,
                        'data_source': 'ms_teams',
                    })
    return results


def scan_team_channels(args, token, team_id, profile_name, max_messages_per_channel=500):
    results = []
    channels = list(_paginate(token, f"/teams/{team_id}/channels"))

    for channel in channels:
        channel_id = channel['id']
        channel_name = channel.get('displayName', channel_id)
        count = 0
        for msg in _paginate(token, f"/teams/{team_id}/channels/{channel_id}/messages"):
            if count >= max_messages_per_channel:
                break
            count += 1
            body = msg.get('body', {}).get('content', '')
            if body:
                # Strip HTML if content is HTML type
                if msg.get('body', {}).get('contentType') == 'html':
                    try:
                        from html.parser import HTMLParser
                        class _Stripper(HTMLParser):
                            def __init__(self):
                                super().__init__()
                                self.parts = []
                            def handle_data(self, d):
                                self.parts.append(d)
                        s = _Stripper()
                        s.feed(body)
                        body = ' '.join(s.parts)
                    except Exception:
                        pass
                results += _scan_text(
                    args, body,
                    f"ms_teams://{team_id}/{channel_name}",
                    profile_name,
                )

        system.print_info(args, f"  Channel '{channel_name}': {count} messages scanned")

    return results


def scan_chats(args, token, profile_name, max_messages_per_chat=200):
    """Scan all chats the app has access to (requires Chat.Read.All)."""
    results = []
    for chat in _paginate(token, "/chats", {"$filter": "chatType eq 'group' or chatType eq 'oneOnOne'"}):
        chat_id = chat['id']
        count = 0
        for msg in _paginate(token, f"/chats/{chat_id}/messages"):
            if count >= max_messages_per_chat:
                break
            count += 1
            body = msg.get('body', {}).get('content', '')
            if body:
                results += _scan_text(args, body, f"ms_teams://chat/{chat_id}", profile_name)

    return results


def execute(args):
    results = []
    system.print_info(args, "Running checks for Microsoft Teams sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    teams_config = sources_config.get('ms_teams')
    if not teams_config:
        system.print_error(args, "No MS Teams connection details found in connection.yml")
        return results

    for key, config in teams_config.items():
        tenant_id = config.get('tenant_id')
        client_id = config.get('client_id')
        client_secret = config.get('client_secret')
        team_ids = config.get('team_ids', [])
        scan_chats_flag = config.get('scan_chats', False)
        max_msg = int(config.get('max_messages_per_channel', 500))

        if not all([tenant_id, client_id, client_secret]):
            system.print_error(args, f"Incomplete MS Teams config for key: {key}")
            continue

        try:
            token = get_access_token(tenant_id, client_id, client_secret)
            system.print_info(args, f"Authenticated with Microsoft Graph (tenant: {tenant_id})")
        except Exception as e:
            system.print_error(args, f"MS Teams auth failed for key {key}: {e}")
            continue

        for team_id in team_ids:
            system.print_info(args, f"Scanning MS Teams team: {team_id}")
            results += scan_team_channels(args, token, team_id, key, max_msg)

        if scan_chats_flag:
            system.print_info(args, "Scanning MS Teams chats")
            results += scan_chats(args, token, key)

    return results
