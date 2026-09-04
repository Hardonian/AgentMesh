-- AgentMesh Migration 000005: Phase 5 Security Hardening, Row Level Security (RLS) & Data Integrity Constraints

-- 1. Enable Row Level Security (RLS) on all tenant-owned tables
ALTER TABLE IF EXISTS agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS policies ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS tools ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS credentials ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS approvals ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS canaries ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS agent_graphs ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS tool_passports ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS a2a_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS evaluation_suites ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS route_outcomes ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS agent_slos ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS proxy_instances ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS routing_models ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS agent_optimization_actions ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS action_workflows ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS routing_specs ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS production_outcomes ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS automation_policies ENABLE ROW LEVEL SECURITY;

-- 2. Define Tenant Isolation Policies (Default-Deny when tenant variable is unset)
DO $$
BEGIN
    -- agents
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'agents' AND policyname = 'tenant_isolation_agents') THEN
        CREATE POLICY tenant_isolation_agents ON agents
            FOR ALL USING (tenant_id = current_setting('app.current_tenant', true))
            WITH CHECK (tenant_id = current_setting('app.current_tenant', true));
    END IF;

    -- policies
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'policies' AND policyname = 'tenant_isolation_policies') THEN
        CREATE POLICY tenant_isolation_policies ON policies
            FOR ALL USING (tenant_id = current_setting('app.current_tenant', true))
            WITH CHECK (tenant_id = current_setting('app.current_tenant', true));
    END IF;

    -- credentials
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'credentials' AND policyname = 'tenant_isolation_credentials') THEN
        CREATE POLICY tenant_isolation_credentials ON credentials
            FOR ALL USING (tenant_id = current_setting('app.current_tenant', true))
            WITH CHECK (tenant_id = current_setting('app.current_tenant', true));
    END IF;

    -- agent_optimization_actions
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'agent_optimization_actions' AND policyname = 'tenant_isolation_opt_actions') THEN
        CREATE POLICY tenant_isolation_opt_actions ON agent_optimization_actions
            FOR ALL USING (organization_id = current_setting('app.current_tenant', true))
            WITH CHECK (organization_id = current_setting('app.current_tenant', true));
    END IF;

    -- action_workflows
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'action_workflows' AND policyname = 'tenant_isolation_action_workflows') THEN
        CREATE POLICY tenant_isolation_action_workflows ON action_workflows
            FOR ALL USING (organization_id = current_setting('app.current_tenant', true))
            WITH CHECK (organization_id = current_setting('app.current_tenant', true));
    END IF;

    -- routing_specs
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'routing_specs' AND policyname = 'tenant_isolation_routing_specs') THEN
        CREATE POLICY tenant_isolation_routing_specs ON routing_specs
            FOR ALL USING (organization_id = current_setting('app.current_tenant', true))
            WITH CHECK (organization_id = current_setting('app.current_tenant', true));
    END IF;

    -- production_outcomes
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'production_outcomes' AND policyname = 'tenant_isolation_prod_outcomes') THEN
        CREATE POLICY tenant_isolation_prod_outcomes ON production_outcomes
            FOR ALL USING (organization_id = current_setting('app.current_tenant', true))
            WITH CHECK (organization_id = current_setting('app.current_tenant', true));
    END IF;
END $$;

-- 3. Data Integrity & Non-Empty Constraints
ALTER TABLE IF EXISTS agents ADD CONSTRAINT chk_agents_tenant_nonempty CHECK (length(trim(tenant_id)) > 0);
ALTER TABLE IF EXISTS policies ADD CONSTRAINT chk_policies_tenant_nonempty CHECK (length(trim(tenant_id)) > 0);
ALTER TABLE IF EXISTS credentials ADD CONSTRAINT chk_credentials_tenant_nonempty CHECK (length(trim(tenant_id)) > 0);
ALTER TABLE IF EXISTS agent_optimization_actions ADD CONSTRAINT chk_opt_actions_org_nonempty CHECK (length(trim(organization_id)) > 0);
ALTER TABLE IF EXISTS routing_specs ADD CONSTRAINT chk_routing_specs_org_nonempty CHECK (length(trim(organization_id)) > 0);

-- 4. Schema Enhancements: LastUsedAt for credential rotation audit
ALTER TABLE IF EXISTS credentials ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX IF NOT EXISTS idx_credentials_tenant_subject ON credentials (tenant_id, subject_id);
