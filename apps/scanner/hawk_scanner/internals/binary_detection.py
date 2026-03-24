"""
Binary file detection utilities for Hawk Scanner
Prevents scanning binary files that contain null bytes
"""

import mimetypes


def is_text_file(filepath, chunk_size=8192):
    """
    Detect if a file is text-based (safe to scan) or binary

    Args:
        filepath: Path to file to check
        chunk_size: Number of bytes to read for detection (default 8KB)

    Returns:
        bool: True if file is text, False if binary
    """
    # Check MIME type first (fast check)
    mime_type, _ = mimetypes.guess_type(filepath)
    if mime_type:
        # Explicitly reject known binary MIME types
        if not mime_type.startswith('text'):
            # But allow some special cases
            allowed_text_types = [
                'application/json',
                'application/xml',
                'application/javascript',
                'application/x-yaml',
                'application/sql'
            ]
            if mime_type not in allowed_text_types:
                return False

    # Read first chunk to check for null bytes
    try:
        with open(filepath, 'rb') as f:
            chunk = f.read(chunk_size)

            # Check for null bytes (binary indicator)
            if b'\x00' in chunk:
                return False

            # Check if content is mostly printable ASCII/UTF-8
            try:
                chunk.decode('utf-8')
                return True
            except UnicodeDecodeError:
                # Try latin-1 as fallback
                try:
                    chunk.decode('latin-1')
                    return True
                except:
                    return False

    except (IOError, OSError):
        # If we can't read the file, assume it's not scannable
        return False


def get_binary_file_stats(files):
    """
    Analyze a list of files and return statistics on binary vs text files

    Args:
        files: List of file paths

    Returns:
        dict: Statistics including text_count, binary_count, skipped files
    """
    stats = {
        'total': len(files),
        'text': 0,
        'binary': 0,
        'skipped_files': []
    }

    for filepath in files:
        if is_text_file(filepath):
            stats['text'] += 1
        else:
            stats['binary'] += 1
            stats['skipped_files'].append(filepath)

    return stats
