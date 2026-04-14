"""
Base connector abstraction for file-format connectors (CSV, PDF, DOCX, etc.)
Migrated from hawk/scanner/app/connectors/ — provides a simple field-record interface
for connectors that don't use the direct system.match_strings() approach.
"""
from dataclasses import dataclass, field
from typing import Any, Optional
import magic


@dataclass
class FieldRecord:
    """A single field/value extracted from a data source."""
    field_name: str
    value: str
    row_number: Optional[int] = None
    column_index: Optional[int] = None
    metadata: dict = field(default_factory=dict)


def validate_magic_bytes(path: str, expected: bytes) -> bool:
    """Validate file magic bytes before processing."""
    try:
        with open(path, 'rb') as f:
            header = f.read(len(expected))
        return header == expected
    except (IOError, OSError):
        return False


def register_connector(name: str):
    """Decorator stub — connectors are registered via hawk_scanner.commands __init__."""
    def decorator(cls):
        return cls
    return decorator


class BaseConnector:
    """Base class for file-format connectors."""
    connector_type: str = ""

    def __init__(self, config: dict):
        self.config = config

    def connect(self) -> bool:
        return True

    def close(self):
        pass

    def stream_fields(self) -> Any:
        raise NotImplementedError

    def __enter__(self):
        self.connect()
        return self

    def __exit__(self, *args):
        self.close()
