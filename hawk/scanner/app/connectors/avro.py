"""Apache Avro connector — uses pyarrow for reading."""

import os
import logging
from typing import Any, Generator

from app.connectors.base import BaseConnector, FieldRecord, validate_magic_bytes
from app.connectors import register_connector

logger = logging.getLogger(__name__)


@register_connector("avro")
class AvroConnector(BaseConnector):
    connector_type = "avro"

    def __init__(self, config: dict):
        super().__init__(config)
        self._file_path = config.get("path", "")

    def connect(self) -> None:
        if not os.path.exists(self._file_path):
            raise FileNotFoundError(f"File not found: {self._file_path}")

        if not validate_magic_bytes(self._file_path, "avro"):
            raise ValueError(f"Invalid Avro file (magic bytes mismatch): {self._file_path}")

        self._connected = True
        self.logger.info("Avro connector ready for %s", self._file_path)

    def discover(self) -> list[dict[str, Any]]:
        import pyarrow.ipc as ipc
        import fastavro

        columns = []
        with open(self._file_path, "rb") as f:
            reader = fastavro.reader(f)
            schema = reader.writer_schema
            if schema and "fields" in schema:
                for field in schema["fields"]:
                    ftype = field.get("type", "unknown")
                    if isinstance(ftype, list):
                        ftype = [t for t in ftype if t != "null"]
                        ftype = ftype[0] if len(ftype) == 1 else str(ftype)
                    columns.append({
                        "name": field["name"],
                        "data_type": str(ftype),
                        "nullable": True,
                    })

        return [{
            "name": os.path.basename(self._file_path),
            "schema": os.path.dirname(self._file_path),
            "table": os.path.basename(self._file_path),
            "type": "avro",
            "columns": columns,
        }]

    def extract(self, target: str, sample_size: int = 1000) -> Generator[FieldRecord, None, None]:
        import fastavro

        source = self._make_source()
        count = 0
        with open(self._file_path, "rb") as f:
            reader = fastavro.reader(f)
            for record in reader:
                if count >= sample_size:
                    break
                for field_name, value in record.items():
                    if value is not None:
                        yield FieldRecord(
                            field=field_name,
                            value=str(value),
                            source=source,
                            field_path=f"{os.path.basename(self._file_path)}.{field_name}",
                            metadata={"connector": self.connector_type, "file": self._file_path},
                        )
                count += 1

    def close(self) -> None:
        self._connected = False

    def _make_source(self) -> str:
        return f"avro://{self._file_path}"
