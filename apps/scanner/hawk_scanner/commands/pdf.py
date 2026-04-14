"""
PDF connector — uses PyPDF2 for text extraction, Tesseract OCR for scanned pages.
Failure SOPs:
  - Password-protected: try empty password -> mark SCAN_BLOCKED_ENCRYPTED
  - Low OCR confidence (<40%): re-run with OpenCV preprocessing -> tag confidence
"""

import os
import logging
from typing import Any, Generator

from hawk_scanner.commands.base import BaseConnector, FieldRecord, validate_magic_bytes
from hawk_scanner.commands.base import register_connector

logger = logging.getLogger(__name__)


@register_connector("pdf")
class PDFConnector(BaseConnector):
    connector_type = "pdf"

    def __init__(self, config: dict):
        super().__init__(config)
        self._file_path = config.get("path", "")
        self._ocr_enabled = config.get("ocr_enabled", True)
        self._status = "success"
        self._ocr_confidences: list[float] = []

    def connect(self) -> None:
        if not os.path.exists(self._file_path):
            raise FileNotFoundError(f"File not found: {self._file_path}")

        if not validate_magic_bytes(self._file_path, "pdf"):
            raise ValueError(f"Invalid PDF (magic bytes mismatch): {self._file_path}")

        self._connected = True
        self.logger.info("PDF connector ready for %s", self._file_path)

    def discover(self) -> list[dict[str, Any]]:
        from PyPDF2 import PdfReader

        try:
            reader = PdfReader(self._file_path)
        except Exception as exc:
            # SOP: Password-protected PDF
            self.logger.warning("Cannot open PDF %s: %s, trying empty password", self._file_path, exc)
            try:
                reader = PdfReader(self._file_path)
                reader.decrypt("")
                if reader.is_encrypted:
                    self._status = "SCAN_BLOCKED_ENCRYPTED"
                    return [{
                        "name": os.path.basename(self._file_path),
                        "type": "pdf",
                        "status": "SCAN_BLOCKED_ENCRYPTED",
                        "columns": [],
                    }]
            except Exception:
                self._status = "SCAN_BLOCKED_ENCRYPTED"
                return [{
                    "name": os.path.basename(self._file_path),
                    "type": "pdf",
                    "status": "SCAN_BLOCKED_ENCRYPTED",
                    "columns": [],
                }]

        num_pages = len(reader.pages)
        return [{
            "name": os.path.basename(self._file_path),
            "schema": os.path.dirname(self._file_path),
            "table": os.path.basename(self._file_path),
            "type": "pdf",
            "num_pages": num_pages,
            "columns": [{"name": "text_content", "data_type": "text", "nullable": True}],
        }]

    def extract(self, target: str, sample_size: int = 1000) -> Generator[FieldRecord, None, None]:
        if self._status == "SCAN_BLOCKED_ENCRYPTED":
            yield FieldRecord(
                field="__status__",
                value="SCAN_BLOCKED_ENCRYPTED",
                source=self._make_source(),
                field_path=f"{os.path.basename(self._file_path)}.__status__",
                metadata={"connector": self.connector_type, "status": "SCAN_BLOCKED_ENCRYPTED"},
            )
            return

        from PyPDF2 import PdfReader

        source = self._make_source()
        reader = PdfReader(self._file_path)

        if reader.is_encrypted:
            try:
                reader.decrypt("")
            except Exception:
                return

        page_count = 0
        for page_num, page in enumerate(reader.pages):
            if page_count >= sample_size:
                break

            text = page.extract_text() or ""

            # If no text extracted, try OCR
            if not text.strip() and self._ocr_enabled:
                text, ocr_conf = self._ocr_page(page_num)
                self._ocr_confidences.append(ocr_conf)
            else:
                # Split text into logical fields (lines or paragraphs)
                pass

            if text.strip():
                # Yield text in chunks for classification
                lines = [l.strip() for l in text.split("\n") if l.strip()]
                for i, line in enumerate(lines):
                    yield FieldRecord(
                        field=f"page_{page_num + 1}_line_{i + 1}",
                        value=line,
                        source=source,
                        field_path=f"{os.path.basename(self._file_path)}.page_{page_num + 1}.line_{i + 1}",
                        metadata={
                            "connector": self.connector_type,
                            "file": self._file_path,
                            "page": page_num + 1,
                            "ocr": bool(self._ocr_confidences),
                            "ocr_confidence": (
                                self._ocr_confidences[-1]
                                if self._ocr_confidences else None
                            ),
                        },
                    )
                    page_count += 1
                    if page_count >= sample_size:
                        break

    def _ocr_page(self, page_num: int) -> tuple[str, float]:
        """
        OCR a PDF page using Tesseract. If confidence < 40%, re-run with OpenCV preprocessing.
        """
        try:
            from hawk_scanner.profiling.ocr import ocr_pdf_page
            text, confidence = ocr_pdf_page(self._file_path, page_num)

            if confidence < 0.40:
                self.logger.warning(
                    "Low OCR confidence (%.2f) on page %d of %s, re-running with preprocessing",
                    confidence, page_num, self._file_path,
                )
                text, confidence = ocr_pdf_page(self._file_path, page_num, preprocess=True)

            return text, confidence
        except Exception as exc:
            self.logger.error("OCR failed for page %d of %s: %s", page_num, self._file_path, exc)
            return "", 0.0

    def close(self) -> None:
        self._connected = False

    def _make_source(self) -> str:
        return f"pdf://{self._file_path}"
