import re

# Regex patterns for variable assignment
# Matches: key = "value", key: "value", key := "value"
ASSIGNMENT_PATTERN = re.compile(r'([a-zA-Z0-9_]+)\s*[:=]\s*["\']')

# Keywords that boost confidence
SENSITIVE_KEYWORDS = {
    'api', 'key', 'secret', 'token', 'access', 'auth',
    'password', 'pwd', 'pass', 'client_id', 'client_secret',
    'private', 'credential'
}

# Comment indicators
COMMENT_MARKERS = ['#', '//', '*', '--', '<!--']


def analyze_line_context(line):
    """
    Analyzes a line of code for semantic context.

    Returns:
        dict: {
            'is_assignment': bool,
            'variable_name': str or None,
            'is_comment': bool,
            'has_sensitive_keyword': bool
        }
    """
    context = {
        'is_assignment': False,
        'variable_name': None,
        'is_comment': False,
        'has_sensitive_keyword': False
    }

    stripped = line.strip()

    # Check for comments
    for marker in COMMENT_MARKERS:
        if stripped.startswith(marker):
            context['is_comment'] = True
            break

    if context['is_comment']:
        return context

    # Check for assignment
    match = ASSIGNMENT_PATTERN.search(line)
    if match:
        context['is_assignment'] = True
        context['variable_name'] = match.group(1).lower()

        # Check for sensitive keywords in variable name
        for keyword in SENSITIVE_KEYWORDS:
            if keyword in context['variable_name']:
                context['has_sensitive_keyword'] = True
                break

    return context
