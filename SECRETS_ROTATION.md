# Secrets Rotation Checklist

## CRITICAL: Supermemory API Key (sm_XXDSFvG3...)

A Supermemory API key prefix was exposed in .env comment history.

### Action Required:
1. Log into https://app.supermemory.ai
2. Navigate to Settings → API Keys
3. Revoke the key starting with `sm_XXDSFvG3`
4. Generate a new key
5. Update in your secrets manager / deployment pipeline
6. Verify old key is dead (should return 401 after revocation)

### Other secrets requiring rotation before staging/production:
- VAULT_DEV_ROOT_TOKEN (dev-root-token-arc-hawk) — switch to AppRole auth
- SCANNER_SERVICE_TOKEN (dev-scanner-token-change-me) — set per-environment via secrets manager
- CI Neo4j password — set as GitHub secret: `gh secret set CI_NEO4J_PASSWORD`
