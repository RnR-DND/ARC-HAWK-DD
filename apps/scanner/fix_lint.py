import os
import re

def fix_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()

    # Fix E712
    content = re.sub(r' == True', '', content)
    content = re.sub(r' == False', ' is False', content)

    # Fix F541 (removing f from empty f-strings)
    # We will look for f"..." or f'...' with no brackets inside, but it's tricky.
    # We can use regex to safely remove 'f' from f-strings without { and }.
    # But for a simpler approach, find all F541 warnings from flake8 output and fix those files line by line.

    with open(filepath, 'w') as f:
        f.write(content)

for root, _, files in os.walk('hawk_scanner'):
    for file in files:
        if file.endswith('.py'):
            fix_file(os.path.join(root, file))
