-- AgentMesh Phase 3 PostgreSQL Schema Migration
-- Migration 000003_phase3.up.sql

-- 1. Task Fingerprints
CREATE TABLE IF NOT EXISTS task_fingerprints (
    fingerprint_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    capability VARCHAR(128) NOT NULL,
    input_size_class VARCHAR(32) NOT NULL DEFAULT 'SMALL',
    output_size_class VARCHAR(32) NOT NULL DEFAULT 'SMALL',
    streaming BOOLEAN NOT NULL DEFAULT FALSE,
    required_tools TEXT[] DEFAULT '{}',
    data_classification VARCHAR(64) NOT NULL DEFAULT 'INTERNAL',
    target_region VARCHAR(64) DEFAULT 'us-central1',
    max_latency_ms BIGINT DEFAULT 5000,
    max_cost_usd NUMERIC(10, 6) DEFAULT 0.10,
    delegation_allowed BOOLEAN NOT NULL DEFAULT TRUE,
    model_constraints TEXT[] DEFAULT '{}',
    complexity_class VARCHAR(32) NOT NULL DEFAULT 'STANDARD',
    structured_output BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, fingerprint_id)
);

-- 2. Canonical Routing Outcomes V3
CREATE TABLE IF NOT EXISTS routing_outcomes_v3 (
    outcome_id VARCHAR(64) NOT NULL,
    organization_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id VARCHAR(128) DEFAULT '',
    task_id VARCHAR(128) NOT NULL,
    capability_id VARCHAR(128) NOT NULL,
    selected_agent_id VARCHAR(128) NOT NULL,
    selected_agent_version VARCHAR(32) NOT NULL,
    candidate_agents TEXT[] DEFAULT '{}',
    routing_objective VARCHAR(32) NOT NULL DEFAULT 'BALANCED',
    policy_version VARCHAR(32) NOT NULL DEFAULT 'v1',
    route_algorithm_version VARCHAR(64) NOT NULL,
    route_confidence NUMERIC(5, 4) DEFAULT 1.0000,
    request_features_json JSONB NOT NULL DEFAULT '{}',
    start_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    end_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    success BOOLEAN NOT NULL DEFAULT TRUE,
    failure_type VARCHAR(32) NOT NULL DEFAULT 'NONE',
    latency_ms BIGINT DEFAULT 0,
    cost NUMERIC(10, 6) DEFAULT 0,
    tool_success BOOLEAN NOT NULL DEFAULT TRUE,
    delegation_success BOOLEAN NOT NULL DEFAULT TRUE,
    evaluation_signal NUMERIC(5, 4) DEFAULT 1.0000,
    human_escalation BOOLEAN NOT NULL DEFAULT FALSE,
    quality_signal NUMERIC(5, 4) DEFAULT 1.0000,
    trace_id VARCHAR(128) DEFAULT '',
    schema_version VARCHAR(32) NOT NULL DEFAULT 'agentmesh.dev/v3alpha1',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organization_id, outcome_id)
);

-- 3. Operational Outcome Graph Nodes & Edges
CREATE TABLE IF NOT EXISTS operational_outcome_nodes (
    node_id VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    node_type VARCHAR(64) NOT NULL,
    properties_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, node_id)
);

CREATE TABLE IF NOT EXISTS operational_outcome_edges (
    edge_id VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    edge_type VARCHAR(64) NOT NULL,
    from_id VARCHAR(128) NOT NULL,
    to_id VARCHAR(128) NOT NULL,
    properties_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, edge_id)
);

-- 4. Reliability Profiles
CREATE TABLE IF NOT EXISTS reliability_profiles (
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id VARCHAR(128) NOT NULL,
    capability_id VARCHAR(128) NOT NULL,
    version VARCHAR(32) NOT NULL DEFAULT '1.0.0',
    total_samples BIGINT DEFAULT 0,
    overall_success_rate NUMERIC(5, 4) DEFAULT 1.0000,
    p50_latency_ms BIGINT DEFAULT 0,
    p95_latency_ms BIGINT DEFAULT 0,
    p99_latency_ms BIGINT DEFAULT 0,
    average_cost_usd NUMERIC(10, 6) DEFAULT 0,
    max_cost_usd NUMERIC(10, 6) DEFAULT 0,
    tool_success_rate NUMERIC(5, 4) DEFAULT 1.0000,
    timeout_rate NUMERIC(5, 4) DEFAULT 0,
    confidence VARCHAR(32) NOT NULL DEFAULT 'COLD_START',
    incident_active BOOLEAN NOT NULL DEFAULT FALSE,
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, agent_id, capability_id)
);

-- 5. Agent SLOs & Tracking
CREATE TABLE IF NOT EXISTS agent_slos (
    id VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id VARCHAR(128) NOT NULL,
    capability_id VARCHAR(128) NOT NULL,
    target_success_rate NUMERIC(5, 4) DEFAULT 0.9900,
    max_p95_latency_ms BIGINT DEFAULT 4000,
    max_cost_usd NUMERIC(10, 6) DEFAULT 0.100000,
    target_tool_success NUMERIC(5, 4) DEFAULT 0.9950,
    current_status VARCHAR(32) NOT NULL DEFAULT 'UNKNOWN',
    remaining_error_budget NUMERIC(5, 4) DEFAULT 1.0000,
    last_evaluated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, id)
);

-- 6. Private Proxy Fleet
CREATE TABLE IF NOT EXISTS proxy_fleet_instances (
    instance_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    cluster VARCHAR(128) NOT NULL,
    region VARCHAR(64) NOT NULL,
    runtime_type VARCHAR(32) NOT NULL DEFAULT 'GKE',
    proxy_version VARCHAR(32) NOT NULL,
    active_config_hash VARCHAR(64) NOT NULL,
    health VARCHAR(32) NOT NULL DEFAULT 'HEALTHY',
    last_heartbeat TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    outbound_only BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (tenant_id, instance_id)
);

-- 7. Learned Routing Model Registry
CREATE TABLE IF NOT EXISTS routing_models (
    model_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    version VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'CANDIDATE',
    dataset_size INT DEFAULT 0,
    supported_capabilities TEXT[] DEFAULT '{}',
    feature_weights_json JSONB NOT NULL DEFAULT '{}',
    accuracy_score NUMERIC(5, 4) DEFAULT 0.0000,
    cost_reduction_pct NUMERIC(5, 2) DEFAULT 0.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    promoted_at TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (tenant_id, model_id)
);

-- Performance Indexes for High-Scale Outcome Queries
CREATE INDEX IF NOT EXISTS idx_routing_outcomes_v3_tenant_cap ON routing_outcomes_v3(organization_id, capability_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_routing_outcomes_v3_agent ON routing_outcomes_v3(organization_id, selected_agent_id, success);
CREATE INDEX IF NOT EXISTS idx_operational_nodes_type ON operational_outcome_nodes(tenant_id, node_type);
CREATE INDEX IF NOT EXISTS idx_operational_edges_from ON operational_outcome_edges(tenant_id, from_id);
CREATE INDEX IF NOT EXISTS idx_operational_edges_to ON operational_outcome_edges(tenant_id, to_id);
CREATE INDEX IF NOT EXISTS idx_reliability_profiles_agent ON reliability_profiles(tenant_id, agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_slos_status ON agent_slos(tenant_id, current_status);
CREATE INDEX IF NOT EXISTS idx_proxy_fleet_tenant_health ON proxy_fleet_instances(tenant_id, health);
CREATE INDEX IF NOT EXISTS idx_routing_models_tenant_status ON routing_models(tenant_id, status);
