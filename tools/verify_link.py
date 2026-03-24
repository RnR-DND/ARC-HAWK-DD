
import os
import sys
import socket
import urllib.request
import time

# ANSI Colors
GREEN = '\033[92m'
RED = '\033[91m'
RESET = '\033[0m'
BOLD = '\033[1m'

def load_env(path):
    """Load .env file manually to avoid dependency"""
    if not os.path.exists(path):
        print(f"{RED}❌ .env file not found at {path}{RESET}")
        return {}
    
    env_vars = {}
    with open(path, 'r') as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith('#'):
                continue
            if '=' in line:
                key, value = line.split('=', 1)
                env_vars[key.strip()] = value.strip()
    return env_vars

def check_tcp(host, port, name):
    print(f"⏳ Checking {name} at {host}:{port}...", end=" ")
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(3)
        result = sock.connect_ex((host, int(port)))
        sock.close()
        if result == 0:
            print(f"{GREEN}✅ UP{RESET}")
            return True
        else:
            print(f"{RED}❌ DOWN (Code: {result}){RESET}")
            return False
    except Exception as e:
        print(f"{RED}❌ ERROR: {e}{RESET}")
        return False

def check_postgres(config):
    # Try using psycopg2 if available, else fall back to TCP
    host = config.get('DATABASE_HOST', 'localhost')
    port = config.get('DATABASE_PORT', '5432')
    user = config.get('DATABASE_USER', 'postgres')
    dbname = config.get('DATABASE_NAME', 'arc_platform')
    password = config.get('DATABASE_PASSWORD', 'postgres')

    print(f"\n{BOLD}[ PostgreSQL Check ]{RESET}")
    tcp_ok = check_tcp(host, port, "PostgreSQL TCP")
    if not tcp_ok:
        return False

    try:
        import psycopg2
        print(f"⏳ Authenticating as user '{user}' to db '{dbname}'...", end=" ")
        conn = psycopg2.connect(
            host=host,
            port=port,
            user=user,
            password=password,
            dbname=dbname
        )
        conn.close()
        print(f"{GREEN}✅ SUCCESS{RESET}")
        return True
    except ImportError:
        print(f"{RED}⚠️  psycopg2 not installed, skipping auth check{RESET}")
        return tcp_ok
    except Exception as e:
        print(f"{RED}❌ AUTH FAILED: {e}{RESET}")
        return False

def check_neo4j(config):
    host = "localhost" # Default
    # Parse URI bolt://localhost:7687
    uri = config.get('NEO4J_URI', 'bolt://localhost:7687')
    if '://' in uri:
        uri_host_port = uri.split('://')[1]
        if ':' in uri_host_port:
            host, port = uri_host_port.split(':')
        else:
            host = uri_host_port
            port = 7687
    else:
        port = 7687

    print(f"\n{BOLD}[ Neo4j Check ]{RESET}")
    return check_tcp(host, port, "Neo4j Bolt")

def check_presidio(config):
    url = config.get('PRESIDIO_URL', 'http://localhost:5001')
    print(f"\n{BOLD}[ Presidio Check ]{RESET}")
    print(f"⏳ Checking HTTP health at {url}/health...", end=" ")
    try:
        # Handle cases where url might not have http prefix
        if not url.startswith('http'):
            url = 'http://' + url
            
        req = urllib.request.Request(f"{url}/health", method='GET')
        with urllib.request.urlopen(req, timeout=3) as response:
            if response.status == 200:
                print(f"{GREEN}✅ UP (Status 200){RESET}")
                return True
            else:
                print(f"{RED}❌ DOWN (Status {response.status}){RESET}")
                return False
    except Exception as e:
        print(f"{RED}❌ ERROR: {e}{RESET}")
        return False

def main():
    print(f"{BOLD}⚡ ARC-Hawk Connectivity Verification ⚡{RESET}\n")
    
    # Path to backend .env
    env_path = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), 'apps/backend/.env')
    print(f"📂 Loading config from: {env_path}")
    
    config = load_env(env_path)
    if not config:
        print(f"{RED}⚠️  Using default values (env file missing or empty){RESET}")

    success = True
    
    if not check_postgres(config): success = False
    if not check_neo4j(config): success = False
    if not check_presidio(config): success = False # Optional, but good to check
    
    # Check Temporal
    print(f"\n{BOLD}[ Temporal Check ]{RESET}")
    temp_host = config.get('TEMPORAL_HOST', 'localhost')
    temp_port = config.get('TEMPORAL_PORT', '7233')
    if not check_tcp(temp_host, temp_port, "Temporal Frontend"): success = False

    print(f"\n" + ("="*40))
    if success:
        print(f"{GREEN}{BOLD}✨ ALL SYSTEMS GO! LINK VERIFIED.{RESET}")
        sys.exit(0)
    else:
        print(f"{RED}{BOLD}💥 LINK VERIFICATION FAILED.{RESET}")
        sys.exit(1)

if __name__ == "__main__":
    main()
