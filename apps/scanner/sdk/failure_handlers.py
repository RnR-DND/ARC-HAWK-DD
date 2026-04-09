"""
Failure SOPs as Code
=====================
Standard Operating Procedures (SOPs) for common scanner failure modes,
implemented as callable handler functions.

Each handler follows the contract:
    handler(context: dict) -> dict
where context carries failure-specific data and the return dict is a
structured outcome:
    {
        'status': str,          # e.g. 'SCAN_BLOCKED_ENCRYPTED', 'RETRIED', ...
        'action_taken': str,    # human-readable summary
        'should_retry': bool,   # whether the caller should retry the scan
        'metadata': dict,       # any extra data useful to the caller
    }

Registry:
    FAILURE_SOPS maps failure type strings to handler callables.
    Look up and dispatch via:
        handler = FAILURE_SOPS.get(failure_type)
        if handler:
            outcome = handler(context)
"""

import io
import os
import re
import time
import struct
import logging
import threading

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# 1. Encrypted PDF
# ---------------------------------------------------------------------------

def handle_encrypted_pdf(context: dict) -> dict:
    """
    SOP: Encrypted PDF encountered.

    Attempt to open with an empty password (many "protected" PDFs use no
    real password — they are marked protected but blank-password unlocks
    them).  If that fails the document is genuinely encrypted and we emit
    SCAN_BLOCKED_ENCRYPTED so the UI can surface it.

    context keys:
        file_path (str): Absolute path to the PDF file.
        password  (str, optional): Password to try before empty-password attempt.

    Returns:
        status OPEN if unlocked, SCAN_BLOCKED_ENCRYPTED if still locked.
    """
    file_path = context.get('file_path', '')
    extra_password = context.get('password', '')

    try:
        import PyPDF2
    except ImportError:
        return {
            'status': 'DEPENDENCY_MISSING',
            'action_taken': 'PyPDF2 not installed; cannot attempt PDF unlock',
            'should_retry': False,
            'metadata': {},
        }

    passwords_to_try = [p for p in [extra_password, '', b'', b'\x00'] if p is not None]

    for pwd in passwords_to_try:
        try:
            with open(file_path, 'rb') as fh:
                reader = PyPDF2.PdfReader(fh)
                if reader.is_encrypted:
                    success = reader.decrypt(pwd)
                    if success:
                        logger.info(
                            "handle_encrypted_pdf: unlocked %s with password=%r",
                            file_path, pwd,
                        )
                        return {
                            'status': 'OPEN',
                            'action_taken': f'PDF unlocked with password={pwd!r}',
                            'should_retry': True,
                            'metadata': {'unlocked_with': repr(pwd)},
                        }
                else:
                    # Not actually encrypted — original open should have worked
                    return {
                        'status': 'OPEN',
                        'action_taken': 'PDF is not encrypted; original read should succeed',
                        'should_retry': True,
                        'metadata': {},
                    }
        except Exception as exc:
            logger.debug("Unlock attempt with password=%r failed: %s", pwd, exc)

    logger.warning("handle_encrypted_pdf: %s remains locked after all attempts", file_path)
    return {
        'status': 'SCAN_BLOCKED_ENCRYPTED',
        'action_taken': 'All password attempts failed; file flagged as SCAN_BLOCKED_ENCRYPTED',
        'should_retry': False,
        'metadata': {'file_path': file_path},
    }


# ---------------------------------------------------------------------------
# 2. Low OCR Confidence
# ---------------------------------------------------------------------------

def handle_low_ocr(context: dict) -> dict:
    """
    SOP: Low OCR confidence on a scanned image.

    Re-runs OCR after applying OpenCV pre-processing:
      1. Convert to greyscale
      2. Gaussian blur (noise reduction)
      3. Adaptive threshold (binarisation)
      4. Dilation (reconnect broken glyphs)

    context keys:
        file_path        (str): Path to the image file.
        pil_image        (PIL.Image, optional): Already-loaded PIL image.
        lang             (str): Tesseract language code(s). Default 'eng'.
        original_confidence (float): OCR confidence from the failed first pass.

    Returns:
        status RETRIED_OCR_OK if retry confidence improved, OCR_LOW_CONFIDENCE if not.
        The 'text' key in metadata carries the best extracted text.
    """
    file_path = context.get('file_path', '')
    lang = context.get('lang', 'eng')
    original_confidence = context.get('original_confidence', 0.0)
    pil_image = context.get('pil_image')

    try:
        import cv2
        import numpy as np
        import pytesseract
        from PIL import Image
    except ImportError as exc:
        return {
            'status': 'DEPENDENCY_MISSING',
            'action_taken': f'Missing dependency: {exc}',
            'should_retry': False,
            'metadata': {},
        }

    # Load image if not provided
    if pil_image is None:
        try:
            pil_image = Image.open(file_path).convert('RGB')
        except Exception as exc:
            return {
                'status': 'FILE_OPEN_ERROR',
                'action_taken': f'Cannot open image {file_path}: {exc}',
                'should_retry': False,
                'metadata': {},
            }

    # OpenCV pre-processing pipeline
    try:
        img_np = np.array(pil_image)
        img_bgr = cv2.cvtColor(img_np, cv2.COLOR_RGB2BGR)
        grey = cv2.cvtColor(img_bgr, cv2.COLOR_BGR2GRAY)
        blurred = cv2.GaussianBlur(grey, (5, 5), 0)
        binary = cv2.adaptiveThreshold(
            blurred, 255,
            cv2.ADAPTIVE_THRESH_GAUSSIAN_C,
            cv2.THRESH_BINARY,
            11, 2,
        )
        kernel = cv2.getStructuringElement(cv2.MORPH_RECT, (2, 2))
        dilated = cv2.dilate(binary, kernel, iterations=1)
        preprocessed = Image.fromarray(dilated)
    except Exception as exc:
        logger.error("handle_low_ocr: pre-processing failed for %s: %s", file_path, exc)
        return {
            'status': 'PREPROCESSING_FAILED',
            'action_taken': f'OpenCV preprocessing failed: {exc}',
            'should_retry': False,
            'metadata': {'file_path': file_path},
        }

    # Re-run OCR on preprocessed image
    try:
        data = pytesseract.image_to_data(
            preprocessed,
            lang=lang,
            output_type=pytesseract.Output.DICT,
        )
        confidences = [
            int(c) for c in data['conf']
            if str(c).strip() not in ('-1', '')
        ]
        new_confidence = sum(confidences) / len(confidences) if confidences else -1
        text = ' '.join(
            w for w, c in zip(data['text'], data['conf'])
            if str(c).strip() not in ('-1', '') and int(c) > 0 and w.strip()
        )
    except Exception as exc:
        logger.error("handle_low_ocr: Tesseract retry failed: %s", exc)
        return {
            'status': 'OCR_RETRY_FAILED',
            'action_taken': f'Tesseract retry raised: {exc}',
            'should_retry': False,
            'metadata': {},
        }

    improved = new_confidence > original_confidence
    status = 'RETRIED_OCR_OK' if improved else 'OCR_LOW_CONFIDENCE'

    logger.info(
        "handle_low_ocr: %s confidence %s→%s (improved=%s)",
        file_path, original_confidence, new_confidence, improved,
    )
    return {
        'status': status,
        'action_taken': (
            f'Retried OCR after OpenCV preprocessing; '
            f'confidence {original_confidence:.1f}% → {new_confidence:.1f}%'
        ),
        'should_retry': improved,
        'metadata': {
            'original_confidence': original_confidence,
            'new_confidence': new_confidence,
            'text': text,
        },
    }


# ---------------------------------------------------------------------------
# 3. Corrupt File
# ---------------------------------------------------------------------------

# Magic byte signatures for common formats
_MAGIC_BYTES: dict = {
    'pdf':    (b'%PDF', 4),
    'parquet': (b'PAR1', 4),
    'orc':    (b'ORC', 3),
    'avro':   (b'Obj\x01', 4),
    'zip':    (b'PK\x03\x04', 4),
    'gzip':   (b'\x1f\x8b', 2),
    'png':    (b'\x89PNG\r\n\x1a\n', 8),
    'jpeg':   (b'\xff\xd8\xff', 3),
}


def handle_corrupt_file(context: dict) -> dict:
    """
    SOP: File may be corrupt.

    Validates magic bytes against the expected format.  If the magic bytes
    do not match the file extension the file is classified as CORRUPT and
    should be quarantined rather than scanned.

    context keys:
        file_path    (str): Path to the file.
        format_hint  (str, optional): Expected format key from _MAGIC_BYTES
                                      (e.g. 'pdf', 'parquet').  If omitted,
                                      inferred from file extension.

    Returns:
        status VALID_MAGIC or CORRUPT.
    """
    file_path = context.get('file_path', '')
    format_hint = context.get('format_hint', '')

    if not file_path or not os.path.exists(file_path):
        return {
            'status': 'FILE_NOT_FOUND',
            'action_taken': f'File does not exist: {file_path}',
            'should_retry': False,
            'metadata': {},
        }

    # Infer format from extension if not given
    if not format_hint:
        ext = os.path.splitext(file_path)[1].lower().lstrip('.')
        format_hint = ext

    magic_info = _MAGIC_BYTES.get(format_hint)
    if magic_info is None:
        return {
            'status': 'UNKNOWN_FORMAT',
            'action_taken': (
                f'No magic byte definition for format {format_hint!r}; '
                'cannot validate — proceeding with caution'
            ),
            'should_retry': True,
            'metadata': {'format_hint': format_hint},
        }

    expected_magic, read_len = magic_info

    try:
        with open(file_path, 'rb') as fh:
            actual_magic = fh.read(read_len)
    except OSError as exc:
        return {
            'status': 'FILE_READ_ERROR',
            'action_taken': f'Cannot read file for magic-byte check: {exc}',
            'should_retry': False,
            'metadata': {},
        }

    if actual_magic[:len(expected_magic)] == expected_magic:
        logger.info("handle_corrupt_file: %s — magic bytes valid for %s", file_path, format_hint)
        return {
            'status': 'VALID_MAGIC',
            'action_taken': f'Magic bytes match expected format {format_hint!r}',
            'should_retry': True,
            'metadata': {'format': format_hint, 'magic': actual_magic.hex()},
        }
    else:
        logger.warning(
            "handle_corrupt_file: %s — expected magic %r got %r",
            file_path, expected_magic, actual_magic,
        )
        return {
            'status': 'CORRUPT',
            'action_taken': (
                f'Magic bytes mismatch for {format_hint!r}. '
                f'Expected {expected_magic!r}, got {actual_magic!r}. '
                'File flagged as CORRUPT — quarantine recommended.'
            ),
            'should_retry': False,
            'metadata': {
                'expected_magic': expected_magic.hex(),
                'actual_magic': actual_magic.hex(),
                'file_path': file_path,
            },
        }


# ---------------------------------------------------------------------------
# 4. Archive Bomb
# ---------------------------------------------------------------------------

# Reject if extracted size would be > 50× compressed size
ARCHIVE_BOMB_RATIO = 50
# Hard cap on extracted size regardless of ratio (500 MB)
ARCHIVE_BOMB_MAX_BYTES = 500 * 1024 * 1024


def handle_archive_bomb(context: dict) -> dict:
    """
    SOP: Potential archive bomb (zip bomb / gzip bomb).

    Reads the archive's stored uncompressed-size metadata without actually
    extracting, and rejects if:
      - extracted_size / compressed_size > ARCHIVE_BOMB_RATIO (50×), OR
      - extracted_size > ARCHIVE_BOMB_MAX_BYTES (500 MB hard cap)

    For ZIP: reads the central directory.
    For GZIP: reads the ISIZE field from the last 4 bytes.

    context keys:
        file_path (str): Path to the archive.

    Returns:
        status SAFE or ARCHIVE_BOMB.
    """
    import zipfile
    import gzip

    file_path = context.get('file_path', '')

    if not file_path or not os.path.exists(file_path):
        return {
            'status': 'FILE_NOT_FOUND',
            'action_taken': f'File does not exist: {file_path}',
            'should_retry': False,
            'metadata': {},
        }

    compressed_size = os.path.getsize(file_path)
    if compressed_size == 0:
        return {
            'status': 'EMPTY_ARCHIVE',
            'action_taken': 'Archive is empty (0 bytes)',
            'should_retry': False,
            'metadata': {},
        }

    ext = os.path.splitext(file_path)[1].lower()

    # --- ZIP ---
    if ext == '.zip':
        try:
            with zipfile.ZipFile(file_path, 'r') as zf:
                total_uncompressed = sum(info.file_size for info in zf.infolist())
        except zipfile.BadZipFile as exc:
            return {
                'status': 'CORRUPT',
                'action_taken': f'Cannot read ZIP central directory: {exc}',
                'should_retry': False,
                'metadata': {},
            }

    # --- GZIP (single member; read ISIZE from trailer) ---
    elif ext in ('.gz', '.gzip'):
        try:
            with open(file_path, 'rb') as fh:
                fh.seek(-4, os.SEEK_END)
                isize_bytes = fh.read(4)
            total_uncompressed = struct.unpack('<I', isize_bytes)[0]
        except Exception as exc:
            return {
                'status': 'READ_ERROR',
                'action_taken': f'Cannot read GZIP ISIZE: {exc}',
                'should_retry': False,
                'metadata': {},
            }

    else:
        return {
            'status': 'UNSUPPORTED_FORMAT',
            'action_taken': f'Archive bomb check not supported for extension {ext!r}',
            'should_retry': True,
            'metadata': {'extension': ext},
        }

    ratio = total_uncompressed / compressed_size if compressed_size else 0
    is_bomb = (ratio > ARCHIVE_BOMB_RATIO) or (total_uncompressed > ARCHIVE_BOMB_MAX_BYTES)

    if is_bomb:
        logger.error(
            "handle_archive_bomb: ARCHIVE_BOMB detected — %s "
            "(compressed=%d bytes, uncompressed=%d bytes, ratio=%.1fx)",
            file_path, compressed_size, total_uncompressed, ratio,
        )
        return {
            'status': 'ARCHIVE_BOMB',
            'action_taken': (
                f'Archive bomb rejected: ratio={ratio:.1f}x (limit {ARCHIVE_BOMB_RATIO}x), '
                f'uncompressed={total_uncompressed:,} bytes (hard cap {ARCHIVE_BOMB_MAX_BYTES:,} bytes). '
                'File will NOT be extracted.'
            ),
            'should_retry': False,
            'metadata': {
                'compressed_size': compressed_size,
                'uncompressed_size': total_uncompressed,
                'ratio': round(ratio, 2),
                'file_path': file_path,
            },
        }

    logger.info(
        "handle_archive_bomb: %s safe (ratio=%.1fx, uncompressed=%d bytes)",
        file_path, ratio, total_uncompressed,
    )
    return {
        'status': 'SAFE',
        'action_taken': (
            f'Archive is within safe bounds: ratio={ratio:.1f}x, '
            f'uncompressed={total_uncompressed:,} bytes'
        ),
        'should_retry': True,
        'metadata': {
            'compressed_size': compressed_size,
            'uncompressed_size': total_uncompressed,
            'ratio': round(ratio, 2),
        },
    }


# ---------------------------------------------------------------------------
# 5. Expired Credentials
# ---------------------------------------------------------------------------

def handle_expired_creds(context: dict) -> dict:
    """
    SOP: Connection credentials have expired or been rejected (e.g. 401/403,
    password-expired DB error, token rotation needed).

    Steps:
      1. Pause the scan for the affected source.
      2. Emit an alert via the configured alert_fn (defaults to logging).
      3. Poll for fresh credentials every 30 s (up to max_wait_seconds).
      4. If new credentials are obtained, return CREDENTIALS_REFRESHED so the
         caller can reconnect.  Otherwise return CREDENTIALS_STILL_EXPIRED.

    context keys:
        source_name       (str): Logical source identifier (for logging/alerts).
        error_message     (str): Original auth error string.
        alert_fn          (callable, optional): alert_fn(source, message) to send
                                               alerts. Defaults to logger.error.
        get_fresh_creds   (callable, optional): Callable() → dict | None.
                                               Returns a credentials dict when
                                               new creds are available, else None.
        max_wait_seconds  (int, optional): Maximum time to poll. Default 150 (5×30s).
        poll_interval     (int, optional): Seconds between polls. Default 30.
    """
    source_name = context.get('source_name', 'unknown_source')
    error_message = context.get('error_message', '')
    alert_fn = context.get('alert_fn', None)
    get_fresh_creds = context.get('get_fresh_creds', None)
    max_wait = int(context.get('max_wait_seconds', 150))
    poll_interval = int(context.get('poll_interval', 30))

    # Emit alert
    alert_message = (
        f"[ARC-HAWK] SCAN PAUSED — Expired credentials for source '{source_name}'. "
        f"Error: {error_message}"
    )
    if callable(alert_fn):
        try:
            alert_fn(source_name, alert_message)
        except Exception as exc:
            logger.error("alert_fn raised: %s", exc)
    else:
        logger.error(alert_message)

    if not callable(get_fresh_creds):
        # No way to refresh; fail immediately
        return {
            'status': 'CREDENTIALS_STILL_EXPIRED',
            'action_taken': (
                'Alert sent; no get_fresh_creds callable provided — '
                'manual credential rotation required'
            ),
            'should_retry': False,
            'metadata': {'source_name': source_name},
        }

    # Poll for fresh credentials
    elapsed = 0
    while elapsed < max_wait:
        time.sleep(poll_interval)
        elapsed += poll_interval

        logger.info(
            "handle_expired_creds: polling for fresh credentials "
            "(elapsed=%ds / max=%ds) — source=%s",
            elapsed, max_wait, source_name,
        )

        try:
            new_creds = get_fresh_creds()
        except Exception as exc:
            logger.warning("get_fresh_creds raised: %s", exc)
            new_creds = None

        if new_creds:
            logger.info(
                "handle_expired_creds: fresh credentials obtained for %s", source_name
            )
            return {
                'status': 'CREDENTIALS_REFRESHED',
                'action_taken': (
                    f'Fresh credentials obtained after {elapsed}s — caller should reconnect'
                ),
                'should_retry': True,
                'metadata': {
                    'source_name': source_name,
                    'elapsed_seconds': elapsed,
                    'new_creds': new_creds,
                },
            }

    logger.error(
        "handle_expired_creds: credentials still expired after %ds for %s",
        max_wait, source_name,
    )
    return {
        'status': 'CREDENTIALS_STILL_EXPIRED',
        'action_taken': (
            f'Polled {max_wait // poll_interval} times over {max_wait}s; '
            'credentials still not refreshed — scan aborted for this source'
        ),
        'should_retry': False,
        'metadata': {'source_name': source_name, 'elapsed_seconds': max_wait},
    }


# ---------------------------------------------------------------------------
# 6. Schema Change
# ---------------------------------------------------------------------------

def handle_schema_change(context: dict) -> dict:
    """
    SOP: Database schema changed between scans (new/removed columns or tables).

    Steps:
      1. Diff the old schema snapshot against the current live schema.
      2. Identify new columns (candidates for PII classification).
      3. Classify new columns using the provided classify_fn or heuristic naming rules.
      4. Return the diff and classification results so the caller can update
         the asset registry and re-trigger targeted column scans.

    context keys:
        old_schema    (dict): {table_name: [col_name, ...]} — previous snapshot.
        new_schema    (dict): {table_name: [col_name, ...]} — current schema.
        classify_fn   (callable, optional): classify_fn(col_name) → pii_category | None.
                                            Defaults to heuristic name-based rules.
        source_name   (str, optional): For logging.
    """
    old_schema: dict = context.get('old_schema', {})
    new_schema: dict = context.get('new_schema', {})
    classify_fn = context.get('classify_fn', None)
    source_name = context.get('source_name', 'unknown')

    if not old_schema or not new_schema:
        return {
            'status': 'INVALID_CONTEXT',
            'action_taken': 'old_schema and new_schema are required',
            'should_retry': False,
            'metadata': {},
        }

    # Default heuristic classifier — matches common PII column name patterns
    _PII_COLUMN_HINTS = {
        r'(email|e_mail|mail)': 'EMAIL',
        r'(phone|mobile|cell|tel)': 'PHONE_NUMBER',
        r'(aadhaar|aadhar|uid)': 'AADHAAR',
        r'(pan|pan_no|pan_number)': 'PAN',
        r'(passport)': 'PASSPORT',
        r'(credit_card|card_no|card_number)': 'CREDIT_CARD',
        r'(ssn|social_security)': 'SSN',
        r'(dob|date_of_birth|birth_date)': 'DATE_OF_BIRTH',
        r'(name|first_name|last_name|full_name)': 'PERSON',
        r'(address|addr|street|city|zip|pincode)': 'ADDRESS',
        r'(ip|ip_address|ipv4|ipv6)': 'IP_ADDRESS',
        r'(bank_account|account_no|ifsc)': 'BANK_ACCOUNT',
    }

    def _heuristic_classify(col_name: str):
        col_lower = col_name.lower()
        for pattern, category in _PII_COLUMN_HINTS.items():
            if re.search(pattern, col_lower):
                return category
        return None

    effective_classify = classify_fn if callable(classify_fn) else _heuristic_classify

    diff = {
        'new_tables': [],
        'removed_tables': [],
        'new_columns': {},      # {table: [col, ...]}
        'removed_columns': {},  # {table: [col, ...]}
        'classified_new_columns': {},  # {table: {col: pii_category}}
    }

    old_tables = set(old_schema.keys())
    new_tables = set(new_schema.keys())

    diff['new_tables'] = sorted(new_tables - old_tables)
    diff['removed_tables'] = sorted(old_tables - new_tables)

    for table in new_tables & old_tables:
        old_cols = set(old_schema[table])
        new_cols = set(new_schema[table])
        added = sorted(new_cols - old_cols)
        removed = sorted(old_cols - new_cols)
        if added:
            diff['new_columns'][table] = added
        if removed:
            diff['removed_columns'][table] = removed

    # Include columns in brand-new tables too
    for table in diff['new_tables']:
        diff['new_columns'][table] = sorted(new_schema[table])

    # Classify all new columns
    for table, cols in diff['new_columns'].items():
        classified = {}
        for col in cols:
            category = effective_classify(col)
            classified[col] = category
        if classified:
            diff['classified_new_columns'][table] = classified

    total_new_cols = sum(len(c) for c in diff['new_columns'].values())
    pii_classified = sum(
        1 for table_cols in diff['classified_new_columns'].values()
        for cat in table_cols.values() if cat is not None
    )

    logger.info(
        "handle_schema_change: source=%s new_tables=%d removed_tables=%d "
        "new_cols=%d pii_classified=%d",
        source_name,
        len(diff['new_tables']),
        len(diff['removed_tables']),
        total_new_cols,
        pii_classified,
    )

    return {
        'status': 'SCHEMA_DIFF_COMPLETE',
        'action_taken': (
            f'Schema diff complete: {len(diff["new_tables"])} new tables, '
            f'{total_new_cols} new columns ({pii_classified} classified as PII)'
        ),
        'should_retry': True,
        'metadata': {'diff': diff, 'source_name': source_name},
    }


# ---------------------------------------------------------------------------
# 7. Kafka Lag
# ---------------------------------------------------------------------------

# Lag threshold above which we increase sampling
KAFKA_LAG_HIGH_THRESHOLD = 100_000
# Increased sampling rate (fraction of messages to read when lagging)
KAFKA_LAG_SAMPLE_RATE = 0.10  # 10%


def handle_kafka_lag(context: dict) -> dict:
    """
    SOP: Kafka consumer lag is too high.

    When consumer lag exceeds KAFKA_LAG_HIGH_THRESHOLD the scanner cannot
    keep up with the stream in full-fidelity mode.  This handler:
      1. Logs a STREAM_LAG_WARNING event.
      2. Returns an increased sampling rate (default 10%) that the caller
         should apply at the next batch boundary.
      3. Recommends partition rebalancing if lag has been sustained.

    context keys:
        topic             (str): Kafka topic name.
        partition         (int, optional): Partition ID if known.
        consumer_group    (str, optional): Consumer group ID.
        current_lag       (int): Current consumer lag (messages behind).
        lag_sustained_s   (int, optional): Seconds the lag has been above threshold.
                                           Used to decide whether to recommend rebalance.
        current_sample_rate (float, optional): Existing sampling rate (0-1). Default 1.0.
    """
    topic = context.get('topic', 'unknown_topic')
    partition = context.get('partition')
    consumer_group = context.get('consumer_group', 'unknown_group')
    current_lag = int(context.get('current_lag', 0))
    lag_sustained_s = int(context.get('lag_sustained_s', 0))
    current_sample_rate = float(context.get('current_sample_rate', 1.0))

    partition_label = f" partition={partition}" if partition is not None else ''

    if current_lag <= KAFKA_LAG_HIGH_THRESHOLD:
        return {
            'status': 'LAG_WITHIN_THRESHOLD',
            'action_taken': (
                f'Lag {current_lag:,} messages is within threshold '
                f'{KAFKA_LAG_HIGH_THRESHOLD:,} — no action needed'
            ),
            'should_retry': False,
            'metadata': {'current_lag': current_lag, 'sample_rate': current_sample_rate},
        }

    # Increase sampling: use KAFKA_LAG_SAMPLE_RATE if that's lower than current
    new_sample_rate = min(current_sample_rate, KAFKA_LAG_SAMPLE_RATE)

    recommend_rebalance = lag_sustained_s > 300  # sustained > 5 minutes

    action_parts = [
        f'Kafka lag alert: topic={topic}{partition_label} group={consumer_group} '
        f'lag={current_lag:,} messages (threshold={KAFKA_LAG_HIGH_THRESHOLD:,})',
        f'Increased sampling rate: {current_sample_rate:.0%} → {new_sample_rate:.0%}',
    ]
    if recommend_rebalance:
        action_parts.append(
            f'Lag sustained for {lag_sustained_s}s (>5 min) — '
            'recommend Kafka consumer group rebalance or adding partitions'
        )

    logger.warning(' | '.join(action_parts))

    return {
        'status': 'STREAM_LAG_WARNING',
        'action_taken': '; '.join(action_parts),
        'should_retry': True,
        'metadata': {
            'topic': topic,
            'partition': partition,
            'consumer_group': consumer_group,
            'current_lag': current_lag,
            'previous_sample_rate': current_sample_rate,
            'new_sample_rate': new_sample_rate,
            'recommend_rebalance': recommend_rebalance,
            'lag_sustained_s': lag_sustained_s,
        },
    }


# ---------------------------------------------------------------------------
# Public Registry
# ---------------------------------------------------------------------------

FAILURE_SOPS: dict = {
    'encrypted_pdf':       handle_encrypted_pdf,
    'low_ocr_confidence':  handle_low_ocr,
    'corrupt_file':        handle_corrupt_file,
    'archive_bomb':        handle_archive_bomb,
    'expired_credentials': handle_expired_creds,
    'schema_change':       handle_schema_change,
    'kafka_lag':           handle_kafka_lag,
}


def dispatch(failure_type: str, context: dict) -> dict:
    """
    Convenience dispatcher.  Looks up the SOP handler by failure_type and
    calls it with context.

    Args:
        failure_type: One of the keys in FAILURE_SOPS.
        context: Failure-specific context dict.

    Returns:
        Outcome dict from the handler, or an 'UNKNOWN_FAILURE_TYPE' outcome.
    """
    handler = FAILURE_SOPS.get(failure_type)
    if handler is None:
        logger.error("dispatch: unknown failure_type=%r", failure_type)
        return {
            'status': 'UNKNOWN_FAILURE_TYPE',
            'action_taken': f'No SOP registered for failure_type={failure_type!r}',
            'should_retry': False,
            'metadata': {'known_types': list(FAILURE_SOPS.keys())},
        }
    return handler(context)
