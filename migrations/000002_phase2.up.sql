-- AgentMesh Phase 2 PostgreSQL Schema Migration
-- Migration 000002_phase2.up.sql

-- 1. Agent Graphs (ADK & Workflow Topologies)
CREATE TABLE IF NOT EXISTS agent_graphs (
    graph_id VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id VARCHAR(128) NOT NULL,
    project_id VARCHAR(128) DEFAULT '',
    version VARCHAR(32) NOT NULL DEFAULT '1.0.0',
    graph_hash VARCHAR(64) NOT NULL,
    schema_version VARCHAR(32) NOT NULL DEFAULT 'agentmesh.dev/v1alpha1',
    entrypoint VARCHAR(128) NOT NULL,
    exitpoints TEXT[] DEFAULT '{}',
    nodes_json JSONB NOT NULL DEFAULT '[]',
    edges_json JSONB NOT NULL DEFAULT '[]',
    tools_json JSONB NOT NULL DEFAULT '[]',
    delegations_json JSONB NOT NULL DEFAULT '[]',
    approval_points_json JSONB NOT NULL DEFAULT '[]',
    model_calls_json JSONB NOT NULL DEFAULT '[]',
    external_calls_json JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, graph_id)
);

-- 2. Capabilities & Evidence
CREATE TABLE IF NOT EXISTS capabilities (
    id VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, id)
);

CREATE TABLE IF NOT EXISTS agent_capabilities (
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id VARCHAR(128) NOT NULL,
    capability_id VARCHAR(128) NOT NULL,
    evidence_tier VARCHAR(32) NOT NULL DEFAULT 'DECLARED_CAPABILITY',
    confidence NUMERIC(5, 4) DEFAULT 0.5000,
    sample_count INT DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, agent_id, capability_id)
);

-- 3. MCP Tool Governance & Tool Passports
CREATE TABLE IF NOT EXISTS tool_fingerprints (
    id VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    server VARCHAR(128) NOT NULL,
    tool_name VARCHAR(128) NOT NULL,
    version VARCHAR(32) NOT NULL DEFAULT '1.0.0',
    provider VARCHAR(64) NOT NULL DEFAULT 'custom',
    risk_class VARCHAR(32) NOT NULL DEFAULT 'UNKNOWN',
    schema_hash VARCHAR(64) NOT NULL,
    schema_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, id)
);

CREATE TABLE IF NOT EXISTS tool_passports (
    tool_id VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    tool_name VARCHAR(128) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    server VARCHAR(128) NOT NULL,
    risk_class VARCHAR(32) NOT NULL DEFAULT 'UNKNOWN',
    schema_fingerprint VARCHAR(64) NOT NULL,
    health_status VARCHAR(32) NOT NULL DEFAULT 'HEALTHY',
    sample_count INT DEFAULT 0,
    success_rate NUMERIC(5, 4) DEFAULT 1.0000,
    p95_latency_ms INT DEFAULT 0,
    policy_coverage VARCHAR(32) DEFAULT 'GOVERNED',
    last_evaluated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, tool_id)
);

-- 4. A2A Compatibility Lab Profiles
CREATE TABLE IF NOT EXISTS a2a_profiles (
    id VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id VARCHAR(128) NOT NULL,
    version VARCHAR(32) NOT NULL,
    protocol_version VARCHAR(32) NOT NULL DEFAULT '0.3.0',
    status VARCHAR(32) NOT NULL DEFAULT 'UNTESTED',
    test_results_json JSONB NOT NULL DEFAULT '{}',
    tester_version VARCHAR(32) NOT NULL DEFAULT '1.0.0',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, id)
);

-- 5. Routing Outcome History & Regret Analysis
CREATE TABLE IF NOT EXISTS route_outcomes (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    request_class VARCHAR(64) NOT NULL,
    capability VARCHAR(128) NOT NULL,
    selected_agent VARCHAR(128) NOT NULL,
    alternatives_json JSONB NOT NULL DEFAULT '[]',
    reason TEXT NOT NULL,
    success BOOLEAN DEFAULT TRUE,
    latency_ms INT DEFAULT 0,
    cost_usd NUMERIC(10, 6) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 6. Evaluations & Baselines
CREATE TABLE IF NOT EXISTS evaluation_suites (
    id VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    capability VARCHAR(128) NOT NULL,
    test_cases_json JSONB NOT NULL DEFAULT '[]',
    constraints_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, id)
);

CREATE TABLE IF NOT EXISTS evaluation_results (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    suite_id VARCHAR(128) NOT NULL,
    agent_id VARCHAR(128) NOT NULL,
    agent_version VARCHAR(32) NOT NULL,
    quality_score NUMERIC(5, 4) DEFAULT 1.0000,
    latency_p95_ms INT DEFAULT 0,
    cost_usd NUMERIC(10, 6) DEFAULT 0,
    policy_pass BOOLEAN DEFAULT TRUE,
    tool_success_rate NUMERIC(5, 4) DEFAULT 1.0000,
    provenance_json JSONB NOT NULL DEFAULT '{}',
    is_baseline BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 7. Policy Canaries & Change Reports
CREATE TABLE IF NOT EXISTS policy_canaries (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    policy_id VARCHAR(128) NOT NULL,
    candidate_version VARCHAR(32) NOT NULL,
    baseline_version VARCHAR(32) NOT NULL,
    shadow_mode BOOLEAN DEFAULT TRUE,
    traffic_eval_count INT DEFAULT 0,
    would_allow_count INT DEFAULT 0,
    would_deny_count INT DEFAULT 0,
    would_approve_count INT DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS agent_change_reports (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id VARCHAR(128) NOT NULL,
    current_version VARCHAR(32) NOT NULL,
    candidate_version VARCHAR(32) NOT NULL,
    new_tools TEXT[] DEFAULT '{}',
    removed_tools TEXT[] DEFAULT '{}',
    new_delegations TEXT[] DEFAULT '{}',
    security_flags_json JSONB NOT NULL DEFAULT '[]',
    safe_to_canary BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 8. Proxy Fleet
CREATE TABLE IF NOT EXISTS proxy_instances (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    region VARCHAR(64) NOT NULL DEFAULT 'us-central1',
    version VARCHAR(32) NOT NULL DEFAULT '1.0.0',
    active_config_hash VARCHAR(64) NOT NULL DEFAULT '',
    health_status VARCHAR(32) NOT NULL DEFAULT 'HEALTHY',
    last_heartbeat TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for Phase 2 Performance
CREATE INDEX IF NOT EXISTS idx_agent_graphs_agent ON agent_graphs(tenant_id, agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_capabilities_cap ON agent_capabilities(tenant_id, capability_id);
CREATE INDEX IF NOT EXISTS idx_tool_passports_risk ON tool_passports(tenant_id, risk_class);
CREATE INDEX IF NOT EXISTS idx_route_outcomes_cap ON route_outcomes(tenant_id, capability);
CREATE INDEX IF NOT EXISTS idx_evaluation_results_agent ON evaluation_results(tenant_id, agent_id, agent_version);
CREATE INDEX IF NOT EXISTS idx_proxy_instances_tenant ON proxy_instances(tenant_id, health_status);
