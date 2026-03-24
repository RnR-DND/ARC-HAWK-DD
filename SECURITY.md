# Security Policy

## Supported Versions

The following versions of ARC-Hawk are currently supported with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 2.1.x   | :white_check_mark: |
| 2.0.x   | :white_check_mark: |
| 1.0.x   | :x:                |
| < 1.0   | :x:                |

---

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability, please follow these steps:

### 1. Do Not Open a Public Issue

Security vulnerabilities should **not** be reported through public GitHub issues, discussions, or pull requests.

### 2. Email the Security Team

Send an email to **security@arc-hawk.io** with:

- **Subject**: `[SECURITY] Brief description of the issue`
- **Description**: Detailed description of the vulnerability
- **Steps to Reproduce**: Clear steps to reproduce the issue
- **Impact**: Potential impact and severity assessment
- **Proof of Concept**: If applicable, include a minimal proof of concept
- **Your Contact**: How we can reach you for follow-up questions

### 3. Response Timeline

- **Acknowledgment**: Within 48 hours
- **Initial Assessment**: Within 5 business days
- **Fix Timeline**: Based on severity:
  - Critical: 7 days
  - High: 14 days
  - Medium: 30 days
  - Low: 90 days

### 4. Disclosure Policy

We follow responsible disclosure:

1. We will acknowledge receipt of your report
2. We will investigate and validate the vulnerability
3. We will develop and test a fix
4. We will release the fix and publicly disclose the vulnerability (with credit if desired)
5. We will update this document with CVE information if applicable

---

## Security Features

### Current Security Measures

#### 1. Mathematical Validation

All PII detection uses mathematically proven validation algorithms:

- **Aadhaar**: Verhoeff algorithm (100% accuracy)
- **Credit Cards**: Luhn algorithm (100% accuracy)
- **PAN**: Weighted Modulo 26 (100% accuracy)
- **Passport**: Checksum validation

This prevents false positives and ensures only valid PII is flagged.

#### 2. Data Privacy

- **No Data Exfiltration**: All processing happens locally
- **No External APIs**: No data sent to third-party services
- **Configurable Retention**: Automatic data retention policies
- **Encryption at Rest**: Database encryption supported

#### 3. Input Validation

- **Strict Type Checking**: All inputs validated against schemas
- **SQL Injection Prevention**: Parameterized queries throughout
- **XSS Prevention**: Output encoding in frontend
- **File Upload Validation**: Type and size validation for uploads

#### 4. Infrastructure Security

- **Container Security**: Minimal base images, non-root users
- **Secret Management**: Environment variables for secrets
- **Network Isolation**: Internal service communication
- **Resource Limits**: Container resource constraints

### Security Limitations

> **Important**: The following security features are **not yet implemented** (see [TODO.md](./TODO.md)):

- [ ] Authentication & Authorization (JWT/RBAC)
- [ ] API Rate Limiting
- [ ] Audit Logging (partial)
- [ ] TLS/SSL enforcement (relies on reverse proxy)

**Do not deploy to production without implementing authentication.**

---

## Security Best Practices

### Deployment

#### 1. Use Environment Variables for Secrets

**✅ Good:**
```bash
# .env file (not committed to git)
DB_PASSWORD=your_secure_password
JWT_SECRET=your_jwt_secret
API_KEY=your_api_key
```

**❌ Bad:**
```go
// Hardcoded in source code
dbPassword := "password123"
```

#### 2. Use Docker Secrets (Production)

```yaml
# docker-compose.yml (production)
version: '3.8'

secrets:
  db_password:
    external: true
  
services:
  backend:
    secrets:
      - db_password
    environment:
      - DB_PASSWORD_FILE=/run/secrets/db_password
```

#### 3. Enable TLS/SSL

Use a reverse proxy (nginx, traefik) with TLS termination:

```nginx
# nginx.conf
server {
    listen 443 ssl;
    server_name arc-hawk.yourdomain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

#### 4. Database Security

- Use strong passwords
- Enable SSL connections
- Limit database user privileges
- Enable query logging
- Regular backups

#### 5. Network Security

- Use internal Docker networks
- Expose only necessary ports
- Use firewall rules
- Implement network segmentation

### Development

#### 1. Never Commit Secrets

Add to `.gitignore`:
```
.env
.env.local
*.pem
*.key
secrets/
config/secrets.yml
```

#### 2. Use Pre-commit Hooks

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    hooks:
      - id: detect-private-key
      - id: check-added-large-files
  - repo: https://github.com/Yelp/detect-secrets
    hooks:
      - id: detect-secrets
```

#### 3. Dependency Scanning

Run security scans on dependencies:

```bash
# Go
cd apps/backend
go list -json -m all | nancy sleuth

# Node.js
cd apps/frontend
npm audit

# Python
cd apps/scanner
pip-audit
```

#### 4. Code Reviews

All code changes should be reviewed for:
- Hardcoded credentials
- SQL injection vulnerabilities
- XSS vulnerabilities
- Insecure deserialization
- Race conditions

---

## Security Checklist

### Pre-Deployment Checklist

- [ ] Changed all default passwords
- [ ] Moved secrets to environment variables
- [ ] Enabled TLS/SSL
- [ ] Configured firewall rules
- [ ] Set up log monitoring
- [ ] Implemented backup strategy
- [ ] Reviewed user permissions
- [ ] Tested disaster recovery

### Development Checklist

- [ ] No secrets in code
- [ ] Input validation implemented
- [ ] Error messages don't leak sensitive info
- [ ] Dependencies up to date
- [ ] Security tests passing
- [ ] Code reviewed by another developer

---

## Incident Response

### If You Suspect a Security Incident

1. **Don't Panic**: Stay calm and document everything
2. **Isolate**: Isolate affected systems if possible
3. **Document**: Record what happened, when, and what was affected
4. **Notify**: Contact the security team immediately
5. **Preserve**: Preserve logs and evidence
6. **Assess**: Determine scope and impact
7. **Remediate**: Fix the vulnerability
8. **Review**: Conduct post-incident review

### Contact Information

- **Security Team**: security@arc-hawk.io
- **Emergency**: +1-XXX-XXX-XXXX (24/7)
- **PGP Key**: [Download Public Key](https://arc-hawk.io/security/pgp-key.asc)

---

## Security-Related Configuration

### Backend Environment Variables

```bash
# Required for security
JWT_SECRET=your_jwt_secret_min_32_chars
ENCRYPTION_KEY=your_encryption_key

# Optional security settings
SECURE_COOKIES=true
TRUSTED_PROXIES=10.0.0.0/8
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60
```

### Scanner Security Settings

```yaml
# config/security.yml
scanning:
  max_file_size: 104857600  # 100MB
  excluded_paths:
    - .git
    - node_modules
    - vendor
  max_depth: 10
  
validation:
  strict_mode: true
  min_confidence: 0.8
  
output:
  encrypt_results: true
  output_format: encrypted_json
```

---

## Compliance

### Certifications

ARC-Hawk is designed to help organizations meet compliance requirements for:

- **DPDPA 2023** (India) - ✅ Implemented
- **GDPR** (EU) - 🔄 In Progress
- **CCPA** (California) - 🔄 Planned
- **HIPAA** (Healthcare) - 🔄 Planned

### Data Handling

- **Data Minimization**: Only collect necessary data
- **Purpose Limitation**: Data used only for stated purposes
- **Storage Limitation**: Automatic data retention policies
- **Integrity**: Data validation and checksums
- **Confidentiality**: Encryption and access controls

---

## Security Roadmap

See [TODO.md](./TODO.md) for detailed security implementation status.

### Q1 2026

- [ ] Authentication & Authorization (JWT/RBAC)
- [ ] API Rate Limiting
- [ ] Comprehensive Audit Logging
- [ ] Security Headers (CSP, HSTS, etc.)

### Q2 2026

- [ ] OAuth 2.0 / OIDC Support
- [ ] Multi-Factor Authentication
- [ ] Automated Security Scanning
- [ ] Penetration Testing

### Q3 2026

- [ ] SOC 2 Compliance
- [ ] ISO 27001 Certification
- [ ] Bug Bounty Program
- [ ] Security Whitepaper

---

## Acknowledgments

We thank the following security researchers for responsible disclosure:

- [Researcher Name] - [Vulnerability Type] - [Date]

---

## License

This security policy is part of the ARC-Hawk project and is licensed under [Apache License 2.0](./LICENSE).

---

**Last Updated**: January 2026  
**Version**: 1.0  
**Next Review**: April 2026
