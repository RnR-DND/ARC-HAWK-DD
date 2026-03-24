import math


def calculate_shannon_entropy(data):
    """
    Calculates the Shannon Entropy of a string.

    Formula: H(X) = - sum(p(x) * log2(p(x)))

    Args:
        data: Input string

    Returns:
        float: Entropy value
    """
    if not data:
        return 0

    entropy = 0
    length = len(data)

    # Count occurrences of each character
    counts = {}
    for char in data:
        counts[char] = counts.get(char, 0) + 1

    # Calculate entropy
    for count in counts.values():
        probability = count / length
        entropy -= probability * math.log2(probability)

    return entropy


def is_high_entropy(data, threshold=3.5):
    """
    Checks if a string has high entropy (indicative of a random secret).

    Args:
        data: String to check
        threshold: Entropy threshold (default 3.5 is good for API keys)

    Returns:
        bool: True if high entropy
    """
    return calculate_shannon_entropy(data) > threshold
