-- AgentMesh Core PostgreSQL Schema
-- Migration 000001_init.up.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Organizations / Tenants
CREATE TABLE IF NOT EXISTS organizations (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Agents
CREATE TABLE IF NOT EXISTS agents (
    id VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'REGISTERED',
    contract_hash VARCHAR(64) NOT NULL,
    contract_json JSONB NOT NULL,
    passport_json JSONB,
    endpoint_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, id)
);

-- Policies
CREATE TABLE IF NOT EXISTS policies (
    id VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(32) NOT NULL,
    rules_json JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, id)
);

-- Tools
CREATE TABLE IF NOT EXISTS tools (
    id VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    server VARCHAR(128),
    risk_class VARCHAR(32) DEFAULT 'LOW',
    data_classification VARCHAR(32) DEFAULT 'INTERNAL',
    approval_required BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, id)
);

-- Credentials / API Keys
CREATE TABLE IF NOT EXISTS credentials (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    subject_id VARCHAR(128) NOT NULL,
    key_prefix VARCHAR(32) NOT NULL,
    hashed_key VARCHAR(128) NOT NULL UNIQUE,
    scopes TEXT[] NOT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Approvals (HITL)
CREATE TABLE IF NOT EXISTS approvals (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id VARCHAR(128) NOT NULL,
    tool VARCHAR(128) NOT NULL,
    action VARCHAR(64) NOT NULL,
    parameters_json JSONB NOT NULL,
    parameters_hash VARCHAR(64) NOT NULL,
    policy_id VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    approval_token VARCHAR(128),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by VARCHAR(128),
    resolution_note TEXT
);

-- Audit Trail (Append Only)
CREATE TABLE IF NOT EXISTS audit_events (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    type VARCHAR(64) NOT NULL,
    actor VARCHAR(128) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    details_json JSONB,
    prev_hash VARCHAR(64) NOT NULL,
    hash VARCHAR(64) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Traces
CREATE TABLE IF NOT EXISTS traces (
    trace_id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    root_agent_id VARCHAR(128) NOT NULL,
    task_id VARCHAR(128),
    status VARCHAR(32) NOT NULL,
    total_cost_usd NUMERIC(10, 6) DEFAULT 0,
    duration_ms BIGINT DEFAULT 0,
    spans_json JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agents_tenant ON agents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_policies_tenant ON policies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_credentials_hash ON credentials(hashed_key);
CREATE INDEX IF NOT EXISTS idx_approvals_pending ON approvals(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_audit_tenant_time ON audit_events(tenant_id, timestamp);
