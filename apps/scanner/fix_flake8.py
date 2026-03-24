import re
import subprocess

def run_flake8():
    result = subprocess.run(['flake8', 'hawk_scanner/', '--count', '--max-complexity=10', '--max-line-length=127', '--statistics'], capture_output=True, text=True)
    return result.stdout

def fix_fstring(filepath, line_num):
    with open(filepath, 'r') as f:
        lines = f.readlines()
    
    idx = line_num - 1
    # replace the first 'f"' or "f'" with '"' or "'" but we shouldn't ruin ones that have {}. 
    # Just replacing the innermost f" or f' on that line:
    line = lines[idx]
    line = re.sub(r'f(["\'])', r'\1', line)
    lines[idx] = line
    
    with open(filepath, 'w') as f:
        f.writelines(lines)

def fix_except(filepath, line_num):
    with open(filepath, 'r') as f:
        lines = f.readlines()
    
    idx = line_num - 1
    # except Exception as e: -> except Exception:
    lines[idx] = re.sub(r' as e:', ':', lines[idx])
    
    with open(filepath, 'w') as f:
        f.writelines(lines)

output = run_flake8()
for line in output.split('\n'):
    if 'F541' in line:
        parts = line.split(':')
        file_path = parts[0]
        line_num = int(parts[1])
        fix_fstring(file_path, line_num)
    elif 'F841' in line and "'e'" in line:
        parts = line.split(':')
        file_path = parts[0]
        line_num = int(parts[1])
        fix_except(file_path, line_num)
