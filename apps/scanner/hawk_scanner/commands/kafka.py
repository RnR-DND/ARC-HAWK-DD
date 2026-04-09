"""
Kafka streaming connector for ARC-HAWK-DD scanner.
Requires: confluent-kafka

Unlike batch connectors, Kafka scanning is time-bounded: the scanner
consumes messages for `scan_window_seconds` (default 300) from the
specified topics, then emits findings. This is NOT a persistent consumer
— it's a snapshot scan for PII discovery.

For continuous streaming monitoring, use the Temporal streaming supervisor
workflow (streaming_supervisor_workflow.go) which wraps this module.
"""

import json
import time

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()

DEFAULT_SCAN_WINDOW_SECONDS = 300  # 5 minutes
DEFAULT_MAX_MESSAGES = 10_000      # safety cap per topic


def scan_topic(args, consumer, topic, profile_name, scan_window_seconds, max_messages):
    from confluent_kafka import KafkaException

    results = []
    message_count = 0
    deadline = time.monotonic() + scan_window_seconds

    consumer.subscribe([topic])
    system.print_info(args, f"Subscribed to Kafka topic: {topic} (window={scan_window_seconds}s, max={max_messages})")

    try:
        while time.monotonic() < deadline and message_count < max_messages:
            msg = consumer.poll(timeout=1.0)
            if msg is None:
                continue
            if msg.error():
                system.print_error(args, f"Kafka message error: {msg.error()}")
                continue

            message_count += 1
            raw = msg.value()
            if raw is None:
                continue

            # Try JSON decode, fall back to raw string
            try:
                payload = json.loads(raw.decode('utf-8', errors='replace'))
                texts = _extract_strings(payload)
            except (json.JSONDecodeError, UnicodeDecodeError):
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
                                'host': topic,
                                'file_path': f"kafka://{topic}",
                                'offset': msg.offset(),
                                'partition': msg.partition(),
                                'pattern_name': match['pattern_name'],
                                'matches': match['matches'],
                                'sample_text': match['sample_text'],
                                'profile': profile_name,
                                'data_source': 'kafka',
                            })
    except KafkaException as e:
        system.print_error(args, f"Kafka error on topic {topic}: {e}")
    finally:
        consumer.unsubscribe()

    system.print_success(args, f"Scanned {message_count:,} messages from {topic}")
    return results


def _extract_strings(obj, depth=0):
    """Recursively extract string leaf values from a JSON structure (max depth 5)."""
    if depth > 5:
        return []
    if isinstance(obj, str):
        return [obj]
    if isinstance(obj, dict):
        result = []
        for v in obj.values():
            result.extend(_extract_strings(v, depth + 1))
        return result
    if isinstance(obj, list):
        result = []
        for item in obj:
            result.extend(_extract_strings(item, depth + 1))
        return result
    return [str(obj)] if obj is not None else []


def execute(args):
    results = []
    system.print_info(args, "Running checks for Kafka sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    kafka_config = sources_config.get('kafka')
    if not kafka_config:
        system.print_error(args, "No Kafka connection details found in connection.yml")
        return results

    try:
        from confluent_kafka import Consumer
    except ImportError:
        system.print_error(args, "confluent-kafka not installed. Run: pip install confluent-kafka")
        return results

    for key, config in kafka_config.items():
        bootstrap_servers = config.get('bootstrap_servers', 'localhost:9092')
        topics = config.get('topics', [])
        group_id = config.get('group_id', f'arc-hawk-scanner-{key}')
        security_protocol = config.get('security_protocol', 'PLAINTEXT')
        sasl_mechanism = config.get('sasl_mechanism')
        sasl_username = config.get('sasl_username')
        sasl_password = config.get('sasl_password')
        scan_window_seconds = int(config.get('scan_window_seconds', DEFAULT_SCAN_WINDOW_SECONDS))
        max_messages = int(config.get('max_messages', DEFAULT_MAX_MESSAGES))

        consumer_conf = {
            'bootstrap.servers': bootstrap_servers,
            'group.id': group_id,
            'auto.offset.reset': 'earliest',
            'security.protocol': security_protocol,
            'enable.auto.commit': False,
        }
        if sasl_mechanism:
            consumer_conf['sasl.mechanisms'] = sasl_mechanism
        if sasl_username:
            consumer_conf['sasl.username'] = sasl_username
        if sasl_password:
            consumer_conf['sasl.password'] = sasl_password

        try:
            consumer = Consumer(consumer_conf)
        except Exception as e:
            system.print_error(args, f"Failed to create Kafka consumer for key {key}: {e}")
            continue

        for topic in topics:
            results += scan_topic(args, consumer, topic, key, scan_window_seconds, max_messages)

        consumer.close()

    return results
