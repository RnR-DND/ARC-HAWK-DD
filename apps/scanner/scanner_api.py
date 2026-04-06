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
from flask import Flask, request, jsonify
from datetime import datetime

app = Flask(__name__)

# Use structured logging instead of print()
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(name)s: %(message)s'
)
logger = logging.getLogger('arc-hawk-scanner')

# Global state for tracking scans
active_scans = {}

BACKEND_URL = os.getenv('BACKEND_URL', 'http://backend:8080')

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
        
        # Mark scan as running
        active_scans[scan_id] = {
            'status': 'running',
            'started_at': datetime.now().isoformat(),
            'config': config
        }
        
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
    if scan_id in active_scans:
        return jsonify(active_scans[scan_id])
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

            # Coerce port to int at the boundary so individual commands
            # never receive a string port from the YAML config
            if 'port' in new_cfg:
                try:
                    new_cfg['port'] = int(new_cfg['port'])
                except (ValueError, TypeError):
                    pass

            normalized[src_type][profile_name] = new_cfg
    return normalized


def execute_scan(scan_id, config):
    """
    Execute the scan using hawk_scanner CLI.
    Results are ingested into the backend via API.
    """
    try:
        sources = config.get('sources', [])

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

        # Update scan status
        active_scans[scan_id]['status'] = 'completed'
        active_scans[scan_id]['completed_at'] = datetime.now().isoformat()
        active_scans[scan_id]['diagnostics'] = diagnostics
        active_scans[scan_id]['status_message'] = status_message

        # Notify backend of completion with diagnostics
        try:
            requests.post(
                f'{BACKEND_URL}/api/v1/scans/{scan_id}/complete',
                json={
                    'status': 'completed',
                    'message': status_message,
                    'diagnostics': diagnostics,
                },
                timeout=10
            )
        except Exception as e:
            logger.warning(f"Failed to notify backend of completion: {e}")

    except Exception as e:
        logger.error(f"Scan {scan_id} failed: {e}")
        active_scans[scan_id]['status'] = 'failed'
        active_scans[scan_id]['error'] = str(e)

        # Notify backend so the scan doesn't stay "running" forever
        try:
            requests.post(
                f'{BACKEND_URL}/api/v1/scans/{scan_id}/complete',
                json={'status': 'failed'},
                timeout=10
            )
        except Exception as notify_err:
            logger.warning(f"Failed to notify backend of scan failure: {notify_err}")


def ingest_results(scan_id, results, config=None):
    """Send scan results to backend for ingestion."""
    try:
        logger.info(f"Raw results keys: {list(results.keys())}")

        # Transform results to VerifiedScanInput format
        verified_findings = []
        # Process all sources
        for source_type, findings in results.items():
            logger.info(f"Found {len(findings)} {source_type} findings")

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
                'IFSC': 'IN_IFSC',
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
                else:
                    logger.error(f"Chunk {idx + 1} failed: {response.status_code} - {response.text}")
            except Exception as chunk_err:
                logger.error(f"Chunk {idx + 1} transport error: {chunk_err}")

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
    """Map hawk_scanner pattern names to backend PII types (all 11 India locked types)."""
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
    # Return None for types not in the backend's locked PII whitelist — caller must skip them
    return None


if __name__ == '__main__':
    port = int(os.getenv('PORT', 5002))
    app.run(host='0.0.0.0', port=port, debug=False)
