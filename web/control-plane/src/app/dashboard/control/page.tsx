"use client";

import { useState } from "react";
import {
  ShieldAlert,
  Play,
  RotateCcw,
  CheckCircle2,
  TrendingDown,
  Zap,
  Lock,
  Flame,
  Activity,
  ArrowRight,
  Sparkles,
} from "lucide-react";

interface ActionItem {
  id: string;
  capability: string;
  actionType: string;
  current: string;
  proposed: string;
  risk: "LOW" | "MEDIUM" | "HIGH" | "CRITICAL";
  savings: string;
  latencyDelta: string;
  confidence: number;
  status: "RECOMMENDED" | "APPROVED" | "CANARYING" | "PROMOTED" | "ROLLED_BACK";
}

interface CanaryItem {
  id: string;
  capability: string;
  targetType: string;
  baseline: string;
  candidate: string;
  trafficPercent: number;
  stage: string;
  p95Latency: string;
  errorRate: string;
  status: "HEALTHY" | "HOLD" | "ROLLED_BACK";
}

export default function OperatorControlCenter() {
  const [frozen, setFrozen] = useState(false);
  const [activeTab, setActiveTab] = useState<"actions" | "canaries" | "outcomes">("actions");

  const [actions, setActions] = useState<ActionItem[]>([
    {
      id: "act-101",
      capability: "financial_forecast",
      actionType: "CHANGE_ROUTE_WEIGHT",
      current: "Agent-V1 (100%)",
      proposed: "Agent-V2-Candidate (25%)",
      risk: "LOW",
      savings: "-18.5%",
      latencyDelta: "-120ms",
      confidence: 0.94,
      status: "CANARYING",
    },
    {
      id: "act-102",
      capability: "code_review",
      actionType: "CHANGE_MODEL_TARGET",
      current: "Gemini-1.5-Flash",
      proposed: "Gemini-1.5-Pro",
      risk: "HIGH",
      savings: "+4.2%",
      latencyDelta: "+80ms",
      confidence: 0.91,
      status: "RECOMMENDED",
    },
  ]);

  const canaries: CanaryItem[] = [
    {
      id: "canary-991",
      capability: "financial_forecast",
      targetType: "AGENT_VERSION",
      baseline: "v1.4.2",
      candidate: "v2.0.0-rc1",
      trafficPercent: 25,
      stage: "Stage 4 of 6 (25%)",
      p95Latency: "840ms (Baseline: 960ms)",
      errorRate: "0.00% (Threshold: 1.00%)",
      status: "HEALTHY",
    },
  ];

  const handleApprove = (id: string) => {
    setActions((prev) =>
      prev.map((a) => (a.id === id ? { ...a, status: "APPROVED" } : a))
    );
  };

  return (
    <div style={{ padding: "2rem", color: "#e2e8f0", maxWidth: "1400px", margin: "0 auto" }}>
      {/* Header */}
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: "2rem" }}>
        <div>
          <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", marginBottom: "0.5rem" }}>
            <span style={{ fontSize: "1.75rem", fontWeight: 700, letterSpacing: "-0.025em" }}>
              Operator Control Center
            </span>
            <span
              style={{
                fontSize: "0.75rem",
                padding: "0.25rem 0.6rem",
                borderRadius: "9999px",
                backgroundColor: "rgba(59, 130, 246, 0.15)",
                color: "#60a5fa",
                border: "1px solid rgba(59, 130, 246, 0.3)",
                fontWeight: 600,
              }}
            >
              PHASE 4 AUTONOMOUS OPS
            </span>
          </div>
          <p style={{ color: "#94a3b8", fontSize: "0.95rem" }}>
            Policy-bounded optimization, progressive canary delivery, and verified production outcome loops.
          </p>
        </div>

        {/* Emergency Kill Switch Button */}
        <div style={{ display: "flex", alignItems: "center", gap: "1rem" }}>
          <button
            onClick={() => setFrozen(!frozen)}
            style={{
              display: "flex",
              alignItems: "center",
              gap: "0.5rem",
              padding: "0.6rem 1.2rem",
              borderRadius: "0.5rem",
              fontWeight: 600,
              fontSize: "0.875rem",
              cursor: "pointer",
              transition: "all 0.2s",
              border: frozen ? "1px solid #ef4444" : "1px solid #dc2626",
              backgroundColor: frozen ? "#ef4444" : "rgba(220, 38, 38, 0.15)",
              color: frozen ? "#ffffff" : "#f87171",
            }}
          >
            <ShieldAlert size={16} />
            {frozen ? "EMERGENCY FREEZE ACTIVE (CLICK TO UNFREEZE)" : "ACTIVATE EMERGENCY KILL SWITCH"}
          </button>
        </div>
      </div>

      {/* KPI Cards */}
      <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: "1rem", marginBottom: "2rem" }}>
        <div style={{ backgroundColor: "#1e293b", padding: "1.25rem", borderRadius: "0.75rem", border: "1px solid #334155" }}>
          <div style={{ color: "#94a3b8", fontSize: "0.8rem", marginBottom: "0.5rem", display: "flex", alignItems: "center", gap: "0.4rem" }}>
            <Activity size={14} color="#60a5fa" /> EXECUTION MODE
          </div>
          <div style={{ fontSize: "1.4rem", fontWeight: 700, color: "#f8fafc" }}>GUARDED_AUTOMATION</div>
          <div style={{ color: "#22c55e", fontSize: "0.75rem", marginTop: "0.25rem" }}>Low-risk mutations policy-authorized</div>
        </div>

        <div style={{ backgroundColor: "#1e293b", padding: "1.25rem", borderRadius: "0.75rem", border: "1px solid #334155" }}>
          <div style={{ color: "#94a3b8", fontSize: "0.8rem", marginBottom: "0.5rem", display: "flex", alignItems: "center", gap: "0.4rem" }}>
            <TrendingDown size={14} color="#22c55e" /> VERIFIED SAVINGS
          </div>
          <div style={{ fontSize: "1.4rem", fontWeight: 700, color: "#22c55e" }}>$1,428.50</div>
          <div style={{ color: "#94a3b8", fontSize: "0.75rem", marginTop: "0.25rem" }}>Past 30 days empirical delta</div>
        </div>

        <div style={{ backgroundColor: "#1e293b", padding: "1.25rem", borderRadius: "0.75rem", border: "1px solid #334155" }}>
          <div style={{ color: "#94a3b8", fontSize: "0.8rem", marginBottom: "0.5rem", display: "flex", alignItems: "center", gap: "0.4rem" }}>
            <Flame size={14} color="#f59e0b" /> ACTIVE CANARIES
          </div>
          <div style={{ fontSize: "1.4rem", fontWeight: 700, color: "#f59e0b" }}>1 In Flight</div>
          <div style={{ color: "#94a3b8", fontSize: "0.75rem", marginTop: "0.25rem" }}>Stage 4 of 6 (25% weight)</div>
        </div>

        <div style={{ backgroundColor: "#1e293b", padding: "1.25rem", borderRadius: "0.75rem", border: "1px solid #334155" }}>
          <div style={{ color: "#94a3b8", fontSize: "0.8rem", marginBottom: "0.5rem", display: "flex", alignItems: "center", gap: "0.4rem" }}>
            <Lock size={14} color="#a855f7" /> HASH-BOUND APPROVALS
          </div>
          <div style={{ fontSize: "1.4rem", fontWeight: 700, color: "#a855f7" }}>100% Enforced</div>
          <div style={{ color: "#94a3b8", fontSize: "0.75rem", marginTop: "0.25rem" }}>Cryptographic replay protection</div>
        </div>
      </div>

      {/* Tabs */}
      <div style={{ display: "flex", gap: "1rem", borderBottom: "1px solid #334155", marginBottom: "1.5rem" }}>
        <button
          onClick={() => setActiveTab("actions")}
          style={{
            padding: "0.6rem 1rem",
            backgroundColor: "transparent",
            border: "none",
            borderBottom: activeTab === "actions" ? "2px solid #3b82f6" : "2px solid transparent",
            color: activeTab === "actions" ? "#60a5fa" : "#94a3b8",
            fontWeight: 600,
            cursor: "pointer",
          }}
        >
          Optimization Actions & Workflows
        </button>
        <button
          onClick={() => setActiveTab("canaries")}
          style={{
            padding: "0.6rem 1rem",
            backgroundColor: "transparent",
            border: "none",
            borderBottom: activeTab === "canaries" ? "2px solid #3b82f6" : "2px solid transparent",
            color: activeTab === "canaries" ? "#60a5fa" : "#94a3b8",
            fontWeight: 600,
            cursor: "pointer",
          }}
        >
          Live Canary Deliveries
        </button>
      </div>

      {/* Tab Content */}
      {activeTab === "actions" && (
        <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
          {actions.map((act) => (
            <div
              key={act.id}
              style={{
                backgroundColor: "#1e293b",
                border: "1px solid #334155",
                borderRadius: "0.75rem",
                padding: "1.5rem",
                display: "flex",
                justifyContent: "space-between",
                alignItems: "center",
              }}
            >
              <div>
                <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", marginBottom: "0.5rem" }}>
                  <span style={{ fontWeight: 700, fontSize: "1.1rem" }}>{act.capability}</span>
                  <span
                    style={{
                      fontSize: "0.75rem",
                      padding: "0.2rem 0.5rem",
                      borderRadius: "0.25rem",
                      backgroundColor:
                        act.risk === "LOW"
                          ? "rgba(34, 197, 94, 0.15)"
                          : act.risk === "MEDIUM"
                          ? "rgba(245, 158, 11, 0.15)"
                          : "rgba(239, 68, 68, 0.15)",
                      color:
                        act.risk === "LOW" ? "#4ade80" : act.risk === "MEDIUM" ? "#fcd34d" : "#f87171",
                      fontWeight: 600,
                    }}
                  >
                    {act.risk} RISK
                  </span>
                  <span
                    style={{
                      fontSize: "0.75rem",
                      padding: "0.2rem 0.5rem",
                      borderRadius: "0.25rem",
                      backgroundColor: "rgba(148, 163, 184, 0.15)",
                      color: "#94a3b8",
                    }}
                  >
                    {act.actionType}
                  </span>
                </div>

                <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", color: "#94a3b8", fontSize: "0.875rem" }}>
                  <span>{act.current}</span>
                  <ArrowRight size={14} />
                  <span style={{ color: "#f8fafc", fontWeight: 600 }}>{act.proposed}</span>
                  <span style={{ color: "#22c55e", marginLeft: "1rem" }}>Savings: {act.savings}</span>
                  <span>Latency: {act.latencyDelta}</span>
                </div>
              </div>

              <div style={{ display: "flex", gap: "0.75rem" }}>
                {act.status === "RECOMMENDED" && (
                  <button
                    onClick={() => handleApprove(act.id)}
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: "0.4rem",
                      padding: "0.5rem 1rem",
                      backgroundColor: "#2563eb",
                      color: "#ffffff",
                      border: "none",
                      borderRadius: "0.375rem",
                      fontWeight: 600,
                      cursor: "pointer",
                    }}
                  >
                    <CheckCircle2 size={16} /> Approve Action
                  </button>
                )}
                {act.status === "APPROVED" && (
                  <button
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: "0.4rem",
                      padding: "0.5rem 1rem",
                      backgroundColor: "#16a34a",
                      color: "#ffffff",
                      border: "none",
                      borderRadius: "0.375rem",
                      fontWeight: 600,
                      cursor: "pointer",
                    }}
                  >
                    <Play size={16} /> Start Canary Rollout
                  </button>
                )}
                {act.status === "CANARYING" && (
                  <span style={{ color: "#f59e0b", fontWeight: 600, fontSize: "0.875rem", display: "flex", alignItems: "center", gap: "0.4rem" }}>
                    <Sparkles size={16} /> CANARY IN FLIGHT (25%)
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {activeTab === "canaries" && (
        <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
          {canaries.map((c) => (
            <div
              key={c.id}
              style={{
                backgroundColor: "#1e293b",
                border: "1px solid #334155",
                borderRadius: "0.75rem",
                padding: "1.5rem",
              }}
            >
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "1rem" }}>
                <div>
                  <span style={{ fontWeight: 700, fontSize: "1.1rem" }}>{c.capability}</span>
                  <span style={{ color: "#94a3b8", fontSize: "0.875rem", marginLeft: "0.75rem" }}>
                    Baseline: {c.baseline} vs Candidate: {c.candidate}
                  </span>
                </div>
                <div style={{ display: "flex", gap: "0.5rem" }}>
                  <button
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: "0.4rem",
                      padding: "0.4rem 0.8rem",
                      backgroundColor: "rgba(239, 68, 68, 0.15)",
                      color: "#f87171",
                      border: "1px solid rgba(239, 68, 68, 0.3)",
                      borderRadius: "0.375rem",
                      fontWeight: 600,
                      cursor: "pointer",
                    }}
                  >
                    <RotateCcw size={14} /> Immediate Rollback
                  </button>
                </div>
              </div>

              {/* Progress bar */}
              <div style={{ marginBottom: "1rem" }}>
                <div style={{ display: "flex", justifyContent: "space-between", fontSize: "0.75rem", color: "#94a3b8", marginBottom: "0.25rem" }}>
                  <span>{c.stage}</span>
                  <span>{c.trafficPercent}% Traffic to Candidate</span>
                </div>
                <div style={{ width: "100%", height: "8px", backgroundColor: "#334155", borderRadius: "9999px", overflow: "hidden" }}>
                  <div style={{ width: `${c.trafficPercent}%`, height: "100%", backgroundColor: "#3b82f6" }} />
                </div>
              </div>

              <div style={{ display: "grid", gridTemplateColumns: "repeat(2, 1fr)", gap: "1rem", fontSize: "0.875rem" }}>
                <div style={{ color: "#94a3b8" }}>
                  P95 Latency: <span style={{ color: "#22c55e", fontWeight: 600 }}>{c.p95Latency}</span>
                </div>
                <div style={{ color: "#94a3b8" }}>
                  Error Rate: <span style={{ color: "#22c55e", fontWeight: 600 }}>{c.errorRate}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
