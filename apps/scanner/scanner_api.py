"""
Scanner HTTP API Service
Provides REST API for triggering scans and ingesting results into the backend.
"""
import uuid
import os
import json
import logging
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
        
        # Load the global connection config synced from the backend
        global_config_path = 'config/connection.yml'
        global_data = {}
        if os.path.exists(global_config_path):
            with open(global_config_path, 'r') as f:
                global_data = yaml.safe_load(f) or {}
                
        # Build the filtered sources block for this scan run
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
                
        connection_data = {
            "sources": filtered_sources,
            "notify": global_data.get('notify', {})
        }
        
        with open(connection_config_path, 'w') as f:
            yaml.dump(connection_data, f)
            
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

            logger.info(f"stdout: {result.stdout[:500] if result.stdout else 'empty'}")
            if result.stderr:
                logger.warning(f"stderr: {result.stderr[:500]}")

        except subprocess.TimeoutExpired:
            logger.error("Scan timed out after 600s")
            raise
        except Exception as e:
            logger.error(f"Error running scanner: {e}")
            raise
        
        # Read and ingest results
        try:
            if os.path.exists(output_file):
                with open(output_file, 'r') as f:
                    scan_results = json.load(f)

                # Ingest into backend
                ingest_results(scan_id, scan_results)
        finally:
            # Cleanup temp files regardless of success/failure
            for tmp in (output_file, connection_config_path):
                try:
                    if os.path.exists(tmp):
                        os.remove(tmp)
                except OSError as cleanup_err:
                    logger.warning(f"Failed to remove temp file {tmp}: {cleanup_err}")

        # Update scan status
        active_scans[scan_id]['status'] = 'completed'
        active_scans[scan_id]['completed_at'] = datetime.now().isoformat()
        
        # Notify backend of completion
        try:
            requests.post(
                f'{BACKEND_URL}/api/v1/scans/{scan_id}/complete',
                json={'status': 'completed'},
                timeout=10
            )
        except Exception as e:
            logger.warning(f"Failed to notify backend of completion: {e}")

    except Exception as e:
        logger.error(f"Scan {scan_id} failed: {e}")
        active_scans[scan_id]['status'] = 'failed'
        active_scans[scan_id]['error'] = str(e)


def ingest_results(scan_id, results):
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

        if len(verified_findings) == 0:
            logger.info(f"Scan {scan_id} completed with zero findings — nothing to ingest")
            return

        payload = {
            "scan_id": scan_id,
            "findings": verified_findings,
            "metadata": {}
        }

        logger.info(f"Sending {len(verified_findings)} verified findings to backend")

        response = requests.post(
            f'{BACKEND_URL}/api/v1/scans/ingest-verified',
            json=payload,
            headers={'Content-Type': 'application/json'},
            timeout=60
        )

        if response.ok:
            logger.info(f"Successfully ingested {len(verified_findings)} findings")
        else:
            logger.error(f"Ingestion failed: {response.status_code} - {response.text}")

    except Exception as e:
        logger.error(f"Ingestion error: {e}", exc_info=True)

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
