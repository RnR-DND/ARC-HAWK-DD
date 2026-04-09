"""
Amazon Kinesis Data Streams connector for ARC-HAWK-DD scanner.
Requires: boto3 (already in requirements)

Scans the tip of the stream (LATEST shard iterators) for up to
`scan_window_seconds` seconds, then returns findings.
"""

import json
import time

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()

DEFAULT_SCAN_WINDOW_SECONDS = 300
DEFAULT_MAX_RECORDS_PER_SHARD = 5_000


def _extract_strings(obj, depth=0):
    if depth > 5:
        return []
    if isinstance(obj, str):
        return [obj]
    if isinstance(obj, dict):
        out = []
        for v in obj.values():
            out.extend(_extract_strings(v, depth + 1))
        return out
    if isinstance(obj, list):
        out = []
        for item in obj:
            out.extend(_extract_strings(item, depth + 1))
        return out
    return [str(obj)] if obj is not None else []


def scan_stream(args, client, stream_name, profile_name,
                scan_window_seconds, max_records_per_shard):
    results = []
    deadline = time.monotonic() + scan_window_seconds

    try:
        resp = client.describe_stream(StreamName=stream_name)
        shards = resp['StreamDescription']['Shards']
    except Exception as e:
        system.print_error(args, f"Failed to describe Kinesis stream {stream_name}: {e}")
        return results

    system.print_info(args, f"Scanning Kinesis stream '{stream_name}' ({len(shards)} shards)")

    for shard in shards:
        shard_id = shard['ShardId']
        try:
            iter_resp = client.get_shard_iterator(
                StreamName=stream_name,
                ShardId=shard_id,
                ShardIteratorType='TRIM_HORIZON',  # From oldest available record
            )
            shard_iter = iter_resp['ShardIterator']
        except Exception as e:
            system.print_error(args, f"Failed to get shard iterator for {shard_id}: {e}")
            continue

        records_scanned = 0
        while shard_iter and time.monotonic() < deadline and records_scanned < max_records_per_shard:
            try:
                get_resp = client.get_records(ShardIterator=shard_iter, Limit=100)
            except Exception as e:
                system.print_error(args, f"get_records failed for {shard_id}: {e}")
                break

            records = get_resp.get('Records', [])
            shard_iter = get_resp.get('NextShardIterator')
            records_scanned += len(records)

            if not records:
                break  # End of shard (no more data)

            for record in records:
                raw = record.get('Data', b'')
                try:
                    payload = json.loads(raw.decode('utf-8', errors='replace'))
                    texts = _extract_strings(payload)
                except Exception:
                    texts = [raw.decode('utf-8', errors='replace')]

                for text in texts:
                    if not text:
                        continue
                    matches = system.match_strings(args, text)
                    if matches:
                        validated = validate_findings(matches, args)
                        if validated:
                            for match in validated:
                                results.append({
                                    'host': stream_name,
                                    'file_path': f"kinesis://{stream_name}/{shard_id}",
                                    'sequence_number': record.get('SequenceNumber', ''),
                                    'pattern_name': match['pattern_name'],
                                    'matches': match['matches'],
                                    'sample_text': match['sample_text'],
                                    'profile': profile_name,
                                    'data_source': 'kinesis',
                                })

        system.print_info(args, f"  Shard {shard_id}: {records_scanned} records")

    return results


def execute(args):
    results = []
    system.print_info(args, "Running checks for Kinesis Data Streams sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    kinesis_config = sources_config.get('kinesis')
    if not kinesis_config:
        system.print_error(args, "No Kinesis connection details found in connection.yml")
        return results

    try:
        import boto3
    except ImportError:
        system.print_error(args, "boto3 not installed. Run: pip install boto3")
        return results

    for key, config in kinesis_config.items():
        region = config.get('region', 'us-east-1')
        streams = config.get('streams', [])
        aws_access_key = config.get('aws_access_key_id')
        aws_secret_key = config.get('aws_secret_access_key')
        scan_window_seconds = int(config.get('scan_window_seconds', DEFAULT_SCAN_WINDOW_SECONDS))
        max_records = int(config.get('max_records_per_shard', DEFAULT_MAX_RECORDS_PER_SHARD))

        client_kwargs = {'region_name': region}
        if aws_access_key and aws_secret_key:
            client_kwargs['aws_access_key_id'] = aws_access_key
            client_kwargs['aws_secret_access_key'] = aws_secret_key

        try:
            client = boto3.client('kinesis', **client_kwargs)
        except Exception as e:
            system.print_error(args, f"Failed to create Kinesis client for key {key}: {e}")
            continue

        for stream_name in streams:
            results += scan_stream(
                args, client, stream_name, key,
                scan_window_seconds, max_records,
            )

    return results
