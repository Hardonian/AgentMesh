-- Migration 000004: Phase 4 Autonomous Progressive Delivery, Optimization Actions, and Closed-Loop Operations

-- 1. Agent Optimization Actions
CREATE TABLE IF NOT EXISTS agent_optimization_actions (
    action_id VARCHAR(128) PRIMARY KEY,
    organization_id VARCHAR(64) NOT NULL,
    project_id VARCHAR(64) NOT NULL,
    capability_id VARCHAR(128) NOT NULL,
    target_type VARCHAR(64) NOT NULL,
    target_id VARCHAR(128) NOT NULL,
    action_type VARCHAR(64) NOT NULL,
    current_state JSONB NOT NULL DEFAULT '{}',
    proposed_state JSONB NOT NULL DEFAULT '{}',
    reason TEXT NOT NULL,
    evidence JSONB NOT NULL DEFAULT '{}',
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    policy_version VARCHAR(64) NOT NULL,
    route_algorithm_version VARCHAR(64) NOT NULL,
    risk_class VARCHAR(32) NOT NULL,
    blast_radius JSONB NOT NULL DEFAULT '{}',
    cost_delta_estimate DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    latency_delta_estimate_ms BIGINT NOT NULL DEFAULT 0,
    quality_delta_estimate DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    reliability_delta_estimate DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    rollback_plan JSONB NOT NULL DEFAULT '{}',
    approval_requirement JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    approved_at TIMESTAMP WITH TIME ZONE,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    result VARCHAR(64),
    schema_version VARCHAR(32) NOT NULL DEFAULT 'v1'
);

CREATE INDEX IF NOT EXISTS idx_opt_actions_org_cap ON agent_optimization_actions (organization_id, capability_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_opt_actions_target ON agent_optimization_actions (organization_id, target_type, target_id);

-- 2. Action Workflows
CREATE TABLE IF NOT EXISTS action_workflows (
    workflow_id VARCHAR(128) PRIMARY KEY,
    organization_id VARCHAR(64) NOT NULL,
    project_id VARCHAR(64) NOT NULL,
    action_id VARCHAR(128) NOT NULL REFERENCES agent_optimization_actions(action_id) ON DELETE CASCADE,
    action_hash VARCHAR(128) NOT NULL,
    current_state VARCHAR(64) NOT NULL,
    current_step_index INT NOT NULL DEFAULT 0,
    target_lock_key VARCHAR(256) NOT NULL,
    approved_by VARCHAR(128),
    approved_action_hash VARCHAR(128),
    error_message TEXT,
    last_evaluated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_action_workflows_org ON action_workflows (organization_id, current_state);
CREATE INDEX IF NOT EXISTS idx_action_workflows_lock ON action_workflows (target_lock_key);

-- 3. Routing Specs (Desired State)
CREATE TABLE IF NOT EXISTS routing_specs (
    capability_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(64) NOT NULL,
    project_id VARCHAR(64) NOT NULL,
    eligible_agents JSONB NOT NULL DEFAULT '[]',
    preferred_agents JSONB NOT NULL DEFAULT '[]',
    weights JSONB NOT NULL DEFAULT '{}',
    fallbacks JSONB NOT NULL DEFAULT '[]',
    minimum_reliability DOUBLE PRECISION NOT NULL DEFAULT 0.98,
    maximum_cost_usd DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    maximum_latency_ms BIGINT NOT NULL DEFAULT 0,
    minimum_quality DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    model_constraints JSONB NOT NULL DEFAULT '[]',
    tool_requirements JSONB NOT NULL DEFAULT '[]',
    data_classification VARCHAR(64),
    region_constraints JSONB NOT NULL DEFAULT '[]',
    exploration_policy VARCHAR(64),
    canary_policy VARCHAR(64),
    version VARCHAR(64) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organization_id, capability_id)
);

-- 4. Last Known Good Routes
CREATE TABLE IF NOT EXISTS last_known_good_routes (
    capability_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(64) NOT NULL,
    project_id VARCHAR(64) NOT NULL,
    route_spec_hash VARCHAR(128) NOT NULL,
    agent_id VARCHAR(128) NOT NULL,
    agent_version VARCHAR(64) NOT NULL,
    model_target VARCHAR(128),
    observation_window_ms BIGINT NOT NULL,
    sample_count BIGINT NOT NULL,
    success_rate DOUBLE PRECISION NOT NULL,
    p95_latency_ms BIGINT NOT NULL,
    cost_per_task_usd DOUBLE PRECISION NOT NULL,
    qualified_at TIMESTAMP WITH TIME ZONE NOT NULL,
    verified_by VARCHAR(64) NOT NULL,
    PRIMARY KEY (organization_id, capability_id)
);

-- 5. Canary Runs V3
CREATE TABLE IF NOT EXISTS canary_runs_v3 (
    run_id VARCHAR(128) PRIMARY KEY,
    organization_id VARCHAR(64) NOT NULL,
    project_id VARCHAR(64) NOT NULL,
    capability_id VARCHAR(128) NOT NULL,
    target_type VARCHAR(64) NOT NULL,
    baseline_target VARCHAR(128) NOT NULL,
    candidate_target VARCHAR(128) NOT NULL,
    stages JSONB NOT NULL DEFAULT '[]',
    current_stage_index INT NOT NULL DEFAULT 0,
    state VARCHAR(64) NOT NULL,
    stage_started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    baseline_metrics JSONB NOT NULL DEFAULT '{}',
    candidate_metrics JSONB NOT NULL DEFAULT '{}',
    rollback_trigger TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_canary_v3_org_cap ON canary_runs_v3 (organization_id, capability_id, state);

-- 6. Production Outcomes (Verified Improvement)
CREATE TABLE IF NOT EXISTS production_outcomes (
    outcome_id VARCHAR(128) PRIMARY KEY,
    action_id VARCHAR(128) NOT NULL REFERENCES agent_optimization_actions(action_id) ON DELETE CASCADE,
    organization_id VARCHAR(64) NOT NULL,
    project_id VARCHAR(64) NOT NULL,
    capability_id VARCHAR(128) NOT NULL,
    target_type VARCHAR(64) NOT NULL,
    target_id VARCHAR(128) NOT NULL,
    status VARCHAR(64) NOT NULL,
    before_window JSONB NOT NULL DEFAULT '{}',
    after_window JSONB NOT NULL DEFAULT '{}',
    workload_normalized BOOLEAN NOT NULL DEFAULT TRUE,
    success_rate_delta DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    latency_delta_ms BIGINT NOT NULL DEFAULT 0,
    cost_delta_usd DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    quality_delta DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    fallback_delta DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    tool_error_delta DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    human_escalation_delta INT NOT NULL DEFAULT 0,
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_prod_outcomes_org ON production_outcomes (organization_id, capability_id, created_at DESC);

-- 7. Automation Policies & Freezes
CREATE TABLE IF NOT EXISTS automation_policies (
    policy_id VARCHAR(128) PRIMARY KEY,
    organization_id VARCHAR(64) NOT NULL,
    project_id VARCHAR(64) NOT NULL,
    mode VARCHAR(64) NOT NULL DEFAULT 'ADVISORY',
    allow JSONB NOT NULL DEFAULT '[]',
    approval_required JSONB NOT NULL DEFAULT '[]',
    deny JSONB NOT NULL DEFAULT '[]',
    requirements JSONB NOT NULL DEFAULT '{}',
    blast_radius JSONB NOT NULL DEFAULT '{}',
    economics JSONB NOT NULL DEFAULT '{}',
    quality JSONB NOT NULL DEFAULT '{}',
    change_windows JSONB NOT NULL DEFAULT '[]',
    frozen BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS automation_freezes (
    freeze_key VARCHAR(128) PRIMARY KEY, -- SCOPE:SCOPE_ID
    scope VARCHAR(64) NOT NULL,
    scope_id VARCHAR(128) NOT NULL,
    reason TEXT NOT NULL,
    frozen_by VARCHAR(128) NOT NULL,
    frozen_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE
);
