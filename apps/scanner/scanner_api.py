"""
Scanner HTTP API Service
Provides REST API for triggering scans and ingesting results into the backend.
"""
import uuid
import os
import json
import logging
import re
import subprocess
import threading
import requests
import redis as redis_lib
from flask import Flask, request, jsonify
from datetime import datetime

app = Flask(__name__)

# Use structured logging instead of print()
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(name)s: %(message)s'
)
logger = logging.getLogger('arc-hawk-scanner')

# ---------------------------------------------------------------------------
# Prometheus metrics (Phase 13)
# ---------------------------------------------------------------------------
try:
    from prometheus_client import Counter, Histogram, Gauge, generate_latest, CONTENT_TYPE_LATEST
    _PROM_AVAILABLE = True

    scan_jobs_total = Counter(
        'arc_hawk_scan_jobs_total',
        'Total scan jobs triggered',
        ['status'],  # running | completed | failed
    )
    pii_fields_detected_total = Counter(
        'arc_hawk_pii_fields_detected_total',
        'Total PII fields detected across all scans',
        ['pii_type'],
    )
    classification_latency_ms = Histogram(
        'arc_hawk_classification_latency_ms',
        'Time spent in classification pipeline per scan (ms)',
        buckets=[100, 500, 1000, 5000, 15000, 60000, 300000],
    )
    llm_fallback_total = Counter(
        'arc_hawk_llm_fallback_total',
        'Findings that fell back from Layer 3 LLM to Layer 1/2',
    )
    active_scans_gauge = Gauge(
        'arc_hawk_active_scans',
        'Number of scans currently running',
    )
    ingest_chunk_errors_total = Counter(
        'arc_hawk_ingest_chunk_errors_total',
        'Failed chunk ingestion attempts',
    )
except ImportError:
    _PROM_AVAILABLE = False
    logger.warning("prometheus_client not installed — metrics endpoint disabled")


@app.route('/metrics', methods=['GET'])
def metrics():
    """Prometheus metrics scrape endpoint."""
    if not _PROM_AVAILABLE:
        return "prometheus_client not installed", 503
    from prometheus_client import generate_latest, CONTENT_TYPE_LATEST
    return generate_latest(), 200, {'Content-Type': CONTENT_TYPE_LATEST}

BACKEND_URL = os.getenv('BACKEND_URL', 'http://backend:8080')

# ---------------------------------------------------------------------------
# OpenTelemetry tracing (Phase 13) — no-op if not configured
# ---------------------------------------------------------------------------
try:
    from opentelemetry import trace
    from opentelemetry.sdk.trace import TracerProvider
    from opentelemetry.sdk.trace.export import BatchSpanProcessor, ConsoleSpanExporter
    _otel_provider = TracerProvider()
    _otel_exporter = ConsoleSpanExporter() if os.getenv("OTEL_CONSOLE_EXPORT") else None
    if _otel_exporter:
        _otel_provider.add_span_processor(BatchSpanProcessor(_otel_exporter))
    trace.set_tracer_provider(_otel_provider)
    _tracer = trace.get_tracer("arc-hawk-scanner")
    _OTEL_AVAILABLE = True
except ImportError:
    _OTEL_AVAILABLE = False
    _tracer = None

# ---------------------------------------------------------------------------
# Tenacity circuit breaker (Phase 13) — wraps backend HTTP calls
# ---------------------------------------------------------------------------
try:
    from tenacity import retry, stop_after_attempt, wait_exponential, retry_if_exception_type
    import requests as _requests_mod

    def _is_transient_error(exc: Exception) -> bool:
        """Only retry on connection/timeout errors, not 4xx HTTP errors."""
        return isinstance(exc, (
            _requests_mod.exceptions.ConnectionError,
            _requests_mod.exceptions.Timeout,
        ))

    _circuit_breaker_retry = retry(
        reraise=True,
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=1, max=10),
        retry=retry_if_exception_type((_requests_mod.exceptions.ConnectionError, _requests_mod.exceptions.Timeout)),
    )
    _TENACITY_AVAILABLE = True
except ImportError:
    _TENACITY_AVAILABLE = False
    _circuit_breaker_retry = lambda f: f  # no-op decorator

# ---------------------------------------------------------------------------
# Redis-backed scan state — replaces the in-process dict that was not
# thread-safe under gunicorn multi-worker (P0-3 fix).
# Keys are stored as "scan:{scan_id}" with a 24-hour TTL.
# Falls back to an in-process dict if Redis is unreachable (dev/test only).
# ---------------------------------------------------------------------------
_REDIS_URL = os.getenv('REDIS_URL', 'redis://redis:6379/0')
_SCAN_TTL_SECONDS = 86400  # 24 hours

try:
    _redis = redis_lib.from_url(_REDIS_URL, decode_responses=True, socket_connect_timeout=2)
    _redis.ping()
    logger.info(f"Scan state backend: Redis at {_REDIS_URL}")
    _redis_available = True
except Exception as _e:
    logger.warning(f"Redis unavailable ({_e}), falling back to in-process dict (not safe for multi-worker)")
    _redis = None
    _redis_available = False
    _fallback_scans: dict = {}


def _scan_key(scan_id: str) -> str:
    return f"scan:{scan_id}"


def scan_state_get(scan_id: str) -> dict | None:
    """Return the state dict for a scan, or None if not found."""
    if _redis_available:
        raw = _redis.get(_scan_key(scan_id))
        return json.loads(raw) if raw else None
    return _fallback_scans.get(scan_id)


def scan_state_set(scan_id: str, state: dict) -> None:
    """Persist scan state (full replacement)."""
    if _redis_available:
        _redis.setex(_scan_key(scan_id), _SCAN_TTL_SECONDS, json.dumps(state))
    else:
        _fallback_scans[scan_id] = state


def scan_state_update(scan_id: str, updates: dict) -> None:
    """Merge updates into existing scan state."""
    state = scan_state_get(scan_id) or {}
    state.update(updates)
    scan_state_set(scan_id, state)

@app.before_request
def log_request_info():
    """Log incoming request details for auditability."""
    app.logger.info(f"{request.method} {request.path} from {request.remote_addr}")

@app.route('/health', methods=['GET'])
def health():
    return jsonify({
        'status': 'healthy',
        'service': 'arc-hawk-scanner',
        'version': '0.3.39'
    })

@app.route('/scan', methods=['POST'])
def trigger_scan():
    """
    Trigger a new scan with the provided configuration.
    Expected body:
    {
        "scan_id": "uuid",
        "scan_name": "string",
        "sources": ["profile_name1", "profile_name2"],
        "pii_types": ["PAN", "AADHAAR", ...],
        "execution_mode": "parallel|sequential"
    }
    """
    try:
        config = request.get_json()
        scan_id = config.get('scan_id') or str(uuid.uuid4())
        
        # Mark scan as running (stored in Redis, safe for multi-worker)
        scan_state_set(scan_id, {
            'status': 'running',
            'started_at': datetime.now().isoformat(),
            'config': config,
        })
        if _PROM_AVAILABLE:
            scan_jobs_total.labels(status='running').inc()
            active_scans_gauge.inc()

        # Execute scan in background thread
        thread = threading.Thread(target=execute_scan, args=(scan_id, config))
        thread.start()
        
        return jsonify({
            'scan_id': scan_id,
            'status': 'running',
            'message': 'Scan started successfully'
        })
        
    except Exception as e:
        return jsonify({
            'error': str(e),
            'status': 'failed'
        }), 500

@app.route('/scan/<scan_id>/status', methods=['GET'])
def get_scan_status(scan_id):
    """Get status of a specific scan."""
    state = scan_state_get(scan_id)
    if state is not None:
        return jsonify(state)
    return jsonify({'status': 'not_found'}), 404


def _parse_scanner_diagnostics(stdout: str) -> dict:
    """Extract structured diagnostics from hawk_scanner stdout."""
    diag = {}
    # Match lines like: "✅ ✅ Scanned 0 rows across 0 tables"
    rows_match = re.search(r'Scanned\s+([\d,]+)\s+rows?\s+across\s+([\d,]+)\s+tables?', stdout)
    if rows_match:
        diag['rows_scanned'] = int(rows_match.group(1).replace(',', ''))
        diag['tables_scanned'] = int(rows_match.group(2).replace(',', ''))

    # Match connection success/failure
    if 'Connected to PostgreSQL' in stdout or 'Connected to MySQL' in stdout:
        diag['connected'] = True
    if 'Failed to connect' in stdout:
        diag['connected'] = False
        fail_match = re.search(r'Failed to connect.*?error:\s*(.+)', stdout)
        if fail_match:
            diag['connection_error'] = fail_match.group(1).strip()

    # Match skipped tables
    skip_match = re.search(r'Skipped\s+(\d+)\s+system/framework tables', stdout)
    if skip_match:
        diag['tables_skipped'] = int(skip_match.group(1))

    return diag


def _build_status_message(diagnostics: dict, findings_count: int) -> str:
    """Build a human-readable status message from scan diagnostics."""
    if diagnostics.get('connected') is False:
        err = diagnostics.get('connection_error', 'unknown error')
        return f"Connection failed: {err}"

    tables = diagnostics.get('tables_scanned')
    rows = diagnostics.get('rows_scanned')

    if tables == 0:
        return "No scannable tables found in the database. The database may be empty or contain only system tables."

    if tables is not None and rows is not None:
        if findings_count > 0:
            return f"Scanned {rows:,} rows across {tables} tables. Found {findings_count} PII findings."
        return f"Scanned {rows:,} rows across {tables} tables. No PII detected."

    if findings_count > 0:
        return f"Scan completed. Found {findings_count} PII findings."
    return "Scan completed. No PII detected."


def _normalize_for_hawk_scanner(sources: dict) -> dict:
    """
    Translate platform field names to hawk_scanner's expected format.

    The frontend stores 'username' but hawk_scanner expects 'user'.
    The frontend stores 'bucket' but hawk_scanner expects 'bucket_name'.
    Platform-only fields (environment, read_only, etc.) are stripped.
    """
    # Fields that hawk_scanner does not understand — remove them
    PLATFORM_ONLY_FIELDS = {'environment', 'read_only', 'allow_remediation', 'ssl_mode'}

    # Field renames per source type family
    DB_TYPES = {'postgresql', 'mysql', 'mongodb', 'couchdb', 'redis'}
    BUCKET_TYPES = {'s3', 'gcs', 'firebase'}

    normalized = {}
    for src_type, profiles in sources.items():
        normalized[src_type] = {}
        if not isinstance(profiles, dict):
            normalized[src_type] = profiles
            continue
        for profile_name, cfg in profiles.items():
            if not isinstance(cfg, dict):
                normalized[src_type][profile_name] = cfg
                continue

            new_cfg = {}
            for k, v in cfg.items():
                if k in PLATFORM_ONLY_FIELDS:
                    continue
                # username → user for database sources
                if k == 'username' and src_type in DB_TYPES:
                    new_cfg['user'] = v
                # bucket → bucket_name for object storage sources
                elif k == 'bucket' and src_type in BUCKET_TYPES:
                    new_cfg['bucket_name'] = v
                else:
                    new_cfg[k] = v

            # Coerce port to int for all source types that use ports.
            # JSON deserialization may produce float (3306.0) or string ("3306").
            if 'port' in new_cfg:
                try:
                    new_cfg['port'] = int(new_cfg['port'])
                except (ValueError, TypeError):
                    pass  # leave as-is if unconvertible; command defaults will apply

            normalized[src_type][profile_name] = new_cfg
    return normalized


def execute_scan(scan_id, config):
    """
    Execute the scan using hawk_scanner CLI.
    Results are ingested into the backend via API.
    """
    try:
        sources = config.get('sources', [])
        classification_mode = config.get('classification_mode', 'contextual')
        custom_patterns = config.get('custom_patterns', [])

        # Create output file for scan results
        output_file = f'/tmp/scan_output_{scan_id}.json'

        # Create a temporary connection config for this scan
        connection_config_path = f'/tmp/connection_{scan_id}.yml'

        import yaml

        # Prefer connection_configs passed directly from backend (includes credentials).
        # This keeps passwords off-disk (C-6 compliance) — they transit over the
        # internal Docker network only.
        runtime_configs = config.get('connection_configs', {})

        # Load the global connection config for notify settings and fallback
        global_config_path = 'config/connection.yml'
        global_data = {}
        if os.path.exists(global_config_path):
            with open(global_config_path, 'r') as f:
                global_data = yaml.safe_load(f) or {}

        if runtime_configs:
            # Use configs passed by backend (has credentials)
            logger.info(f"Using {sum(len(v) for v in runtime_configs.values())} runtime connection configs from backend")
            filtered_sources = runtime_configs
        else:
            # Fallback: filter from connection.yml (may lack passwords)
            logger.warning("No runtime configs from backend, falling back to connection.yml")
            filtered_sources = {}
            global_sources = global_data.get('sources', {})

            for source in sources:
                found = False
                for src_type, profiles in global_sources.items():
                    if profiles and source in profiles:
                        if src_type not in filtered_sources:
                            filtered_sources[src_type] = {}
                        filtered_sources[src_type][source] = profiles[source]
                        found = True
                        break

                # Fallback if the profile wasn't found (treat as fs path)
                if not found:
                    if 'fs' not in filtered_sources:
                        filtered_sources['fs'] = {}
                    filtered_sources['fs'][f"scan_{scan_id}_{source}"] = {
                        "path": source
                    }

        # Normalize field names so hawk_scanner gets what it expects
        # (e.g. 'username' → 'user', strip platform-only fields)
        filtered_sources = _normalize_for_hawk_scanner(filtered_sources)

        connection_data = {
            "sources": filtered_sources,
            "notify": global_data.get('notify', {})
        }

        with open(connection_config_path, 'w') as f:
            yaml.dump(connection_data, f)

        # Debug: log the YAML so we can diagnose scanner issues
        logger.info(f"Connection YAML for scan {scan_id}:\n{yaml.dump(connection_data, default_flow_style=False)}")
            
        # Build scan command using the native CLI entrypoint.
        # 'all' tells the CLI to execute the pipeline over every data source type found in the YAML.
        cmd = [
            'hawk_scanner',
            'all',
            '--json', output_file,
            '--connection', connection_config_path
        ]
        # classification_mode controls which detection engines are used:
        # 'regex' = regex+validators only (fast), 'ner' = adds spaCy NER,
        # 'contextual' (default) = all engines enabled.
        if classification_mode and classification_mode != 'contextual':
            cmd += ['--classification-mode', classification_mode]
            
        logger.info(f"Executing: {' '.join(cmd)}")

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=600  # 10 minute timeout
            )

            # Strip ASCII art banner for cleaner logs
            stdout_clean = result.stdout or ''
            if '=====' in stdout_clean:
                stdout_clean = stdout_clean[stdout_clean.rfind('=====') + 5:].strip()
            logger.info(f"stdout: {stdout_clean[:2000] if stdout_clean else 'empty'}")
            if result.stderr:
                logger.warning(f"stderr (full): {result.stderr}")

        except subprocess.TimeoutExpired:
            logger.error("Scan timed out after 600s")
            raise
        except Exception as e:
            logger.error(f"Error running scanner: {e}")
            raise
        
        # Parse hawk_scanner stdout for diagnostics
        diagnostics = _parse_scanner_diagnostics(result.stdout or '')

        # Read and ingest results
        findings_count = 0
        try:
            if os.path.exists(output_file):
                with open(output_file, 'r') as f:
                    scan_results = json.load(f)

                findings_count = sum(len(v) for v in scan_results.values() if isinstance(v, list))
                # Ingest into backend
                ingest_results(scan_id, scan_results, config)
        finally:
            # Cleanup temp files regardless of success/failure
            for tmp in (output_file, connection_config_path):
                try:
                    if os.path.exists(tmp):
                        os.remove(tmp)
                except OSError as cleanup_err:
                    logger.warning(f"Failed to remove temp file {tmp}: {cleanup_err}")

        # Build a human-readable status message
        status_message = _build_status_message(diagnostics, findings_count)
        if diagnostics.get('tables_scanned', -1) == 0:
            logger.warning(f"Scan {scan_id}: {status_message}")
        else:
            logger.info(f"Scan {scan_id}: {status_message}")

        # Update scan status in Redis
        scan_state_update(scan_id, {
            'status': 'completed',
            'completed_at': datetime.now().isoformat(),
            'diagnostics': diagnostics,
            'status_message': status_message,
        })
        if _PROM_AVAILABLE:
            scan_jobs_total.labels(status='completed').inc()
            active_scans_gauge.dec()

        # Notify backend of completion with diagnostics
        try:
            requests.post(
                f'{BACKEND_URL}/api/v1/scans/{scan_id}/complete',
                json={
                    'status': 'completed',
                    'message': status_message,
                    'diagnostics': diagnostics,
                    'classification_mode': classification_mode,
                },
                timeout=10
            )
        except Exception as e:
            logger.warning(f"Failed to notify backend of completion: {e}")

    except Exception as e:
        logger.error(f"Scan {scan_id} failed: {e}")
        scan_state_update(scan_id, {'status': 'failed', 'error': str(e)})
        if _PROM_AVAILABLE:
            scan_jobs_total.labels(status='failed').inc()
            active_scans_gauge.dec()

        # Notify backend so the scan doesn't stay "running" forever
        try:
            requests.post(
                f'{BACKEND_URL}/api/v1/scans/{scan_id}/complete',
                json={'status': 'failed'},
                timeout=10
            )
        except Exception as notify_err:
            logger.warning(f"Failed to notify backend of scan failure: {notify_err}")


_CUSTOM_PATTERN_MAX_LEN = 512
_CUSTOM_PATTERN_MAX_TEXT_LEN = 50_000
# Rough catastrophic-backtracking heuristic: nested quantifiers like (a+)+, (a*)*,
# (a+)*, (a|b)+ inside another quantifier. Not exhaustive, but catches the classic
# ReDoS shapes. Patterns that trip this are rejected at compile time; operators who
# need complex patterns should use the backend validator service instead.
_REDOS_SHAPES = __import__('re').compile(
    r'(\([^)]*[+*][^)]*\)[+*]|\([^)]*\|[^)]*\)[+*])'
)
_custom_pattern_cache: dict = {}


def _compile_custom_pattern_safely(name: str, pattern_regex: str):
    """Compile a user-supplied regex with guardrails against catastrophic backtracking.

    Returns a compiled pattern or None if the pattern is rejected. Rejections are
    logged and cached to avoid re-warning on every finding.
    """
    import re as _re
    cache_key = (name, pattern_regex)
    cached = _custom_pattern_cache.get(cache_key)
    if cached is not None:
        return cached if cached is not False else None

    if not pattern_regex or len(pattern_regex) > _CUSTOM_PATTERN_MAX_LEN:
        logger.warning(
            f"Custom pattern '{name}' rejected: length {len(pattern_regex)} exceeds "
            f"{_CUSTOM_PATTERN_MAX_LEN} chars"
        )
        _custom_pattern_cache[cache_key] = False
        return None

    if _REDOS_SHAPES.search(pattern_regex):
        logger.warning(
            f"Custom pattern '{name}' rejected: contains nested quantifier shape "
            f"associated with catastrophic backtracking (ReDoS risk)"
        )
        _custom_pattern_cache[cache_key] = False
        return None

    try:
        compiled = _re.compile(pattern_regex)
    except _re.error as compile_err:
        logger.warning(f"Custom pattern '{name}' failed to compile: {compile_err}")
        _custom_pattern_cache[cache_key] = False
        return None

    _custom_pattern_cache[cache_key] = compiled
    return compiled


def _apply_custom_patterns(raw_findings: list, custom_patterns: list) -> list:
    """Run user-defined regex patterns against sample_text in each finding row.

    Returns extra finding dicts (same shape as hawk_scanner findings) for every match.

    Guardrails applied to user-supplied regex:
      - reject patterns over _CUSTOM_PATTERN_MAX_LEN chars
      - reject patterns matching known ReDoS shapes (nested quantifiers)
      - truncate input text to _CUSTOM_PATTERN_MAX_TEXT_LEN before matching
      - compiled patterns are cached by (name, regex) tuple to avoid recompile
    """
    if not custom_patterns:
        return []

    # Precompile all patterns up-front so a bad pattern is rejected once, not
    # per-finding. Rejected patterns are dropped from this scan.
    compiled_patterns = []
    for cp in custom_patterns:
        compiled = _compile_custom_pattern_safely(
            cp.get('name', 'CUSTOM'), cp.get('regex', '')
        )
        if compiled is not None:
            compiled_patterns.append((cp, compiled))

    if not compiled_patterns:
        return []

    extra = []
    for f in raw_findings:
        text = f.get('sample_text', '') or f.get('value', '') or ''
        if not text:
            continue
        if len(text) > _CUSTOM_PATTERN_MAX_TEXT_LEN:
            text = text[:_CUSTOM_PATTERN_MAX_TEXT_LEN]
        for cp, compiled in compiled_patterns:
            try:
                matches = compiled.findall(text)
            except Exception as cp_err:
                logger.warning(f"Custom pattern '{cp.get('name')}' match error: {cp_err}")
                continue
            if not matches:
                continue
            extra.append({
                'pattern_name': cp.get('name', 'CUSTOM'),
                'matches': matches[:5],
                'sample_text': text[:200],
                'confidence_score': 0.75,
                'file_path': f.get('file_path', ''),
                'column': f.get('column', ''),
                'table': f.get('table', ''),
                'host': f.get('host', ''),
                '_custom': True,
                '_display_name': cp.get('display_name', cp.get('name', 'CUSTOM')),
                '_category': cp.get('category', 'Custom'),
            })
    return extra


def ingest_results(scan_id, results, config=None):
    """Send scan results to backend for ingestion."""
    try:
        logger.info(f"Raw results keys: {list(results.keys())}")
        custom_patterns = (config or {}).get('custom_patterns', [])

        # Transform results to VerifiedScanInput format
        verified_findings = []
        # Process all sources
        for source_type, findings in results.items():
            logger.info(f"Found {len(findings)} {source_type} findings")

            # Apply custom patterns against each finding's text
            if custom_patterns:
                extra = _apply_custom_patterns(findings, custom_patterns)
                if extra:
                    logger.info(f"Custom patterns added {len(extra)} extra findings for {source_type}")
                    findings = list(findings) + extra

            for f in findings:
                # Map pattern name to PII Type
                pattern_name = f.get('pattern_name', 'Unknown')
                pii_type = map_pattern_to_pii_type(pattern_name)

                # Skip types the backend rejects (not in India-specific locked PII whitelist)
                if pii_type is None:
                    continue

                # Format validation — reject false positives before sending to backend
                match_value = ''
                if f.get('matches') and len(f['matches']) > 0:
                    match_value = f['matches'][0]
                elif f.get('sample_text'):
                    match_value = f['sample_text']

                if match_value and not validate_pii_format(pii_type, match_value):
                    logger.debug(f"Format validation rejected {pattern_name} ({pii_type}): {match_value[:30]!r}")
                    continue

                # Extract flexible metadata across different database types
                path = f.get('file_path', '') or f.get('file_name', '') or f.get('channel_name', '')
                column = f.get('column', '') or f.get('field', '') or f.get('key', '')
                table = f.get('table', '') or f.get('collection', '') or f.get('bucket', '')

                vf = {
                    "pii_type": pii_type,
                    "value_hash": "",
                    "source": {
                        "path": path,
                        "line": 0,
                        "column": column,
                        "table": table,
                        "data_source": source_type,
                        "host": f.get('host', 'localhost')
                    },
                    "validators_passed": ["pattern_match"],
                    "validation_method": "regex",
                    "ml_confidence": f.get('confidence_score', 0.5),
                    "ml_entity_type": pii_type,
                    "context_excerpt": f.get('sample_text', ''),
                    "context_keywords": [],
                    "pattern_name": pattern_name,
                    "detected_at": datetime.utcnow().isoformat() + "Z",
                    "scanner_version": "0.3.39",
                    "metadata": f.get('file_data', {})
                }
                verified_findings.append(vf)

        # Filter by requested PII types if specified
        # Frontend sends short names (PAN, AADHAAR, EMAIL) but internal types
        # use prefixed names (IN_PAN, IN_AADHAAR, EMAIL_ADDRESS). Build a
        # lookup that accepts both forms.
        requested_pii_types = (config or {}).get('pii_types', [])
        if requested_pii_types:
            _SHORT_TO_INTERNAL = {
                'PAN': 'IN_PAN', 'AADHAAR': 'IN_AADHAAR', 'EMAIL': 'EMAIL_ADDRESS',
                'PHONE': 'IN_PHONE', 'PASSPORT': 'IN_PASSPORT', 'VOTER_ID': 'IN_VOTER_ID',
                'DRIVING_LICENSE': 'IN_DRIVING_LICENSE', 'CREDIT_CARD': 'CREDIT_CARD',
                'UPI_ID': 'IN_UPI', 'BANK_ACCOUNT': 'IN_BANK_ACCOUNT', 'GST': 'IN_GST',
                'IFSC': 'IN_IFSC', 'GSTIN': 'IN_GST',
            }
            allowed = set()
            for t in requested_pii_types:
                allowed.add(t)
                if t in _SHORT_TO_INTERNAL:
                    allowed.add(_SHORT_TO_INTERNAL[t])

            before_count = len(verified_findings)
            verified_findings = [f for f in verified_findings if f['pii_type'] in allowed]
            filtered_out = before_count - len(verified_findings)
            if filtered_out > 0:
                logger.info(f"Filtered {filtered_out} findings not in requested PII types {allowed}")

        if len(verified_findings) == 0:
            logger.info(f"Scan {scan_id} completed with zero findings — nothing to ingest")
            return

        # Layer 3 LLM classification — run Claude on ambiguous-confidence findings.
        # Findings with confidence in [0.65, 0.80] (or DPDPA-sensitive categories)
        # are batched and sent to Claude for contextual classification.
        # On API error or budget exhaustion, findings fall back to Layer 1/2 result.
        try:
            from sdk.llm_classifier import get_classifier
            # Pass the Redis client if available (for caching)
            _redis_for_llm = _redis if _redis_available else None
            llm = get_classifier(redis_client=_redis_for_llm)
            # Identify which findings need LLM classification
            needs_llm = [f for f in verified_findings if llm.should_invoke(f)]
            if needs_llm:
                logger.info(f"[scan={scan_id}] Routing {len(needs_llm)}/{len(verified_findings)} "
                            f"findings to Layer 3 LLM classifier")
                # Classify — results are merged back by list position
                classified = llm.classify_batch(needs_llm, scan_id=scan_id)
                # Merge classified results back into verified_findings
                llm_idx = 0
                for i, f in enumerate(verified_findings):
                    if llm.should_invoke(f) and llm_idx < len(classified):
                        verified_findings[i] = classified[llm_idx]
                        llm_idx += 1
        except ImportError:
            pass  # llm_classifier module not available in this environment
        except Exception as llm_err:
            logger.warning(f"[scan={scan_id}] Layer 3 LLM classification skipped: {llm_err}")

        # Smart chunking: overlap ensures PII near boundaries isn't lost
        # and related findings from the same asset stay together in at
        # least one chunk.  Backend dedup index prevents duplicate storage.
        chunks = _smart_chunk(verified_findings, chunk_size=2000, overlap=200)
        logger.info(f"Sending {len(verified_findings)} findings in {len(chunks)} smart chunks (overlap=200)")

        import time as _time
        for idx, chunk in enumerate(chunks):
            payload = {
                "scan_id": scan_id,
                "findings": chunk,
                "metadata": {"chunk": idx + 1, "total_chunks": len(chunks)}
            }

            logger.info(f"Chunk {idx + 1}/{len(chunks)} — {len(chunk)} findings")

            try:
                response = requests.post(
                    f'{BACKEND_URL}/api/v1/scans/ingest-verified',
                    json=payload,
                    headers={'Content-Type': 'application/json'},
                    timeout=300
                )

                if response.ok:
                    logger.info(f"Chunk {idx + 1} ingested successfully")
                    if _PROM_AVAILABLE:
                        for f in chunk:
                            pii_fields_detected_total.labels(pii_type=f.get('pii_type', 'UNKNOWN')).inc()
                else:
                    logger.error(f"Chunk {idx + 1} failed: {response.status_code} - {response.text}")
                    if _PROM_AVAILABLE:
                        ingest_chunk_errors_total.inc()
            except Exception as chunk_err:
                logger.error(f"Chunk {idx + 1} transport error: {chunk_err}")
                if _PROM_AVAILABLE:
                    ingest_chunk_errors_total.inc()

            # Pause between chunks to let backend commit and GC
            if idx < len(chunks) - 1:
                _time.sleep(2)

    except Exception as e:
        logger.error(f"Ingestion error: {e}", exc_info=True)

# ---------------------------------------------------------------------------
# Format validators — first line of defense before sending to backend
# ---------------------------------------------------------------------------

def _extract_digits(value: str) -> str:
    return ''.join(c for c in value if c.isdigit())


def _luhn_check(digits: str) -> bool:
    total = 0
    parity = len(digits) % 2
    for i, ch in enumerate(digits):
        d = int(ch)
        if i % 2 == parity:
            d *= 2
            if d > 9:
                d -= 9
        total += d
    return total % 10 == 0


_VERHOEFF_D = [
    [0,1,2,3,4,5,6,7,8,9],[1,2,3,4,0,6,7,8,9,5],
    [2,3,4,0,1,7,8,9,5,6],[3,4,0,1,2,8,9,5,6,7],
    [4,0,1,2,3,9,5,6,7,8],[5,9,8,7,6,0,4,3,2,1],
    [6,5,9,8,7,1,0,4,3,2],[7,6,5,9,8,2,1,0,4,3],
    [8,7,6,5,9,3,2,1,0,4],[9,8,7,6,5,4,3,2,1,0],
]
_VERHOEFF_P = [
    [0,1,2,3,4,5,6,7,8,9],[1,5,7,6,2,8,3,0,9,4],
    [5,8,0,3,7,9,6,1,4,2],[8,9,1,6,0,4,3,5,2,7],
    [9,4,5,3,1,2,6,8,7,0],[4,2,8,6,5,7,3,9,0,1],
    [2,7,9,3,8,0,6,4,1,5],[7,0,4,6,9,1,3,2,5,8],
]


def _verhoeff_check(digits: str) -> bool:
    c = 0
    n = len(digits)
    for i in range(n - 1, -1, -1):
        pos = n - 1 - i
        c = _VERHOEFF_D[c][_VERHOEFF_P[pos % 8][int(digits[i])]]
    return c == 0


_PAN_RE = re.compile(r'^[A-Z]{5}[0-9]{4}[A-Z]$')
_IFSC_RE = re.compile(r'^[A-Z]{4}0[A-Z0-9]{6}$')
_PASSPORT_RE = re.compile(r'^[A-Z][0-9]{7}$')
_VOTER_RE = re.compile(r'^[A-Z]{3}[0-9]{7}$')
_UPI_RE = re.compile(r'^[a-zA-Z0-9._-]+@[a-zA-Z][a-zA-Z0-9]*$')
_DL_RE = re.compile(r'^[A-Z]{2}[\s-]?\d{2}[\s-]?\d{4}[\s-]?\d{7}$')


def validate_pii_format(pii_type: str, value: str) -> bool:
    """Return True if value passes format validation for the given PII type."""
    value = (value or '').strip()
    if not value:
        return False

    if pii_type == 'CREDIT_CARD':
        d = _extract_digits(value)
        return 13 <= len(d) <= 19 and _luhn_check(d)

    if pii_type == 'IN_AADHAAR':
        d = _extract_digits(value)
        return len(d) == 12 and d[0] not in ('0', '1') and _verhoeff_check(d)

    if pii_type == 'IN_PAN':
        return bool(_PAN_RE.match(value.upper())) and len(value.strip()) == 10

    if pii_type == 'EMAIL_ADDRESS':
        at = value.rfind('@')
        if at < 1 or at >= len(value) - 1:
            return False
        domain = value[at+1:]
        return '.' in domain and domain[0] not in ('.','-')

    if pii_type == 'IN_PHONE':
        d = _extract_digits(value)
        if d.startswith('91') and len(d) == 12:
            d = d[2:]
        return len(d) == 10 and d[0] in '6789'

    if pii_type == 'IN_UPI':
        return bool(_UPI_RE.match(value))

    if pii_type == 'IN_IFSC':
        return bool(_IFSC_RE.match(value.upper())) and len(value.strip()) == 11

    if pii_type == 'IN_PASSPORT':
        return bool(_PASSPORT_RE.match(value.upper())) and len(value.strip()) == 8

    if pii_type == 'IN_VOTER_ID':
        return bool(_VOTER_RE.match(value.upper())) and len(value.strip()) == 10

    if pii_type == 'IN_DRIVING_LICENSE':
        v = value.upper().strip()
        cleaned = v.replace('-', '').replace(' ', '')
        return 13 <= len(cleaned) <= 16 and bool(_DL_RE.match(v))

    if pii_type == 'IN_BANK_ACCOUNT':
        d = _extract_digits(value)
        return 9 <= len(d) <= 18

    if pii_type == 'IN_GST':
        v = value.strip().upper()
        if len(v) != 15:
            return False
        try:
            state = int(v[:2])
            if state < 1 or state > 37:
                return False
        except ValueError:
            return False
        return bool(_PAN_RE.match(v[2:12]))

    return True  # Unknown type — don't reject


def _smart_chunk(findings: list, chunk_size: int = 2000, overlap: int = 200) -> list:
    """Split findings into overlapping chunks grouped by source asset.

    1. Group findings by source path (same table/file stay together).
    2. Pack groups into chunks up to *chunk_size*.
    3. Copy the last *overlap* items of each chunk to the start of the
       next one so PII spanning the boundary gets full context in at
       least one chunk.  Backend dedup index drops the duplicates.

    Returns a list of lists (each inner list is one chunk).
    """
    if len(findings) <= chunk_size:
        return [findings]

    # Group by source path so related findings stay together
    from collections import OrderedDict
    groups: OrderedDict[str, list] = OrderedDict()
    for f in findings:
        key = f.get('source', {}).get('path', '') or f.get('pattern_name', '')
        groups.setdefault(key, []).append(f)

    # Pack groups into chunks, hard-capping at chunk_size
    chunks: list[list] = []
    current: list = []

    for group_findings in groups.values():
        # If adding this group exceeds chunk_size, flush current
        if current and len(current) + len(group_findings) > chunk_size:
            chunks.append(current)
            current = current[-overlap:] if overlap else []

        current.extend(group_findings)

        # Hard cap: if a single group is very large, split it
        while len(current) > chunk_size:
            chunks.append(current[:chunk_size])
            current = current[chunk_size - overlap:]

    if current:
        chunks.append(current)

    return chunks


def map_pattern_to_pii_type(pattern_name):
    """Map hawk_scanner pattern names to backend PII types."""
    name = pattern_name.lower()
    if 'pan' in name: return 'IN_PAN'
    if 'aadhaar' in name: return 'IN_AADHAAR'
    if 'credit' in name: return 'CREDIT_CARD'
    if 'email' in name: return 'EMAIL_ADDRESS'
    if 'phone' in name: return 'IN_PHONE'
    if 'passport' in name: return 'IN_PASSPORT'
    if 'upi' in name: return 'IN_UPI'
    if 'ifsc' in name: return 'IN_IFSC'
    if 'bank' in name or 'account' in name: return 'IN_BANK_ACCOUNT'
    if 'voter' in name: return 'IN_VOTER_ID'
    if 'driving' in name or 'license' in name or 'licence' in name: return 'IN_DRIVING_LICENSE'
    if 'gst' in name or 'gstin' in name: return 'IN_GST'
    # Custom user-defined patterns: prefix with CUSTOM_ so backend can identify them
    if name.startswith('custom_') or name.startswith('usr_'):
        return pattern_name.upper()
    # Return None for types not in the backend's locked PII whitelist — caller must skip them
    return None


if __name__ == '__main__':
    port = int(os.getenv('PORT', 5002))
    app.run(host='0.0.0.0', port=port, debug=False)
