"""Apache Parquet connector — uses pyarrow."""

import os
import logging
from typing import Any, Generator

from app.connectors.base import BaseConnector, FieldRecord, validate_magic_bytes
from app.connectors import register_connector

logger = logging.getLogger(__name__)


@register_connector("parquet")
class ParquetConnector(BaseConnector):
    connector_type = "parquet"

    def __init__(self, config: dict):
        super().__init__(config)
        self._file_path = config.get("path", "")
        self._table = None

    def connect(self) -> None:
        if not os.path.exists(self._file_path):
            raise FileNotFoundError(f"File not found: {self._file_path}")

        if not validate_magic_bytes(self._file_path, "parquet"):
            raise ValueError(f"Invalid Parquet file (magic bytes mismatch): {self._file_path}")

        self._connected = True
        self.logger.info("Parquet connector ready for %s", self._file_path)

    def discover(self) -> list[dict[str, Any]]:
        import pyarrow.parquet as pq

        pf = pq.ParquetFile(self._file_path)
        schema = pf.schema_arrow
        columns = []
        for i in range(len(schema)):
            field = schema.field(i)
            columns.append({
                "name": field.name,
                "data_type": str(field.type),
                "nullable": field.nullable,
            })

        return [{
            "name": os.path.basename(self._file_path),
            "schema": os.path.dirname(self._file_path),
            "table": os.path.basename(self._file_path),
            "type": "parquet",
            "columns": columns,
            "num_row_groups": pf.metadata.num_row_groups,
            "num_rows": pf.metadata.num_rows,
        }]

    def extract(self, target: str, sample_size: int = 1000) -> Generator[FieldRecord, None, None]:
        import pyarrow.parquet as pq

        source = self._make_source()
        table = pq.read_table(self._file_path)

        # Limit rows
        if table.num_rows > sample_size:
            table = table.slice(0, sample_size)

        df = table.to_pandas()
        for _, row in df.iterrows():
            for col in df.columns:
                value = row[col]
                if value is not None and str(value) != "nan":
                    yield FieldRecord(
                        field=str(col),
                        value=str(value),
                        source=source,
                        field_path=f"{os.path.basename(self._file_path)}.{col}",
                        metadata={"connector": self.connector_type, "file": self._file_path},
                    )

    def close(self) -> None:
        self._table = None
        self._connected = False

    def _make_source(self) -> str:
        return f"parquet://{self._file_path}"
