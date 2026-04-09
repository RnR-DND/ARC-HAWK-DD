"""
HubSpot connector for ARC-HAWK-DD scanner.
Requires: hubspot-api-client
Scans Contacts, Companies, and Deals for PII patterns.
"""

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()

# HubSpot object types and their common PII-bearing properties
OBJECT_CONFIGS = {
    'contacts': ['email', 'firstname', 'lastname', 'phone', 'mobilephone',
                 'address', 'city', 'state', 'zip', 'country', 'company'],
    'companies': ['name', 'phone', 'address', 'city', 'state', 'zip', 'country', 'website'],
    'deals': ['dealname', 'description'],
}


def connect_hubspot(args, access_token):
    try:
        from hubspot import HubSpot
        client = HubSpot(access_token=access_token)
        system.print_info(args, "Connected to HubSpot")
        return client
    except ImportError:
        system.print_error(args, "hubspot-api-client not installed. Run: pip install hubspot-api-client")
        return None
    except Exception as e:
        system.print_error(args, f"HubSpot connection failed: {e}")
        return None


def scan_object_type(args, client, object_type, properties, profile_name, max_records=1000):
    results = []
    after = None
    fetched = 0

    while fetched < max_records:
        try:
            if object_type == 'contacts':
                resp = client.crm.contacts.basic_api.get_page(
                    limit=min(100, max_records - fetched),
                    after=after,
                    properties=properties,
                )
            elif object_type == 'companies':
                resp = client.crm.companies.basic_api.get_page(
                    limit=min(100, max_records - fetched),
                    after=after,
                    properties=properties,
                )
            elif object_type == 'deals':
                resp = client.crm.deals.basic_api.get_page(
                    limit=min(100, max_records - fetched),
                    after=after,
                    properties=properties,
                )
            else:
                break
        except Exception as e:
            system.print_error(args, f"Error fetching HubSpot {object_type}: {e}")
            break

        for record in resp.results:
            fetched += 1
            props = record.properties or {}
            for prop_name in properties:
                value = props.get(prop_name)
                if not value or not isinstance(value, str):
                    continue
                matches = system.match_strings(args, value)
                if matches:
                    validated = validate_findings(matches, args)
                    if validated:
                        for match in validated:
                            results.append({
                                'host': 'hubspot',
                                'file_path': f"hubspot://{object_type}/{record.id}",
                                'table': object_type,
                                'column': prop_name,
                                'pattern_name': match['pattern_name'],
                                'matches': match['matches'],
                                'sample_text': match['sample_text'],
                                'profile': profile_name,
                                'data_source': 'hubspot',
                            })

        paging = getattr(resp, 'paging', None)
        if paging and hasattr(paging, 'next') and paging.next:
            after = paging.next.after
        else:
            break

    return results


def execute(args):
    results = []
    system.print_info(args, "Running checks for HubSpot sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    hs_config = sources_config.get('hubspot')
    if not hs_config:
        system.print_error(args, "No HubSpot connection details found in connection.yml")
        return results

    for key, config in hs_config.items():
        access_token = config.get('access_token')
        objects = config.get('objects', list(OBJECT_CONFIGS.keys()))
        max_records = int(config.get('max_records', 1000))

        if not access_token:
            system.print_error(args, f"Missing access_token for HubSpot key: {key}")
            continue

        client = connect_hubspot(args, access_token)
        if not client:
            continue

        for obj_type in objects:
            props = OBJECT_CONFIGS.get(obj_type, [])
            if not props:
                system.print_error(args, f"Unknown HubSpot object type: {obj_type}")
                continue
            system.print_info(args, f"Scanning HubSpot {obj_type}")
            results += scan_object_type(args, client, obj_type, props, key, max_records)

    return results
