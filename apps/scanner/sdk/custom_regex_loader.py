"""
Custom Regex Hot-Reload Loader
==============================
Loads custom regex patterns from the PostgreSQL `custom_patterns` table and
hot-reloads them in a background thread every `reload_interval` seconds
(default 60s).

New patterns are picked up at the next batch boundary — a `patterns_updated`
flag is set after each successful reload so callers can decide when to apply
the new set.

Usage:

    loader = CustomRegexLoader(db_url="postgresql://user:pass@host/db")
    loader.start()                        # non-blocking; starts background thread

    patterns = loader.get_patterns()      # thread-safe read
    if loader.patterns_updated:           # True when a new batch is loaded
        loader.clear_updated_flag()       # reset after applying at batch boundary

Schema expected in PostgreSQL:

    CREATE TABLE custom_patterns (
        id            SERIAL PRIMARY KEY,
        name          TEXT NOT NULL,
        pattern       TEXT NOT NULL,         -- Python regex string
        pii_category  TEXT NOT NULL,         -- e.g. 'AADHAAR', 'PAN', 'CUSTOM_ID'
        sensitivity   TEXT NOT NULL DEFAULT 'High',
        is_active     BOOLEAN NOT NULL DEFAULT TRUE,
        deleted_at    TIMESTAMPTZ,
        created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
"""

import re
import time
import logging
import threading
from typing import List, Optional

logger = logging.getLogger(__name__)


class CustomRegexLoader:
    """
    Loads custom regex patterns from the `custom_patterns` PostgreSQL table
    and refreshes them every `reload_interval` seconds in a daemon thread.

    Thread-safety:
        All reads / writes to `_patterns` and `_patterns_updated` are
        protected by `_lock` (reentrant lock).

    Attributes:
        _patterns (list[dict]): Currently loaded active patterns.
        _patterns_updated (bool): Set to True each time patterns are refreshed.
            Cleared by the caller via `clear_updated_flag()`.
        _db_url (str): psycopg2-compatible connection URL.
        _reload_interval (int): Seconds between background reloads.
        _thread (threading.Thread): Background daemon thread.
    """

    def __init__(self, db_url: str, reload_interval: int = 60) -> None:
        """
        Args:
            db_url: PostgreSQL connection URL understood by psycopg2,
                    e.g. "postgresql://user:pass@host:5432/dbname"
            reload_interval: Seconds between background reloads. Default 60.
        """
        if not db_url:
            raise ValueError("db_url must be a non-empty connection string")
        if reload_interval < 5:
            raise ValueError("reload_interval must be >= 5 seconds")

        self._db_url: str = db_url
        self._reload_interval: int = reload_interval

        self._patterns: List[dict] = []
        self._patterns_updated: bool = False
        self._lock = threading.RLock()
        self._thread: Optional[threading.Thread] = None
        self._stop_event = threading.Event()

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def start(self) -> None:
        """
        Perform an initial synchronous load, then start the background
        reload thread.

        This method is idempotent — calling it more than once is safe (the
        second call is a no-op).
        """
        if self._thread is not None and self._thread.is_alive():
            logger.debug("CustomRegexLoader already running — start() is a no-op")
            return

        # Synchronous initial load so patterns are available immediately
        self._load_patterns()

        self._stop_event.clear()
        self._thread = threading.Thread(
            target=self._reload_loop,
            name="custom-regex-loader",
            daemon=True,
        )
        self._thread.start()
        logger.info(
            "CustomRegexLoader started (interval=%ds, patterns=%d)",
            self._reload_interval,
            len(self._patterns),
        )

    def stop(self) -> None:
        """
        Signal the background thread to stop and wait for it to exit.
        Useful in tests or graceful shutdown hooks.
        """
        self._stop_event.set()
        if self._thread is not None:
            self._thread.join(timeout=self._reload_interval + 5)
        logger.info("CustomRegexLoader stopped")

    def get_patterns(self) -> List[dict]:
        """
        Return a snapshot of the currently loaded patterns.

        Thread-safe: acquires the internal lock and returns a shallow copy
        so the caller cannot mutate the internal list.

        Returns:
            list[dict] where each dict has keys:
                id, name, pattern, pii_category, sensitivity
        """
        with self._lock:
            return list(self._patterns)

    @property
    def patterns_updated(self) -> bool:
        """
        True if patterns were reloaded since the last `clear_updated_flag()`.
        Intended to be polled at batch boundaries.
        """
        with self._lock:
            return self._patterns_updated

    def clear_updated_flag(self) -> None:
        """
        Reset the `patterns_updated` flag. Call this after applying the
        new pattern set at a batch boundary.
        """
        with self._lock:
            self._patterns_updated = False

    def pattern_count(self) -> int:
        """Return the number of currently loaded patterns."""
        with self._lock:
            return len(self._patterns)

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _reload_loop(self) -> None:
        """
        Background loop: sleep for `_reload_interval` seconds, then reload.
        Exits cleanly when `_stop_event` is set.
        """
        while not self._stop_event.wait(timeout=self._reload_interval):
            self._load_patterns()

    def _load_patterns(self) -> None:
        """
        Connect to PostgreSQL and load all active, non-deleted custom patterns.

        Query:
            SELECT id, name, pattern, pii_category, sensitivity
            FROM   custom_patterns
            WHERE  is_active = true
              AND  deleted_at IS NULL

        Invalid regex patterns are skipped with a warning rather than
        raising — a bad pattern in the DB should not abort the loader.

        On any database error the existing in-memory patterns are preserved
        (fail-open) so scanning continues with stale-but-valid patterns.
        """
        try:
            import psycopg2
            import psycopg2.extras
        except ImportError:
            logger.error(
                "psycopg2 not installed — cannot load custom patterns. "
                "Run: pip install psycopg2-binary"
            )
            return

        try:
            conn = psycopg2.connect(self._db_url)
        except Exception as exc:
            logger.warning(
                "CustomRegexLoader: could not connect to DB (%s) — "
                "keeping existing %d pattern(s)",
                exc, len(self._patterns),
            )
            return

        try:
            with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
                cur.execute(
                    """
                    SELECT id, name, pattern, pii_category, sensitivity
                    FROM   custom_patterns
                    WHERE  is_active  = true
                      AND  deleted_at IS NULL
                    ORDER  BY id
                    """
                )
                rows = cur.fetchall()
        except Exception as exc:
            logger.warning(
                "CustomRegexLoader: query failed (%s) — "
                "keeping existing %d pattern(s)",
                exc, len(self._patterns),
            )
            conn.close()
            return
        finally:
            conn.close()

        # Validate regex before accepting
        valid_patterns: List[dict] = []
        for row in rows:
            raw_pattern = row.get('pattern', '')
            try:
                re.compile(raw_pattern)
            except re.error as exc:
                logger.warning(
                    "CustomRegexLoader: pattern id=%s name=%r has invalid regex %r: %s — skipped",
                    row.get('id'), row.get('name'), raw_pattern, exc,
                )
                continue
            valid_patterns.append({
                'id': row['id'],
                'name': row['name'],
                'pattern': raw_pattern,
                'pii_category': row['pii_category'],
                'sensitivity': row.get('sensitivity', 'High'),
            })

        prev_count = len(self._patterns)
        with self._lock:
            self._patterns = valid_patterns
            self._patterns_updated = True

        logger.info(
            "CustomRegexLoader: loaded %d pattern(s) "
            "(was %d, delta=%+d)",
            len(valid_patterns), prev_count,
            len(valid_patterns) - prev_count,
        )

    # ------------------------------------------------------------------
    # Convenience: build a Presidio PatternRecognizer from loaded patterns
    # ------------------------------------------------------------------

    def build_presidio_recognizers(self):
        """
        Convert loaded custom patterns into Presidio PatternRecognizer instances.

        Returns:
            list[presidio_analyzer.PatternRecognizer]
        """
        try:
            from presidio_analyzer import PatternRecognizer, Pattern
        except ImportError:
            logger.error(
                "presidio-analyzer not installed — cannot build recognizers"
            )
            return []

        recognizers = []
        for p in self.get_patterns():
            try:
                presidio_pattern = Pattern(
                    name=p['name'],
                    regex=p['pattern'],
                    score=0.85,
                )
                recognizer = PatternRecognizer(
                    supported_entity=p['pii_category'],
                    patterns=[presidio_pattern],
                    name=f"CustomDB_{p['name']}",
                )
                recognizers.append(recognizer)
            except Exception as exc:
                logger.warning(
                    "Could not build Presidio recognizer for pattern id=%s: %s",
                    p['id'], exc,
                )

        return recognizers


# ---------------------------------------------------------------------------
# Module-level singleton (optional convenience)
# ---------------------------------------------------------------------------

_global_loader: Optional[CustomRegexLoader] = None


def get_global_loader(db_url: str = None, reload_interval: int = 60) -> CustomRegexLoader:
    """
    Get or create a module-level singleton CustomRegexLoader.

    Args:
        db_url: Required on first call; ignored on subsequent calls.
        reload_interval: Reload interval in seconds (first call only).

    Returns:
        The global CustomRegexLoader instance.
    """
    global _global_loader
    if _global_loader is None:
        if not db_url:
            raise ValueError(
                "db_url is required when calling get_global_loader() for the first time"
            )
        _global_loader = CustomRegexLoader(db_url=db_url, reload_interval=reload_interval)
        _global_loader.start()
    return _global_loader
