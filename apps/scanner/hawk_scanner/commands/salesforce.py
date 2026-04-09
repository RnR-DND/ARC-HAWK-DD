"""
Salesforce connector for ARC-HAWK-DD scanner.
Requires: simple-salesforce
Scans standard and custom objects for PII patterns using SOQL.
"""

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()

# Standard Salesforce objects that commonly contain PII
DEFAULT_OBJECTS = [
    'Account', 'Contact', 'Lead', 'Opportunity', 'Case',
    'User', 'CampaignMember', 'Contract',
]


def connect_salesforce(args, username, password, security_token,
                       domain='login', consumer_key=None, consumer_secret=None):
    try:
        from simple_salesforce import Salesforce
        if consumer_key and consumer_secret:
            sf = Salesforce(
                username=username,
                password=password,
                security_token=security_token,
                consumer_key=consumer_key,
                consumer_secret=consumer_secret,
                domain=domain,
            )
        else:
            sf = Salesforce(
                username=username,
                password=password,
                security_token=security_token,
                domain=domain,
            )
        system.print_info(args, f"Connected to Salesforce ({domain})")
        return sf
    except ImportError:
        system.print_error(args, "simple-salesforce not installed. Run: pip install simple-salesforce")
        return None
    except Exception as e:
        system.print_error(args, f"Salesforce connection failed: {e}")
        return None


def scan_object(args, sf, obj_name, profile_name, limit_rows=100):
    from simple_salesforce import SalesforceMalformedRequest

    results = []
    try:
        # Get field names
        desc = getattr(sf, obj_name).describe()
        text_types = {'string', 'textarea', 'email', 'phone', 'url', 'picklist', 'id'}
        fields = [f['name'] for f in desc['fields'] if f['type'] in text_types]

        if not fields:
            return results

        soql = f"SELECT {', '.join(fields)} FROM {obj_name} LIMIT {limit_rows}"
        records = sf.query(soql)

        for record in records.get('records', []):
            for field in fields:
                value = record.get(field)
                if not value or not isinstance(value, str):
                    continue
                matches = system.match_strings(args, value)
                if matches:
                    validated = validate_findings(matches, args)
                    if validated:
                        for match in validated:
                            results.append({
                                'host': 'salesforce',
                                'table': obj_name,
                                'column': field,
                                'pattern_name': match['pattern_name'],
                                'matches': match['matches'],
                                'sample_text': match['sample_text'],
                                'profile': profile_name,
                                'data_source': 'salesforce',
                            })
    except SalesforceMalformedRequest as e:
        system.print_error(args, f"SOQL error for {obj_name}: {e}")
    except Exception as e:
        system.print_error(args, f"Error scanning Salesforce object {obj_name}: {e}")

    return results


def execute(args):
    results = []
    system.print_info(args, "Running checks for Salesforce sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    sf_config = sources_config.get('salesforce')
    if not sf_config:
        system.print_error(args, "No Salesforce connection details found in connection.yml")
        return results

    for key, config in sf_config.items():
        username = config.get('username')
        password = config.get('password')
        security_token = config.get('security_token', '')
        domain = config.get('domain', 'login')
        consumer_key = config.get('consumer_key')
        consumer_secret = config.get('consumer_secret')
        objects = config.get('objects', DEFAULT_OBJECTS)
        limit_rows = int(config.get('limit_rows', 100))

        if not all([username, password]):
            system.print_error(args, f"Incomplete Salesforce config for key: {key}")
            continue

        sf = connect_salesforce(
            args, username, password, security_token, domain,
            consumer_key, consumer_secret,
        )
        if not sf:
            continue

        for obj in objects:
            system.print_info(args, f"Scanning Salesforce object: {obj}")
            results += scan_object(args, sf, obj, key, limit_rows)

    return results
