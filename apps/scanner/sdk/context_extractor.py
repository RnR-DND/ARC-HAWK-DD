"""
Context Window Extractor
=========================
Extracts surrounding context for PII matches to enable intelligent ML validation.

Instead of sending just "123456789012" to Presidio, we send:
"Customer Aadhaar number 123456789012 is enrolled"

This allows Presidio's NLP to boost confidence based on keywords.
"""

def extract_context_from_file(file_path: str, line_number: int, window_size: int = 100) -> str:
    """
    Extract context window around a line in a file.
    
    Args:
        file_path: Path to the file
        line_number: Line number (1-indexed)
        window_size: Number of lines before/after to include
        
    Returns:
        Context string with surrounding lines
    """
    try:
        with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
            lines = f.readlines()
        
        # Convert to 0-indexed
        line_idx = line_number - 1
        
        # Calculate window boundaries
        start = max(0, line_idx - window_size)
        end = min(len(lines), line_idx + window_size + 1)
        
        # Extract and join context
        context = ''.join(lines[start:end])
        return context
    
    except Exception as e:
        # Fallback: return empty context
        return ""


def extract_context_from_database_row(row_data: dict, column_name: str, window_chars: int = 200) -> str:
    """
    Extract context from a database row.
    
    For databases, "context" means the entire row or adjacent columns.
    
    Args:
        row_data: Dictionary of column_name -> value
        column_name: The column containing the PII
        window_chars: Max characters of context
        
    Returns:
        Context string
    """
    try:
        # Build context from row
        context_parts = []
        
        for col, value in row_data.items():
            if col == column_name:
                # Include the PII column
                context_parts.append(f"{col}: {value}")
            else:
                # Include adjacent columns for context
                context_parts.append(f"{col}: {str(value)[:50]}")
        
        # Join and limit length
        context = " | ".join(context_parts)
        
        if len(context) > window_chars:
            context = context[:window_chars]
        
        return context
    
    except Exception as e:
        return ""


if __name__ == "__main__":
    print("=== Context Window Extractor Tests ===\n")
    
    # Test file extraction (would need actual file)
    print("File context extraction: Implemented")
    
    # Test database context
    sample_row = {
        "id": 123,
        "customer_name": "John Doe",
        "aadhaar_number": "9999 1111 2226",
        "email": "john@example.com"
    }
    
    context = extract_context_from_database_row(sample_row, "aadhaar_number")
    print(f"\nDatabase context for Aadhaar:\n{context}")
    
    print("\n✓ Context extractor ready")
