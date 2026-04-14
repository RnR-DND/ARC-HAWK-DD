"""CSV and Excel file connector — uses pandas for both CSV and XLS/XLSX reading."""

import os
import logging
from typing import Any, Generator

from hawk_scanner.commands.base import BaseConnector, FieldRecord
from hawk_scanner.commands.base import register_connector

logger = logging.getLogger(__name__)


@register_connector("csv")
@register_connector("excel")
@register_connector("csv_excel")
class CSVExcelConnector(BaseConnector):
    connector_type = "csv_excel"

    def __init__(self, config: dict):
        super().__init__(config)
        self._file_path = config.get("path", "")
        self._file_type = config.get("file_type", "")  # csv, xls, xlsx
        self._encoding = config.get("encoding", "utf-8")
        self._delimiter = config.get("delimiter", ",")
        self._sheet_name = config.get("sheet_name", None)
        self._df = None

    def connect(self) -> None:
        if not os.path.exists(self._file_path):
            raise FileNotFoundError(f"File not found: {self._file_path}")

        file_size = os.path.getsize(self._file_path)
        if file_size == 0:
            raise ValueError(f"Zero-byte file: {self._file_path}")

        # Detect file type
        if not self._file_type:
            ext = os.path.splitext(self._file_path)[1].lower()
            self._file_type = ext.lstrip(".")

        self._connected = True
        self.logger.info("CSV/Excel connector ready for %s", self._file_path)

    def discover(self) -> list[dict[str, Any]]:
        import pandas as pd

        targets = []
        if self._file_type == "csv":
            df = pd.read_csv(self._file_path, nrows=0, encoding=self._encoding,
                             delimiter=self._delimiter)
            columns = [{"name": c, "data_type": str(df[c].dtype), "nullable": True}
                       for c in df.columns]
            targets.append({
                "name": os.path.basename(self._file_path),
                "schema": os.path.dirname(self._file_path),
                "table": os.path.basename(self._file_path),
                "type": "csv",
                "columns": columns,
            })
        else:
            # Excel — may have multiple sheets
            xls = pd.ExcelFile(self._file_path)
            sheet_names = [self._sheet_name] if self._sheet_name else xls.sheet_names
            for sheet in sheet_names:
                df = pd.read_excel(xls, sheet_name=sheet, nrows=0)
                columns = [{"name": c, "data_type": str(df[c].dtype), "nullable": True}
                           for c in df.columns]
                targets.append({
                    "name": f"{os.path.basename(self._file_path)}:{sheet}",
                    "schema": os.path.dirname(self._file_path),
                    "table": sheet,
                    "type": "excel_sheet",
                    "columns": columns,
                })

        self.logger.info("Discovered %d targets in %s", len(targets), self._file_path)
        return targets

    def extract(self, target: str, sample_size: int = 1000) -> Generator[FieldRecord, None, None]:
        import pandas as pd

        source = self._make_source()

        if self._file_type == "csv":
            df = pd.read_csv(
                self._file_path, nrows=sample_size,
                encoding=self._encoding, delimiter=self._delimiter,
            )
        else:
            # target may be "filename:sheetname"
            sheet = target.split(":", 1)[1] if ":" in target else 0
            df = pd.read_excel(self._file_path, sheet_name=sheet, nrows=sample_size)

        for _, row in df.iterrows():
            for col in df.columns:
                value = row[col]
                if pd.notna(value):
                    yield FieldRecord(
                        field=str(col),
                        value=str(value),
                        source=source,
                        field_path=f"{os.path.basename(self._file_path)}.{col}",
                        metadata={
                            "connector": self.connector_type,
                            "file": self._file_path,
                            "file_type": self._file_type,
                        },
                    )

    def close(self) -> None:
        self._df = None
        self._connected = False

    def _make_source(self) -> str:
        return f"file://{self._file_path}"
