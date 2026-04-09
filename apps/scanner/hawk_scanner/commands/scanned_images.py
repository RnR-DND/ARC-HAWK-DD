"""
Scanned Images connector for ARC-HAWK-DD scanner.
Uses OCR to extract text from scanned image files, then applies PII pattern matching.

Requires:
  - pytesseract (already in requirements.txt)
  - Pillow / PIL (already in requirements.txt)
  - opencv-python / cv2 (already in requirements.txt)
  - System dependency: tesseract-ocr binary must be installed
    (e.g. apt-get install tesseract-ocr  OR  brew install tesseract)

Supported image formats: .png, .jpg, .jpeg, .tiff, .tif, .bmp, .gif, .webp, .pdf

Configuration in connection.yml:

    sources:
      scanned_images:
        id_documents:
          paths:
            - /data/scanned_docs/
            - /uploads/kyc/
          recursive: true
          min_ocr_confidence: 60   # 0-100; default 60. Pages below threshold are retried with pre-processing.
          languages: eng           # Tesseract language codes; default 'eng'
          file_extensions:         # optional; defaults to common image types
            - .png
            - .jpg
            - .tiff
"""

import os
import io
from pathlib import Path

from hawk_scanner.internals import system
from hawk_scanner.internals.validation_integration import validate_findings
from rich.console import Console

console = Console()

# Supported image extensions (lowercase)
DEFAULT_EXTENSIONS = {'.png', '.jpg', '.jpeg', '.tiff', '.tif', '.bmp', '.gif', '.webp'}

# Confidence threshold below which we apply OpenCV pre-processing and retry
DEFAULT_MIN_CONFIDENCE = 60


# ---------------------------------------------------------------------------
# OCR helpers
# ---------------------------------------------------------------------------

def _load_image_pil(file_path):
    """
    Load image via PIL. Returns PIL.Image or raises on error.
    """
    from PIL import Image
    return Image.open(file_path).convert('RGB')


def _preprocess_for_ocr(pil_image):
    """
    Apply OpenCV pre-processing to improve OCR accuracy on low-quality scans.

    Steps:
      1. Convert PIL → numpy array (BGR for OpenCV)
      2. Convert to greyscale
      3. Apply Gaussian blur to reduce noise
      4. Adaptive thresholding for binarisation
      5. Dilation to thicken strokes
      6. Convert back to PIL Image
    """
    import numpy as np
    import cv2
    from PIL import Image

    img_np = np.array(pil_image)
    # PIL gives RGB; OpenCV expects BGR
    img_bgr = cv2.cvtColor(img_np, cv2.COLOR_RGB2BGR)
    grey = cv2.cvtColor(img_bgr, cv2.COLOR_BGR2GRAY)

    # Gaussian blur to reduce scanner noise
    blurred = cv2.GaussianBlur(grey, (5, 5), 0)

    # Adaptive threshold — handles uneven lighting across scanned pages
    binary = cv2.adaptiveThreshold(
        blurred, 255,
        cv2.ADAPTIVE_THRESH_GAUSSIAN_C,
        cv2.THRESH_BINARY,
        11, 2
    )

    # Dilate to reconnect broken characters
    kernel = cv2.getStructuringElement(cv2.MORPH_RECT, (2, 2))
    dilated = cv2.dilate(binary, kernel, iterations=1)

    return Image.fromarray(dilated)


def _run_tesseract(pil_image, lang='eng'):
    """
    Run Tesseract OCR on a PIL image.

    Returns:
        (text: str, mean_confidence: float)
        mean_confidence is 0-100; -1 if data unavailable.
    """
    import pytesseract

    # Get detailed OCR data including per-word confidence scores
    try:
        data = pytesseract.image_to_data(
            pil_image,
            lang=lang,
            output_type=pytesseract.Output.DICT
        )
        confidences = [
            int(c) for c in data['conf']
            if str(c).strip() not in ('-1', '')
        ]
        mean_conf = sum(confidences) / len(confidences) if confidences else -1
        text = ' '.join(
            w for w, c in zip(data['text'], data['conf'])
            if str(c).strip() not in ('-1', '') and int(c) > 0 and w.strip()
        )
    except Exception:
        # Fall back to simple string extraction
        text = pytesseract.image_to_string(pil_image, lang=lang)
        mean_conf = -1

    return text.strip(), mean_conf


def connect_scanned_images(args, path):
    """
    Validate that a scanned-images path exists and required libraries are importable.

    Args:
        args: Parsed CLI arguments
        path: File or directory path

    Returns:
        str path if valid, None on failure
    """
    # Check pytesseract availability
    try:
        import pytesseract  # noqa: F401
    except ImportError:
        system.print_error(
            args,
            "pytesseract not installed. Run: pip install pytesseract"
        )
        return None

    # Check that the tesseract binary is reachable
    try:
        import pytesseract
        version = pytesseract.get_tesseract_version()
        system.print_info(args, f"Tesseract version: {version}")
    except Exception as e:
        system.print_error(
            args,
            f"Tesseract binary not found or not executable: {e}. "
            f"Install with: apt-get install tesseract-ocr  OR  brew install tesseract"
        )
        return None

    # Check Pillow
    try:
        from PIL import Image  # noqa: F401
    except ImportError:
        system.print_error(args, "Pillow not installed. Run: pip install Pillow")
        return None

    # Check cv2
    try:
        import cv2  # noqa: F401
    except ImportError:
        system.print_error(args, "opencv-python not installed. Run: pip install opencv-python")
        return None

    if not os.path.exists(path):
        system.print_error(args, f"Scanned images path does not exist: {path}")
        return None

    system.print_info(args, f"Scanned images source validated: {path}")
    return path


def scan_image_file(args, file_path, profile_name, lang='eng', min_confidence=DEFAULT_MIN_CONFIDENCE):
    """
    Scan a single image file for PII using OCR.

    Pipeline:
      1. Load image with PIL
      2. Run Tesseract OCR
      3. If mean confidence < min_confidence: apply OpenCV pre-processing and retry
      4. Match extracted text against PII patterns
      5. Return findings with OCR confidence metadata

    Args:
        args: Parsed CLI arguments
        file_path: Absolute path to the image file
        profile_name: Profile key from connection.yml
        lang: Tesseract language code(s), e.g. 'eng' or 'eng+hin'
        min_confidence: Minimum acceptable OCR confidence (0-100)

    Returns:
        List of finding dicts
    """
    results = []

    try:
        pil_image = _load_image_pil(file_path)
    except Exception as e:
        system.print_error(args, f"Cannot open image {file_path}: {e}")
        return []

    # First OCR pass
    text, confidence = _run_tesseract(pil_image, lang=lang)
    system.print_info(
        args,
        f"  {Path(file_path).name}: OCR confidence={confidence:.1f}%, "
        f"text_length={len(text)} chars"
    )

    # Retry with pre-processing if confidence is below threshold
    if confidence != -1 and confidence < min_confidence:
        system.print_info(
            args,
            f"  Low OCR confidence ({confidence:.1f}% < {min_confidence}%) — "
            f"retrying with OpenCV pre-processing"
        )
        try:
            preprocessed = _preprocess_for_ocr(pil_image)
            retry_text, retry_confidence = _run_tesseract(preprocessed, lang=lang)
            system.print_info(
                args,
                f"  After pre-processing: confidence={retry_confidence:.1f}%, "
                f"text_length={len(retry_text)} chars"
            )
            # Use whichever pass produced better confidence
            if retry_confidence > confidence or confidence == -1:
                text, confidence = retry_text, retry_confidence
        except Exception as e:
            system.print_error(
                args,
                f"  OpenCV pre-processing failed for {file_path}: {e}"
            )

    if not text:
        system.print_info(args, f"  No text extracted from {file_path} — skipping")
        return []

    # Match PII patterns against extracted text
    matches = system.match_strings(args, text)
    if matches:
        validated = validate_findings(matches, args)
        if validated:
            for match in validated:
                results.append({
                    'host': str(Path(file_path).parent),
                    'file_path': file_path,
                    'column': 'ocr_text',
                    'ocr_confidence': round(confidence, 1),
                    'pattern_name': match['pattern_name'],
                    'matches': match['matches'],
                    'sample_text': match['sample_text'],
                    'profile': profile_name,
                    'data_source': 'scanned_images',
                })

    return results


def _collect_image_files(base_path, extensions, recursive):
    """
    Collect all image files matching the given extensions under base_path.

    Args:
        base_path: File or directory
        extensions: Set of lowercase file extensions (e.g. {'.png', '.jpg'})
        recursive: Whether to recurse into subdirectories

    Returns:
        List of absolute file paths (str)
    """
    if os.path.isfile(base_path):
        ext = Path(base_path).suffix.lower()
        return [base_path] if ext in extensions else []

    if not os.path.isdir(base_path):
        return []

    pattern = '**/*' if recursive else '*'
    all_files = Path(base_path).glob(pattern)
    return [
        str(f) for f in all_files
        if f.is_file() and f.suffix.lower() in extensions
    ]


def execute(args):
    """
    Entry point — called by all.py and hawk_scanner CLI.

    Reads 'scanned_images' section from connection.yml sources:

        sources:
          scanned_images:
            kyc_uploads:
              paths:
                - /data/kyc/
                - /uploads/id_docs/
              recursive: true
              languages: eng
              min_ocr_confidence: 60
              file_extensions:
                - .png
                - .jpg
                - .jpeg
                - .tiff
    """
    results = []
    system.print_info(args, "Running checks for Scanned Images sources")
    connections = system.get_connection(args)

    sources_config = connections.get('sources', {})
    images_config = sources_config.get('scanned_images')

    if not images_config:
        system.print_error(
            args, "No scanned_images configuration found in connection.yml"
        )
        return results

    for key, config in images_config.items():
        paths = config.get('paths', [])
        lang = config.get('languages', 'eng')
        min_confidence = int(config.get('min_ocr_confidence', DEFAULT_MIN_CONFIDENCE))
        recursive = config.get('recursive', True)
        raw_extensions = config.get('file_extensions', list(DEFAULT_EXTENSIONS))
        extensions = {ext.lower() if ext.startswith('.') else f'.{ext.lower()}'
                      for ext in raw_extensions}

        if not paths:
            system.print_error(args, f"No paths defined for scanned_images profile '{key}'")
            continue

        system.print_info(
            args,
            f"Checking scanned_images profile '{key}' "
            f"(lang={lang}, min_confidence={min_confidence}%)"
        )

        for base_path in paths:
            validated_path = connect_scanned_images(args, base_path)
            if validated_path is None:
                continue

            image_files = _collect_image_files(validated_path, extensions, recursive)
            if not image_files:
                system.print_info(args, f"No image files found in: {validated_path}")
                continue

            system.print_info(
                args,
                f"Found {len(image_files)} image file(s) in {validated_path}"
            )

            for img_path in sorted(image_files):
                system.print_info(args, f"  Scanning image: {img_path}")
                file_results = scan_image_file(
                    args, img_path, key,
                    lang=lang,
                    min_confidence=min_confidence
                )
                results += file_results

    system.print_success(
        args, f"Scanned images scan complete: {len(results)} finding(s)"
    )
    return results


if __name__ == "__main__":
    execute(None)
