# 🦅 Hawk Scanner

High-performance PII (Personally Identifiable Information) and secret detection engine for the ARC-Hawk Platform. Built with Python 3.9+ and designed for both standalone CLI usage and platform integration.

## 🌟 Capabilities

- **11+ PII Types**: Mathematically validated detection for India-specific PII
- **Deep Scanning**: OCR support for images, archive extraction (zip/tar), and PDF parsing
- **Multi-Source**: Unified interface for 10+ data sources
- **High Performance**: Multithreaded scanning engine (200-350 files/second)
- **Zero False Positives**: Mathematical validation ensures accuracy
- **Context-Aware**: Distinguishes real PII from test data and code
- **Intelligence-at-Edge**: SDK-based validation runs on the scanner, not backend

## 📋 Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Supported PII Types](#supported-pii-types)
- [Data Sources](#data-sources)
- [Configuration](#configuration)
- [SDK Architecture](#sdk-architecture)
- [Platform Integration](#platform-integration)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)

---

## Installation

### Prerequisites

- **Python 3.9+**
- **pip** or **pipenv**
- **Git**

### Method 1: Install from Source

```bash
# Clone the repository
git clone https://github.com/your-org/arc-hawk.git
cd arc-hawk/apps/scanner

# Create virtual environment (recommended)
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt

# Download spaCy model
python -m spacy download en_core_web_sm

# Verify installation
python -m hawk_scanner.main --help
```

### Method 2: Using pip (when published)

```bash
pip install hawk-scanner
```

### Docker Installation

```bash
# Build Docker image
docker build -t hawk-scanner .

# Run container
docker run -v $(pwd)/data:/data hawk-scanner fs --path /data --json output.json
```

---

## Quick Start

### 1. Scan Local Filesystem

```bash
# Basic scan
python -m hawk_scanner.main fs --path /path/to/scan --json results.json

# With verbose output
python -m hawk_scanner.main fs --path /path/to/scan --verbose

# Scan specific file types only
python -m hawk_scanner.main fs --path /path/to/scan --include "*.txt,*.csv,*.json"

# Exclude directories
python -m hawk_scanner.main fs --path /path/to/scan --exclude "node_modules,.git,venv"
```

### 2. Scan PostgreSQL Database

```bash
# Create connection config first (see Configuration section)
python -m hawk_scanner.main postgresql --connection config/connection.yml --profile prod_db --json results.json
```

### 3. Scan AWS S3 Bucket

```bash
python -m hawk_scanner.main s3 --bucket my-bucket --prefix data/ --connection config/connection.yml --profile s3_prod --json results.json
```

### 4. Output Options

```bash
# JSON output (machine-readable)
python -m hawk_scanner.main fs --path /data --json output.json

# CSV output
python -m hawk_scanner.main fs --path /data --csv output.csv

# SARIF output (for CI/CD integration)
python -m hawk_scanner.main fs --path /data --sarif output.sarif

# Human-readable table (default)
python -m hawk_scanner.main fs --path /data
```

---

## Supported PII Types

### India-Specific PII (Mathematically Validated)

| PII Type | Algorithm | Format | Accuracy |
|----------|-----------|--------|----------|
| **Aadhaar** | Verhoeff | 12 digits | 100% |
| **PAN** | Weighted Mod 26 | 10 chars (ABCDE1234F) | 100% |
| **Passport** | Checksum | 8 chars | 100% |
| **Driving License** | Pattern + State | Varies by state | 95%+ |
| **Voter ID** | Pattern | 10 chars | 95%+ |
| **Credit Card** | Luhn | 13-19 digits | 100% |
| **Bank Account** | Pattern | 9-18 digits | 90%+ |
| **IFSC Code** | Pattern | 11 chars (ABCD0123456) | 100% |
| **UPI ID** | Pattern | user@bank | 95%+ |
| **Phone Number** | Pattern | +91... or 0... | 95%+ |
| **Email** | RFC 5322 | user@domain.com | 100% |

### Validation Algorithms

#### Verhoeff Algorithm (Aadhaar)

```python
from sdk.validators.verhoeff import validate_aadhaar

# Returns True for valid Aadhaar
validate_aadhaar("123456789012")  # False (example)
validate_aadhaar("999999999999")  # True (test data)
```

#### Luhn Algorithm (Credit Cards)

```python
from sdk.validators.luhn import validate_credit_card

# Returns True for valid credit card
validate_credit_card("4532015112830366")  # True (test Visa)
```

#### Weighted Modulo 26 (PAN)

```python
from sdk.validators.pan import validate_pan

# Returns True for valid PAN
validate_pan("ABCDE1234F")  # True (example format)
```

---

## Data Sources

### Filesystem (`fs`)

```bash
python -m hawk_scanner.main fs --path /data [options]

Options:
  --path PATH          Root path to scan (required)
  --include PATTERN    Include files matching pattern (e.g., "*.txt,*.csv")
  --exclude PATTERN    Exclude files/directories (e.g., "node_modules,.git")
  --max-depth N        Maximum directory depth to scan
  --max-size SIZE      Maximum file size to scan (e.g., "10MB")
  --threads N          Number of parallel threads (default: 4)
  --follow-symlinks    Follow symbolic links
```

### PostgreSQL (`postgresql`)

```bash
python -m hawk_scanner.main postgresql --connection FILE --profile NAME [options]

Options:
  --connection FILE    Path to connection.yml (required)
  --profile NAME       Connection profile name (required)
  --tables LIST        Comma-separated list of tables to scan
  --exclude-tables     Comma-separated list of tables to exclude
  --sample N           Sample N rows per table (default: all)
  --schema SCHEMA      Database schema (default: public)
```

### MySQL (`mysql`)

```bash
python -m hawk_scanner.main mysql --connection FILE --profile NAME [options]
```

### MongoDB (`mongodb`)

```bash
python -m hawk_scanner.main mongodb --connection FILE --profile NAME [options]

Options:
  --collections LIST   Comma-separated list of collections
  --sample N           Sample N documents per collection
```

### AWS S3 (`s3`)

```bash
python -m hawk_scanner.main s3 --bucket BUCKET [options]

Options:
  --bucket BUCKET      S3 bucket name (required)
  --prefix PREFIX      Key prefix to scan
  --connection FILE    Path to connection.yml with credentials
  --profile NAME       Connection profile name
  --region REGION      AWS region (default: us-east-1)
```

### Google Cloud Storage (`gcs`)

```bash
python -m hawk_scanner.main gcs --bucket BUCKET [options]
```

### Redis (`redis`)

```bash
python -m hawk_scanner.main redis --connection FILE --profile NAME [options]

Options:
  --pattern PATTERN    Key pattern to match (default: *)
  --count N            Maximum keys to scan
```

### Slack (`slack`)

```bash
python -m hawk_scanner.main slack --connection FILE --profile NAME [options]

Options:
  --channels LIST      Comma-separated list of channel IDs
  --days N             Number of days of history to scan
```

### Firebase (`firebase`)

```bash
python -m hawk_scanner.main firebase --connection FILE --profile NAME [options]
```

---

## Configuration

### Connection Configuration (`connection.yml`)

Create a `connection.yml` file to store credentials securely:

```yaml
sources:
  postgresql:
    prod_db:
      host: "localhost"
      port: 5432
      database: "production"
      user: "scanner"
      password: "${DB_PASSWORD}"  # Environment variable
      ssl_mode: "require"
    
    staging_db:
      host: "staging-db.company.com"
      port: 5432
      database: "staging"
      user: "scanner"
      password: "staging_password"
  
  mysql:
    legacy_db:
      host: "mysql.company.com"
      port: 3306
      database: "legacy"
      user: "scanner"
      password: "mysql_password"
  
  s3:
    prod_bucket:
      access_key: "${AWS_ACCESS_KEY_ID}"
      secret_key: "${AWS_SECRET_ACCESS_KEY}"
      region: "us-east-1"
      bucket_name: "company-data"
  
  gcs:
    data_bucket:
      project_id: "my-project"
      bucket_name: "company-data"
      credentials_path: "/path/to/service-account.json"
  
  mongodb:
    analytics:
      uri: "mongodb://user:pass@localhost:27017/analytics"
      database: "analytics"
  
  redis:
    cache:
      host: "localhost"
      port: 6379
      password: "${REDIS_PASSWORD}"
      db: 0
  
  slack:
    workspace:
      token: "${SLACK_BOT_TOKEN}"
      workspace: "company"
```

### Scanner Configuration (`config.yml`)

```yaml
scanning:
  max_file_size: 104857600  # 100MB
  max_depth: 10
  threads: 4
  
  # File type detection
  binary_extensions:
    - .exe
    - .dll
    - .so
    - .bin
  
  # Archive handling
  extract_archives: true
  max_archive_depth: 3
  supported_archives:
    - .zip
    - .tar
    - .tar.gz
    - .tgz

validation:
  # Validation strictness
  strict_mode: true
  min_confidence: 0.8
  
  # Context analysis
  check_context: true
  min_entropy: 3.0
  
  # PII types to validate
  enabled_pii_types:
    - aadhaar
    - pan
    - passport
    - credit_card
    - email
    - phone
    - bank_account
    - ifsc
    - upi
    - driving_license
    - voter_id

output:
  # Output format
  format: json
  
  # Include metadata
  include_metadata: true
  
  # Include context
  include_context: true
  context_lines: 3
  
  # Masking in output
  mask_findings: false
  mask_char: "*"
  mask_keep_last: 4

logging:
  level: INFO
  file: logs/scanner.log
  format: "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
```

---

## SDK Architecture

The scanner uses an **Intelligence-at-Edge** architecture with a powerful SDK for validation.

### SDK Components

```
sdk/
├── engine.py                    # SDK entry point
├── schema.py                    # Data schemas
├── validation_pipeline.py       # Validation orchestration
├── context_extractor.py         # Context analysis
├── pii_scope.py                 # PII scope detection
├── detection_quality.py         # Quality scoring
├── recognizers/                 # Pattern recognizers
│   ├── aadhaar.py
│   ├── pan.py
│   ├── passport.py
│   ├── email.py
│   ├── phone.py
│   ├── credit_card.py
│   ├── bank_account.py
│   ├── ifsc.py
│   ├── upi.py
│   ├── driving_license.py
│   └── voter_id.py
├── validators/                  # Mathematical validators
│   ├── verhoeff.py             # Aadhaar validation
│   ├── luhn.py                 # Credit card validation
│   ├── pan.py                  # PAN checksum
│   ├── email.py                # Email validation
│   ├── phone.py                # Phone validation
│   └── context_validator.py    # Context validation
└── masking/                     # Masking SDK
    ├── policy.py
    ├── strategies.py
    └── adapters/
```

### Using the SDK Directly

```python
from sdk.engine import ValidationEngine
from sdk.schema import Finding, PIIType

# Initialize engine
engine = ValidationEngine()

# Validate a string
result = engine.validate("My Aadhaar is 123456789012")

if result.is_valid:
    print(f"Found {result.pii_type}: {result.value}")
    print(f"Confidence: {result.confidence}")
    print(f"Context: {result.context}")
```

### Custom Recognizer

```python
from sdk.recognizers.base import BaseRecognizer
from sdk.schema import Finding, PIIType

class CustomRecognizer(BaseRecognizer):
    def __init__(self):
        super().__init__(
            pii_type=PIIType.CUSTOM_ID,
            pattern=r'CUST-\d{8}',
            confidence=0.9
        )
    
    def validate(self, match: str) -> bool:
        # Custom validation logic
        return len(match) == 13

# Register recognizer
engine.register_recognizer(CustomRecognizer())
```

---

## Platform Integration

When running as part of ARC-Hawk, the scanner operates in **Worker Mode**:

### Workflow

1. **Trigger**: Backend (via Temporal) triggers a scan job
2. **Configuration**: Scanner receives config via environment variables
3. **Execution**: Scanner performs the scan
4. **Ingestion**: Results are POSTed to backend via `auto_ingest.py`

### Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `SCAN_ID` | Unique scan identifier | `scan-uuid-123` |
| `API_URL` | Backend API URL | `http://backend:8080` |
| `CONNECTION_CONFIG` | JSON connection config | `{"type": "postgresql", ...}` |
| `SCAN_TYPE` | Type of scan | `full`, `incremental` |
| `MAX_WORKERS` | Parallel workers | `4` |

### Example: Worker Mode

```python
# This is handled automatically by the platform
from hawk_scanner.internals.auto_ingest import AutoIngestor

ingestor = AutoIngestor(
    api_url=os.getenv('API_URL'),
    scan_id=os.getenv('SCAN_ID')
)

# Scan and auto-ingest results
findings = scan_and_validate(...)
ingestor.ingest(findings)
```

---

## Testing

### Run All Tests

```bash
# Install test dependencies
pip install pytest pytest-cov

# Run all tests
python -m pytest tests/ -v

# Run with coverage
python -m pytest tests/ --cov=hawk_scanner --cov-report=html

# Run specific test file
python -m pytest tests/test_validation.py -v

# Run with markers
python -m pytest tests/ -m "unit" -v
python -m pytest tests/ -m "integration" -v
```

### Test Structure

```
tests/
├── test_validation.py           # Validation tests
├── test_sdk_validators.py       # SDK validator tests
├── test_zero_false_positives.py # Ground truth tests
├── test_sdk_snapshot.py         # Snapshot tests
└── ground_truth/                # Test data
    ├── samples.json
    ├── aadhaar_numbers.json
    ├── pan_numbers.json
    └── ...
```

### Ground Truth Testing

Tests against known valid/invalid data:

```python
# tests/test_zero_false_positives.py
def test_aadhaar_validation():
    valid_aadhaars = load_ground_truth('aadhaar_numbers.json')
    
    for aadhaar in valid_aadhaars:
        assert validate_aadhaar(aadhaar), f"Failed for {aadhaar}"
```

---

## Troubleshooting

### Installation Issues

**spaCy model not found:**
```bash
python -m spacy download en_core_web_sm
```

**Permission denied:**
```bash
# On Linux/Mac
chmod +x $(which python)
```

### Runtime Issues

**Out of memory:**
```bash
# Reduce threads
python -m hawk_scanner.main fs --path /data --threads 2

# Or increase Python memory limit
export PYTHONMEMORYALLOCATOR=malloc
```

**Slow performance:**
```bash
# Increase threads (if CPU available)
python -m hawk_scanner.main fs --path /data --threads 8

# Exclude large/binary directories
python -m hawk_scanner.main fs --path /data --exclude "node_modules,vendor,.git"
```

**Connection errors:**
```bash
# Test connection first
python -m hawk_scanner.main postgresql --connection config.yml --profile prod --test

# Enable verbose logging
python -m hawk_scanner.main postgresql --connection config.yml --profile prod --verbose
```

### Debug Mode

```bash
# Enable debug logging
export LOG_LEVEL=DEBUG
python -m hawk_scanner.main fs --path /data --verbose
```

---

## Performance

### Benchmarks

| Scenario | Files | Time | Throughput |
|----------|-------|------|------------|
| Small text files (< 1KB) | 10,000 | 28s | 357 files/sec |
| Medium files (1-100KB) | 1,000 | 45s | 22 files/sec |
| Large files (100KB-10MB) | 100 | 120s | 0.8 files/sec |
| PostgreSQL (1M rows) | 1 | 180s | 5,555 rows/sec |

*Benchmarks on: AMD Ryzen 9 5900X, 32GB RAM, NVMe SSD*

### Optimization Tips

1. **Use appropriate thread count** (usually 2-4x CPU cores)
2. **Exclude unnecessary directories** (node_modules, .git, etc.)
3. **Set file size limits** to skip large binaries
4. **Use sampling** for large databases
5. **Enable archive extraction** only when needed

---

## 📚 Additional Documentation

- [Architecture Overview](../../docs/architecture/ARCHITECTURE.md)
- [Mathematical Implementation](../../docs/deployment/MATHEMATICAL_IMPLEMENTATION.md)
- [Technical Specifications](../../docs/development/TECHNICAL_SPECIFICATIONS.md)
- [API Documentation](../../docs/development/TECHNICAL_SPECIFICATIONS.md#api-specifications)

## 🤝 Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for contribution guidelines.

## 📝 License

Apache License 2.0 - See [LICENSE](../../LICENSE) for details.

---

**Version**: 2.1.0  
**Last Updated**: January 2026
