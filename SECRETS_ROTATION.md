# Secrets Rotation Checklist

## CRITICAL: Supermemory API Key (sm_XXDSFvG3...)

A Supermemory API key prefix was exposed in .env comment history. The full key may have been committed previously.

### Action Required:
1. Log into https://app.supermemory.ai
2. Navigate to Settings → API Keys
3. Revoke the key starting with `sm_XXDSFvG3`
4. Generate a new key
5. Update the key in your secrets manager / deployment pipeline
6. Verify old key is dead: `curl -H "Authorization: Bearer sm_XXDSFvG3..." https://api.supermemory.ai/v1/memories` should return 401

### Other secrets requiring rotation if deployed to shared environments:
- VAULT_DEV_ROOT_TOKEN (dev-root-token-arc-hawk) — replace with AppRole auth
- SCANNER_SERVICE_TOKEN (dev-scanner-token-change-me) — set per-environment secrets
- CI_NEO4J_PASSWORD — set as GitHub secret via: `gh secret set CI_NEO4J_PASSWORD`
