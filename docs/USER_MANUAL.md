# ARC-Hawk User Manual 🦅

## Introduction
ARC-Hawk is your command center for Data Privacy. It allows you to scan, visualize, and secure sensitive data across your organization.

## 🚀 Getting Started

1.  **Access Dashboard**: Open `http://localhost:3000`.
2.  **Check Status**: Ensure all System Health indicators are Green on the Compliance page.

---

## 🖥️ Dashboard Features

### 1. Risk Summary
The homepage provides a high-level view of your risk posture:
- **Total Findings**: Count of unaddressed PII instances.
- **Critical Assets**: Files/Tables with High Severity PII.
- **Scan Status**: Live progress of active scans.

### 2. Findings Explorer
A detailed grid view of all detected PII.
- **Filters**: Filter by Status (Active/Remediated), Asset, Risk Level.
- **Actions**:
    - **Mark False Positive**: If the detection is incorrect.
    - **Remediate**: Launch a Masking/Deletion job.

### 3. Lineage Graph
Visual map of data flow.
- **Nodes**: Blue (System), Green (Asset), Red (PII).
- **Interactions**: Click on nodes to see detailed metadata.

### 4. Remediation Center
Track the status of fix requests.
- **History**: See who remediated what and when.
- **Retry**: Re-run failed remediation jobs.

---

## ⚙️ Configuration

### Adding Data Sources
1.  Click **"Add Source"** in the top-right.
2.  Select Type (S3, Postgres, GCS, etc.).
3.  Enter Credentials.
4.  Click **"Test Connection"** before saving.

### Managing Scans
- **Ad-hoc Scan**: Trigger manually from the Asset details page.
- **Scheduled Scan**: Configure via the Backend API (Cron support coming soon).

---

## 🆘 Support

For issues, please refer to the [Troubleshooting Guide](FAILURE_MODES.md) or contact your administrator.
