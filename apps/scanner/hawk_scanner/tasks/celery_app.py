"""
Celery Application — ARC-HAWK-DD Scanner Task Queues
======================================================
Defines the Celery app with 7 dedicated queues and task stubs for each queue.

Queue routing:
  scan_default      — standard on-demand data-source scans
  classify          — PII classification pipeline tasks
  ingest_streaming  — streaming-source ingestion (Kafka, Kinesis)
  ingest_cloud      — cloud-source ingestion (S3, GCS, Azure Blob)
  ingest_agent      — agent-pushed data ingestion
  remediation       — remediation action tasks
  escalation        — SLA-breach escalation and alerting

Usage:
    # Start a worker that consumes all queues:
    celery -A hawk_scanner.tasks.celery_app worker --queues=scan_default,classify,...

    # Start a dedicated classify worker:
    celery -A hawk_scanner.tasks.celery_app worker --queues=classify --concurrency=4

Environment variables:
    REDIS_URL     — Celery broker + result backend URL
                    Default: redis://localhost:6379/0
    CELERY_TASK_ALWAYS_EAGER
                  — Set to '1' in tests to run tasks synchronously
"""

import os
import logging
from celery import Celery
from celery.utils.log import get_task_logger

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# App configuration
# ---------------------------------------------------------------------------

REDIS_URL = os.getenv('REDIS_URL', 'redis://localhost:6379/0')

app = Celery(
    'hawk_scanner',
    broker=REDIS_URL,
    backend=REDIS_URL,
)

# ---------------------------------------------------------------------------
# Celery settings
# ---------------------------------------------------------------------------

app.conf.update(
    # Serialisation
    task_serializer='json',
    result_serializer='json',
    accept_content=['json'],

    # Timezone
    timezone='UTC',
    enable_utc=True,

    # Reliability — ack tasks only after completion so they are retried on
    # worker crash, not lost
    task_acks_late=True,
    task_reject_on_worker_lost=True,

    # Result TTL: keep results 24 h
    result_expires=86400,

    # Soft time limit 10 min; hard kill at 15 min
    task_soft_time_limit=600,
    task_time_limit=900,

    # Allow tasks to run synchronously in tests without a broker
    task_always_eager=os.getenv('CELERY_TASK_ALWAYS_EAGER', '0') == '1',

    # Beat schedule placeholder (populated by individual modules if needed)
    beat_schedule={},
)

# ---------------------------------------------------------------------------
# Queue routing
# ---------------------------------------------------------------------------

app.conf.task_routes = {
    'tasks.scan.*':              {'queue': 'scan_default'},
    'tasks.classify.*':          {'queue': 'classify'},
    'tasks.ingest.streaming.*':  {'queue': 'ingest_streaming'},
    'tasks.ingest.cloud.*':      {'queue': 'ingest_cloud'},
    'tasks.ingest.agent.*':      {'queue': 'ingest_agent'},
    'tasks.remediation.*':       {'queue': 'remediation'},
    'tasks.escalation.*':        {'queue': 'escalation'},
}

# Declare all queues so workers create them automatically on startup
app.conf.task_queues = None  # let Celery auto-create from routing

task_logger = get_task_logger(__name__)


# ===========================================================================
# Queue 1: scan_default — standard on-demand scans
# ===========================================================================

@app.task(
    name='tasks.scan.trigger',
    bind=True,
    max_retries=3,
    default_retry_delay=30,
    queue='scan_default',
)
def scan_trigger(self, source_type: str, connection_config: dict, scan_run_id: str):
    """
    Trigger a scan for a single data source.

    Args:
        source_type: One of the SUPPORTED_COMMANDS (e.g. 'postgresql', 'oracle').
        connection_config: Connection parameters dict for this source.
        scan_run_id: UUID of the ScanRun record in the backend DB.

    Returns:
        dict with 'findings_count' and 'scan_run_id'.
    """
    task_logger.info(
        "scan_trigger: source=%s scan_run_id=%s", source_type, scan_run_id
    )
    try:
        module = __import__(
            f"hawk_scanner.commands.{source_type}",
            fromlist=[source_type],
        )

        class _FakeArgs:
            """Minimal args namespace so connectors can call system helpers."""
            quiet = False
            debug = False
            no_write = True
            connection = None
            connection_json = None
            fingerprint = None
            stdout = False

        fake_args = _FakeArgs()
        # Inject the connection config so system.get_connection() picks it up
        import json as _json
        fake_args.connection_json = _json.dumps({'sources': {source_type: connection_config}})

        results = module.execute(fake_args)
        findings_count = len(results) if results else 0

        task_logger.info(
            "scan_trigger: DONE source=%s findings=%d scan_run_id=%s",
            source_type, findings_count, scan_run_id,
        )
        return {'findings_count': findings_count, 'scan_run_id': scan_run_id}

    except Exception as exc:
        task_logger.error("scan_trigger failed: %s", exc, exc_info=True)
        raise self.retry(exc=exc)


@app.task(
    name='tasks.scan.all_sources',
    bind=True,
    max_retries=1,
    queue='scan_default',
)
def scan_all_sources(self, connection_path: str, scan_run_id: str, parallel: bool = False):
    """
    Fan-out scan across all configured data sources (wraps all.py execute).

    Args:
        connection_path: Path to connection.yml on the worker filesystem.
        scan_run_id: UUID of the ScanRun record.
        parallel: If True, use parallel execution mode.

    Returns:
        dict with 'total_findings' and 'scan_run_id'.
    """
    task_logger.info(
        "scan_all_sources: connection_path=%s parallel=%s scan_run_id=%s",
        connection_path, parallel, scan_run_id,
    )
    try:
        from hawk_scanner.commands import all as all_cmd
        from hawk_scanner.internals import system

        class _FakeArgs:
            quiet = False
            debug = False
            no_write = True
            connection = connection_path
            connection_json = None
            fingerprint = None
            stdout = False

        fake_args = _FakeArgs()
        if parallel:
            results = all_cmd.execute_parallel(fake_args)
        else:
            results = all_cmd.execute_sequential(fake_args)

        total = len(results) if results else 0
        task_logger.info(
            "scan_all_sources: DONE total_findings=%d scan_run_id=%s", total, scan_run_id
        )
        return {'total_findings': total, 'scan_run_id': scan_run_id}

    except Exception as exc:
        task_logger.error("scan_all_sources failed: %s", exc, exc_info=True)
        raise self.retry(exc=exc)


# ===========================================================================
# Queue 2: classify — PII classification pipeline
# ===========================================================================

@app.task(
    name='tasks.classify.field',
    bind=True,
    max_retries=3,
    default_retry_delay=10,
    queue='classify',
)
def classify_field(self, field_value: str, field_name: str, source: str, context: dict = None):
    """
    Classify a single field value for PII.

    Args:
        field_value: Raw string value to classify.
        field_name:  Column or field name (used as context hint).
        source:      Data source identifier.
        context:     Optional extra metadata passed through to results.

    Returns:
        dict with 'pii_types', 'confidence', and passed-through 'context'.
    """
    task_logger.debug("classify_field: source=%s field=%s", source, field_name)
    try:
        from sdk.engine import SharedAnalyzerEngine
        engine = SharedAnalyzerEngine.get_engine()
        results = engine.analyze(text=field_value, language='en')
        pii_types = [
            {'entity': r.entity_type, 'score': round(r.score, 3)}
            for r in results
        ]
        return {
            'pii_types': pii_types,
            'field_name': field_name,
            'source': source,
            'context': context or {},
        }
    except Exception as exc:
        task_logger.error("classify_field failed: %s", exc, exc_info=True)
        raise self.retry(exc=exc)


@app.task(
    name='tasks.classify.batch',
    bind=True,
    max_retries=2,
    default_retry_delay=15,
    queue='classify',
)
def classify_batch(self, records: list, source: str):
    """
    Classify a batch of records.

    Args:
        records: List of dicts, each with 'field_name' and 'field_value'.
        source:  Data source identifier.

    Returns:
        List of classification result dicts (same order as input).
    """
    task_logger.info("classify_batch: source=%s records=%d", source, len(records))
    results = []
    try:
        from sdk.engine import SharedAnalyzerEngine
        engine = SharedAnalyzerEngine.get_engine()
        for rec in records:
            field_value = str(rec.get('field_value', '') or '')
            if not field_value:
                results.append({'field_name': rec.get('field_name'), 'pii_types': []})
                continue
            hits = engine.analyze(text=field_value, language='en')
            results.append({
                'field_name': rec.get('field_name'),
                'pii_types': [
                    {'entity': h.entity_type, 'score': round(h.score, 3)}
                    for h in hits
                ],
            })
        return results
    except Exception as exc:
        task_logger.error("classify_batch failed: %s", exc, exc_info=True)
        raise self.retry(exc=exc)


# ===========================================================================
# Queue 3: ingest_streaming — streaming source ingestion (Kafka, Kinesis)
# ===========================================================================

@app.task(
    name='tasks.ingest.streaming.kafka_snapshot',
    bind=True,
    max_retries=3,
    default_retry_delay=60,
    queue='ingest_streaming',
)
def ingest_kafka_snapshot(self, topic: str, bootstrap_servers: str,
                           group_id: str, max_messages: int,
                           scan_run_id: str):
    """
    Take a point-in-time snapshot of a Kafka topic and classify messages.

    Args:
        topic:             Kafka topic name.
        bootstrap_servers: Comma-separated Kafka brokers.
        group_id:          Consumer group ID.
        max_messages:      Maximum messages to consume in this snapshot.
        scan_run_id:       UUID of the ScanRun record.

    Returns:
        dict with 'messages_scanned', 'findings_count', 'scan_run_id'.
    """
    task_logger.info(
        "ingest_kafka_snapshot: topic=%s max_messages=%d scan_run_id=%s",
        topic, max_messages, scan_run_id,
    )
    try:
        from hawk_scanner.commands.kafka import execute as kafka_execute

        class _FakeArgs:
            quiet = False
            debug = False
            no_write = True
            connection = None
            connection_json = None
            fingerprint = None
            stdout = False

        import json as _json
        fake_args = _FakeArgs()
        fake_args.connection_json = _json.dumps({
            'sources': {
                'kafka': {
                    'snapshot_task': {
                        'bootstrap_servers': bootstrap_servers,
                        'topics': [topic],
                        'group_id': group_id,
                        'max_messages': max_messages,
                    }
                }
            }
        })
        results = kafka_execute(fake_args)
        return {
            'messages_scanned': max_messages,
            'findings_count': len(results) if results else 0,
            'scan_run_id': scan_run_id,
        }
    except Exception as exc:
        task_logger.error("ingest_kafka_snapshot failed: %s", exc, exc_info=True)
        raise self.retry(exc=exc)


@app.task(
    name='tasks.ingest.streaming.kinesis_snapshot',
    bind=True,
    max_retries=3,
    default_retry_delay=60,
    queue='ingest_streaming',
)
def ingest_kinesis_snapshot(self, stream_name: str, region: str,
                             shard_iterator_type: str,
                             max_records: int, scan_run_id: str):
    """
    Snapshot a Kinesis stream shard and classify records.

    Args:
        stream_name:          Kinesis stream name.
        region:               AWS region.
        shard_iterator_type:  e.g. 'LATEST' or 'TRIM_HORIZON'.
        max_records:          Maximum records per shard to consume.
        scan_run_id:          UUID of the ScanRun record.

    Returns:
        dict with 'records_scanned', 'findings_count', 'scan_run_id'.
    """
    task_logger.info(
        "ingest_kinesis_snapshot: stream=%s region=%s max_records=%d scan_run_id=%s",
        stream_name, region, max_records, scan_run_id,
    )
    try:
        from hawk_scanner.commands.kinesis import execute as kinesis_execute

        class _FakeArgs:
            quiet = False
            debug = False
            no_write = True
            connection = None
            connection_json = None
            fingerprint = None
            stdout = False

        import json as _json
        fake_args = _FakeArgs()
        fake_args.connection_json = _json.dumps({
            'sources': {
                'kinesis': {
                    'snapshot_task': {
                        'stream_name': stream_name,
                        'region': region,
                        'shard_iterator_type': shard_iterator_type,
                        'max_records': max_records,
                    }
                }
            }
        })
        results = kinesis_execute(fake_args)
        return {
            'records_scanned': max_records,
            'findings_count': len(results) if results else 0,
            'scan_run_id': scan_run_id,
        }
    except Exception as exc:
        task_logger.error("ingest_kinesis_snapshot failed: %s", exc, exc_info=True)
        raise self.retry(exc=exc)


# ===========================================================================
# Queue 4: ingest_cloud — cloud object store ingestion (S3, GCS, Azure Blob)
# ===========================================================================

@app.task(
    name='tasks.ingest.cloud.s3_scan',
    bind=True,
    max_retries=3,
    default_retry_delay=30,
    queue='ingest_cloud',
)
def ingest_s3_scan(self, bucket: str, prefix: str, aws_region: str,
                   scan_run_id: str, credentials: dict = None):
    """
    Scan objects in an S3 bucket prefix for PII.

    Args:
        bucket:      S3 bucket name.
        prefix:      Key prefix to restrict the scan (e.g. 'data/exports/').
        aws_region:  AWS region string.
        scan_run_id: UUID of the ScanRun record.
        credentials: Optional dict with 'access_key' and 'secret_key'.
                     Defaults to IAM role / environment credentials.

    Returns:
        dict with 'objects_scanned', 'findings_count', 'scan_run_id'.
    """
    task_logger.info(
        "ingest_s3_scan: bucket=%s prefix=%s region=%s scan_run_id=%s",
        bucket, prefix, aws_region, scan_run_id,
    )
    try:
        from hawk_scanner.commands.s3 import execute as s3_execute

        class _FakeArgs:
            quiet = False
            debug = False
            no_write = True
            connection = None
            connection_json = None
            fingerprint = None
            stdout = False

        import json as _json
        config = {
            'bucket': bucket,
            'prefix': prefix,
            'region': aws_region,
        }
        if credentials:
            config['access_key'] = credentials.get('access_key', '')
            config['secret_key'] = credentials.get('secret_key', '')

        fake_args = _FakeArgs()
        fake_args.connection_json = _json.dumps({'sources': {'s3': {'task': config}}})
        results = s3_execute(fake_args)
        return {
            'objects_scanned': 'unknown',
            'findings_count': len(results) if results else 0,
            'scan_run_id': scan_run_id,
        }
    except Exception as exc:
        task_logger.error("ingest_s3_scan failed: %s", exc, exc_info=True)
        raise self.retry(exc=exc)


@app.task(
    name='tasks.ingest.cloud.gcs_scan',
    bind=True,
    max_retries=3,
    default_retry_delay=30,
    queue='ingest_cloud',
)
def ingest_gcs_scan(self, bucket: str, prefix: str, project_id: str, scan_run_id: str):
    """
    Scan objects in a GCS bucket for PII.

    Args:
        bucket:      GCS bucket name.
        prefix:      Object prefix filter.
        project_id:  GCP project ID.
        scan_run_id: UUID of the ScanRun record.

    Returns:
        dict with 'findings_count' and 'scan_run_id'.
    """
    task_logger.info(
        "ingest_gcs_scan: bucket=%s prefix=%s project=%s scan_run_id=%s",
        bucket, prefix, project_id, scan_run_id,
    )
    try:
        from hawk_scanner.commands.gcs import execute as gcs_execute

        class _FakeArgs:
            quiet = False
            debug = False
            no_write = True
            connection = None
            connection_json = None
            fingerprint = None
            stdout = False

        import json as _json
        fake_args = _FakeArgs()
        fake_args.connection_json = _json.dumps({
            'sources': {
                'gcs': {
                    'task': {
                        'bucket': bucket,
                        'prefix': prefix,
                        'project_id': project_id,
                    }
                }
            }
        })
        results = gcs_execute(fake_args)
        return {
            'findings_count': len(results) if results else 0,
            'scan_run_id': scan_run_id,
        }
    except Exception as exc:
        task_logger.error("ingest_gcs_scan failed: %s", exc, exc_info=True)
        raise self.retry(exc=exc)


@app.task(
    name='tasks.ingest.cloud.azure_blob_scan',
    bind=True,
    max_retries=3,
    default_retry_delay=30,
    queue='ingest_cloud',
)
def ingest_azure_blob_scan(self, container: str, connection_string: str,
                            prefix: str, scan_run_id: str):
    """
    Scan blobs in an Azure Blob Storage container for PII.

    Args:
        container:          Azure container name.
        connection_string:  Azure Storage account connection string.
        prefix:             Blob prefix filter (virtual directory).
        scan_run_id:        UUID of the ScanRun record.

    Returns:
        dict with 'findings_count' and 'scan_run_id'.
    """
    task_logger.info(
        "ingest_azure_blob_scan: container=%s prefix=%s scan_run_id=%s",
        container, prefix, scan_run_id,
    )
    try:
        from hawk_scanner.commands.azure_blob import execute as azure_execute

        class _FakeArgs:
            quiet = False
            debug = False
            no_write = True
            connection = None
            connection_json = None
            fingerprint = None
            stdout = False

        import json as _json
        fake_args = _FakeArgs()
        fake_args.connection_json = _json.dumps({
            'sources': {
                'azure_blob': {
                    'task': {
                        'connection_string': connection_string,
                        'container': container,
                        'prefix': prefix,
                    }
                }
            }
        })
        results = azure_execute(fake_args)
        return {
            'findings_count': len(results) if results else 0,
            'scan_run_id': scan_run_id,
        }
    except Exception as exc:
        task_logger.error("ingest_azure_blob_scan failed: %s", exc, exc_info=True)
        raise self.retry(exc=exc)


# ===========================================================================
# Queue 5: ingest_agent — agent-pushed data ingestion
# ===========================================================================

@app.task(
    name='tasks.ingest.agent.push_findings',
    bind=True,
    max_retries=5,
    default_retry_delay=15,
    queue='ingest_agent',
)
def ingest_agent_push_findings(self, findings: list, agent_id: str,
                                scan_run_id: str, backend_url: str = None):
    """
    Accept a batch of findings pushed directly by a scanner agent and
    forward them to the backend API.

    Args:
        findings:    List of finding dicts from the agent.
        agent_id:    Identifier of the pushing agent.
        scan_run_id: UUID of the ScanRun record.
        backend_url: Backend API base URL. Defaults to BACKEND_URL env var.

    Returns:
        dict with 'ingested_count' and 'scan_run_id'.
    """
    import requests as _req

    url = backend_url or os.getenv('BACKEND_URL', 'http://backend:8080')
    ingest_endpoint = f"{url}/api/v1/scan/ingest"

    task_logger.info(
        "ingest_agent_push_findings: agent=%s findings=%d scan_run_id=%s",
        agent_id, len(findings), scan_run_id,
    )
    try:
        payload = {
            'scan_run_id': scan_run_id,
            'agent_id': agent_id,
            'findings': findings,
        }
        resp = _req.post(ingest_endpoint, json=payload, timeout=30)
        resp.raise_for_status()
        task_logger.info(
            "ingest_agent_push_findings: ingested %d findings for scan_run_id=%s",
            len(findings), scan_run_id,
        )
        return {'ingested_count': len(findings), 'scan_run_id': scan_run_id}
    except Exception as exc:
        task_logger.error("ingest_agent_push_findings failed: %s", exc, exc_info=True)
        raise self.retry(exc=exc)


@app.task(
    name='tasks.ingest.agent.heartbeat',
    queue='ingest_agent',
)
def agent_heartbeat(agent_id: str, status: str, metadata: dict = None):
    """
    Record an agent heartbeat (liveness signal).

    Args:
        agent_id:  Agent identifier.
        status:    'alive' | 'idle' | 'scanning' | 'error'.
        metadata:  Optional dict with extra agent state.

    Returns:
        dict with 'agent_id' and 'acknowledged'.
    """
    task_logger.debug("agent_heartbeat: agent=%s status=%s", agent_id, status)
    return {'agent_id': agent_id, 'acknowledged': True, 'status': status}


# ===========================================================================
# Queue 6: remediation — remediation action execution
# ===========================================================================

@app.task(
    name='tasks.remediation.apply_mask',
    bind=True,
    max_retries=3,
    default_retry_delay=30,
    queue='remediation',
)
def remediation_apply_mask(self, finding_id: str, strategy: str,
                            source_type: str, field_path: dict,
                            backend_url: str = None):
    """
    Apply a masking/anonymisation strategy to a specific finding's source field.

    Args:
        finding_id:  UUID of the Finding record.
        strategy:    Masking strategy name (e.g. 'hash', 'redact', 'tokenise').
        source_type: Data source type (e.g. 'postgresql', 's3').
        field_path:  Dict identifying the field: {'table', 'column', ...}.
        backend_url: Backend API URL.

    Returns:
        dict with 'finding_id', 'strategy', 'status'.
    """
    import requests as _req

    url = backend_url or os.getenv('BACKEND_URL', 'http://backend:8080')
    endpoint = f"{url}/api/v1/remediation/apply"

    task_logger.info(
        "remediation_apply_mask: finding=%s strategy=%s source=%s",
        finding_id, strategy, source_type,
    )
    try:
        resp = _req.post(endpoint, json={
            'finding_id': finding_id,
            'strategy': strategy,
            'source_type': source_type,
            'field_path': field_path,
        }, timeout=60)
        resp.raise_for_status()
        return {'finding_id': finding_id, 'strategy': strategy, 'status': 'APPLIED'}
    except Exception as exc:
        task_logger.error("remediation_apply_mask failed: %s", exc, exc_info=True)
        raise self.retry(exc=exc)


@app.task(
    name='tasks.remediation.notify_owner',
    bind=True,
    max_retries=3,
    default_retry_delay=60,
    queue='remediation',
)
def remediation_notify_owner(self, finding_id: str, owner_email: str,
                              pii_summary: dict, backend_url: str = None):
    """
    Notify a data owner that PII was found in their dataset.

    Args:
        finding_id:   UUID of the Finding record.
        owner_email:  Email address of the data owner.
        pii_summary:  Summary dict (pii_types, severity, table, column).
        backend_url:  Backend API URL.

    Returns:
        dict with 'finding_id', 'notified_email', 'status'.
    """
    import requests as _req

    url = backend_url or os.getenv('BACKEND_URL', 'http://backend:8080')
    endpoint = f"{url}/api/v1/remediation/notify"

    task_logger.info(
        "remediation_notify_owner: finding=%s owner=%s", finding_id, owner_email
    )
    try:
        resp = _req.post(endpoint, json={
            'finding_id': finding_id,
            'owner_email': owner_email,
            'pii_summary': pii_summary,
        }, timeout=30)
        resp.raise_for_status()
        return {
            'finding_id': finding_id,
            'notified_email': owner_email,
            'status': 'SENT',
        }
    except Exception as exc:
        task_logger.error("remediation_notify_owner failed: %s", exc, exc_info=True)
        raise self.retry(exc=exc)


# ===========================================================================
# Queue 7: escalation — SLA-breach escalation and alerting
# ===========================================================================

@app.task(
    name='tasks.escalation.sla_breach',
    bind=True,
    max_retries=2,
    default_retry_delay=120,
    queue='escalation',
)
def escalation_sla_breach(self, finding_id: str, sla_deadline: str,
                           severity: str, assignee: str,
                           backend_url: str = None):
    """
    Handle an SLA breach: escalate an unresolved finding to the next tier.

    Args:
        finding_id:   UUID of the Finding record.
        sla_deadline: ISO-8601 datetime string of the original SLA deadline.
        severity:     Finding severity ('Critical', 'High', 'Medium', 'Low').
        assignee:     Current assignee email.
        backend_url:  Backend API URL.

    Returns:
        dict with 'finding_id', 'escalated_to', 'status'.
    """
    import requests as _req

    url = backend_url or os.getenv('BACKEND_URL', 'http://backend:8080')
    endpoint = f"{url}/api/v1/escalation/sla-breach"

    task_logger.warning(
        "escalation_sla_breach: finding=%s severity=%s deadline=%s assignee=%s",
        finding_id, severity, sla_deadline, assignee,
    )
    try:
        resp = _req.post(endpoint, json={
            'finding_id': finding_id,
            'sla_deadline': sla_deadline,
            'severity': severity,
            'current_assignee': assignee,
        }, timeout=30)
        resp.raise_for_status()
        result = resp.json()
        return {
            'finding_id': finding_id,
            'escalated_to': result.get('escalated_to'),
            'status': 'ESCALATED',
        }
    except Exception as exc:
        task_logger.error("escalation_sla_breach failed: %s", exc, exc_info=True)
        raise self.retry(exc=exc)


@app.task(
    name='tasks.escalation.critical_alert',
    bind=True,
    max_retries=5,
    default_retry_delay=30,
    queue='escalation',
)
def escalation_critical_alert(self, finding_id: str, pii_type: str,
                               source: str, alert_channels: list,
                               backend_url: str = None):
    """
    Broadcast a critical PII discovery alert to one or more channels
    (Slack, email, PagerDuty, etc.) via the backend notification service.

    Args:
        finding_id:      UUID of the Finding record.
        pii_type:        Detected PII type (e.g. 'AADHAAR', 'CREDIT_CARD').
        source:          Data source where PII was found.
        alert_channels:  List of channel names to notify (e.g. ['slack', 'email']).
        backend_url:     Backend API URL.

    Returns:
        dict with 'finding_id', 'channels_notified', 'status'.
    """
    import requests as _req

    url = backend_url or os.getenv('BACKEND_URL', 'http://backend:8080')
    endpoint = f"{url}/api/v1/escalation/critical-alert"

    task_logger.warning(
        "escalation_critical_alert: finding=%s pii_type=%s source=%s channels=%s",
        finding_id, pii_type, source, alert_channels,
    )
    try:
        resp = _req.post(endpoint, json={
            'finding_id': finding_id,
            'pii_type': pii_type,
            'source': source,
            'alert_channels': alert_channels,
        }, timeout=30)
        resp.raise_for_status()
        return {
            'finding_id': finding_id,
            'channels_notified': alert_channels,
            'status': 'ALERTED',
        }
    except Exception as exc:
        task_logger.error("escalation_critical_alert failed: %s", exc, exc_info=True)
        raise self.retry(exc=exc)


# ---------------------------------------------------------------------------
# Auto-discover tasks in this module when the worker starts
# ---------------------------------------------------------------------------

app.autodiscover_tasks(lambda: ['hawk_scanner.tasks'])
