# Runbook: Expired Cloud Credentials

## Trigger

A scan job targeting a cloud data source (AWS S3, GCS, Azure Blob, Snowflake, BigQuery, Redshift, etc.) fails because the configured credentials are expired, revoked, or have insufficient permissions.

Activates when the scanner logs `error_code: CREDENTIALS_EXPIRED` or `error_code: AUTH_FAILED`, or when a connection shows `connection_status: auth_error` in the dashboard.

---

## Symptoms

**Logs (scanner worker):**
```
ERROR  hawk_scanner  source=s3  bucket=prod-data-lake  error="ExpiredTokenException: The security token included in the request is expired"
ERROR  hawk_scanner  source=snowflake  account=myorg.us-east-1  error="390114: Snowflake token is no longer valid"
```

**Temporal UI:** The scan workflow activity `ScanDataSourceActivity` shows `ActivityTaskFailed` with the auth error message. Temporal will retry automatically according to the retry policy.

**Dashboard (Connections page):** The connection record shows:
- `last_health_check_status: failed`
- `error_message: authentication failed`
- Connection status badge shows red.

**Metrics:** `hawk_scanner_auth_failures_total{source="s3"}` counter increments.

---

## Automated Response

The scanner catches the authentication exception, logs it with the connection ID and source type, and marks the connection `health_status = 'auth_error'`. The scan job for that connection is failed (not retried — retrying with the same invalid credential is pointless).

Other connections in the same scan run are unaffected and continue normally.

---

## Manual Steps

1. **Identify the failing connection** from the dashboard (Connections > filter by `auth_error`) or from the scan error log. Note the connection ID and source type.

2. **Obtain fresh credentials** from the appropriate source:

   **AWS S3 / IAM role (recommended):**
   ```bash
   # If using IAM roles, the issue is likely the instance role or assumed role chain.
   # Check the role ARN configured for the connection:
   aws sts get-caller-identity  # on the host running the scanner
   aws sts assume-role --role-arn arn:aws:iam::123456789012:role/HawkScanRole \
     --role-session-name test
   ```

   **AWS access keys:**
   - Rotate the key in the AWS IAM console.
   - Generate new `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.

   **GCS:**
   - Regenerate the service account JSON key from Google Cloud Console or:
     ```bash
     gcloud iam service-accounts keys create new-key.json \
       --iam-account hawk-scanner@myproject.iam.gserviceaccount.com
     ```

   **Azure Blob:**
   - Rotate the Storage Account key or regenerate the SAS token in the Azure Portal.
   - Or rotate the service principal client secret in Entra ID.

   **Snowflake:**
   ```sql
   ALTER USER hawk_scanner_user SET PASSWORD = 'new-password';
   -- or rotate the RSA key pair if using key-pair authentication
   ```

3. **Update the connection credentials** in ARC-Hawk:
   ```bash
   curl -X PUT http://localhost:8080/api/v1/connections/{connection_id} \
     -H "Content-Type: application/json" \
     -d '{
       "credentials": {
         "access_key_id": "AKIA...",
         "secret_access_key": "..."
       }
     }'
   ```
   Or use the UI: Connections > Edit > update the credential fields > Save.

4. **Test the connection** using the health check endpoint:
   ```bash
   curl -X POST http://localhost:8080/api/v1/connections/{connection_id}/test
   ```
   Expect `{"status": "ok"}` in the response.

5. **Trigger a new scan** for the affected connection:
   ```bash
   curl -X POST http://localhost:8080/api/v1/scan/trigger \
     -H "Content-Type: application/json" \
     -d '{"connection_id": "{connection_id}"}'
   ```

6. **Delete the old (expired) key material** from the cloud provider's console once the new credentials are confirmed working.

---

## Resolution Criteria

- The connection health check returns `status: ok`.
- A scan job for the affected connection completes successfully.
- The expired credentials have been revoked/deleted in the cloud provider.

---

## Prevention

- **Credential rotation reminders:** Configure alerts when credentials approach their expiry. Most cloud providers support credential expiry metadata — set a reminder 7 days before expiry.
- **Use short-lived credentials with automatic rotation:** Prefer IAM roles (AWS), Workload Identity (GCP), or Managed Identities (Azure) over long-lived access keys.
- **Store credentials in a secrets manager:** Use AWS Secrets Manager, HashiCorp Vault, or GCP Secret Manager with automatic rotation policies. The scanner can be extended to fetch credentials at scan time rather than storing them in the database.
- **Monitor `auth_error` connections:** Set an alert on `hawk_scanner_auth_failures_total > 0` and on any connection with `health_status = 'auth_error'` for more than 1 hour.
- **Periodic connection health checks:** Configure the backend to run connection health checks on a schedule (e.g., daily) independent of scan runs, so credential expiry is detected before the next scheduled scan.
