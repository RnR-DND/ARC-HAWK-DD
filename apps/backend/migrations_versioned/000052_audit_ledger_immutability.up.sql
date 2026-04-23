-- Enforce immutability on audit tables (DPDPA compliance evidence)
-- audit_ledger
CREATE RULE no_update_audit_ledger AS ON UPDATE TO audit_ledger DO INSTEAD NOTHING;
CREATE RULE no_delete_audit_ledger AS ON DELETE TO audit_ledger DO INSTEAD NOTHING;

-- audit_logs (hash-chained)
CREATE RULE no_update_audit_logs AS ON UPDATE TO audit_logs DO INSTEAD NOTHING;
CREATE RULE no_delete_audit_logs AS ON DELETE TO audit_logs DO INSTEAD NOTHING;
