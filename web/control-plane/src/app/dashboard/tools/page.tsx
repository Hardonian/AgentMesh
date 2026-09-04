"use client";

import { useState } from "react";
import { 
  Wrench, 
  ShieldAlert, 
  ShieldCheck, 
  AlertTriangle, 
  Clock, 
  Activity, 
  CheckCircle2, 
  XCircle, 
  Search,
  Filter,
  Fingerprint
} from "lucide-react";

interface ToolPassportUI {
  toolId: string;
  toolName: string;
  provider: string;
  server: string;
  riskClass: "READ" | "WRITE" | "DESTRUCTIVE" | "FINANCIAL" | "INFRASTRUCTURE";
  schemaFingerprint: string;
  driftStatus: "UNCHANGED" | "COMPATIBLE_CHANGE" | "POTENTIALLY_BREAKING" | "BREAKING";
  healthStatus: "HEALTHY" | "DEGRADED" | "UNHEALTHY";
  sampleCount: number;
  successRate: number;
  p95LatencyMs: number;
  lastEvaluated: string;
}

const mockToolPassports: ToolPassportUI[] = [
  {
    toolId: "bigquery.read",
    toolName: "BigQuery Read Tool",
    provider: "google-cloud-managed-mcp",
    server: "mcp-bigquery-europe-west1",
    riskClass: "READ",
    schemaFingerprint: "d9e810a42f5342a19b48c772e0a293817f0932c64e81fa214309a90184b2cd5e",
    driftStatus: "UNCHANGED",
    healthStatus: "HEALTHY",
    sampleCount: 14200,
    successRate: 0.999,
    p95LatencyMs: 310,
    lastEvaluated: "2 mins ago"
  },
  {
    toolId: "sap.po.create",
    toolName: "SAP Purchase Order Creator",
    provider: "enterprise-erp-mcp",
    server: "mcp-erp-internal",
    riskClass: "FINANCIAL",
    schemaFingerprint: "f2c9081a4b5201c59b48c772e0a293817f0932c64e81fa214309a90184b2ab11",
    driftStatus: "COMPATIBLE_CHANGE",
    healthStatus: "HEALTHY",
    sampleCount: 1240,
    successRate: 0.992,
    p95LatencyMs: 820,
    lastEvaluated: "15 mins ago"
  },
  {
    toolId: "gmail.send",
    toolName: "Google Workspace Email Dispatcher",
    provider: "google-workspace-mcp",
    server: "mcp-workspace-us-central1",
    riskClass: "WRITE",
    schemaFingerprint: "318af9d28c1102e59b48c772e0a293817f0932c64e81fa214309a90184b299a1",
    driftStatus: "UNCHANGED",
    healthStatus: "HEALTHY",
    sampleCount: 3890,
    successRate: 0.998,
    p95LatencyMs: 240,
    lastEvaluated: "1 hour ago"
  },
  {
    toolId: "gke.cluster.delete",
    toolName: "GKE Cluster Destroyer",
    provider: "google-cloud-managed-mcp",
    server: "mcp-gke-prod",
    riskClass: "DESTRUCTIVE",
    schemaFingerprint: "88a01f92e10427c59b48c772e0a293817f0932c64e81fa214309a90184b200b2",
    driftStatus: "BREAKING",
    healthStatus: "DEGRADED",
    sampleCount: 12,
    successRate: 1.0,
    p95LatencyMs: 14500,
    lastEvaluated: "Just now"
  }
];

export default function ToolsPassportPage() {
  const [searchTerm, setSearchTerm] = useState("");

  const filtered = mockToolPassports.filter((t) => 
    t.toolName.toLowerCase().includes(searchTerm.toLowerCase()) || 
    t.toolId.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const getRiskBadge = (risk: ToolPassportUI["riskClass"]) => {
    switch (risk) {
      case "DESTRUCTIVE":
        return <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-rose-500/20 text-rose-400 border border-rose-500/40 font-bold">DESTRUCTIVE</span>;
      case "FINANCIAL":
        return <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-amber-500/20 text-amber-400 border border-amber-500/40 font-bold">FINANCIAL</span>;
      case "WRITE":
        return <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-orange-500/20 text-orange-400 border border-orange-500/40">WRITE</span>;
      default:
        return <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-emerald-500/20 text-emerald-400 border border-emerald-500/40">READ ONLY</span>;
    }
  };

  const getDriftBadge = (drift: ToolPassportUI["driftStatus"]) => {
    switch (drift) {
      case "BREAKING":
        return <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-rose-500/20 text-rose-400 font-bold">BREAKING DRIFT</span>;
      case "COMPATIBLE_CHANGE":
        return <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-cyan-500/20 text-cyan-400">COMPATIBLE CHANGE</span>;
      default:
        return <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-slate-800 text-slate-400">UNCHANGED</span>;
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-white tracking-tight">MCP Tool Passports & Governance</h1>
            <span className="px-2.5 py-0.5 rounded-full text-xs font-mono bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">
              Google-Managed & Open MCP
            </span>
          </div>
          <p className="text-sm text-slate-400 mt-1">
            Empirical tool passports, stable SHA-256 schema fingerprints, and conservative schema drift detection.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="w-4 h-4 absolute left-3 top-2.5 text-slate-500" />
            <input 
              type="text"
              placeholder="Search tools..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="bg-slate-900 border border-slate-800 rounded-lg pl-9 pr-3 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-emerald-500 w-64 font-mono"
            />
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {filtered.map((tp) => (
          <div key={tp.toolId} className="glass-panel p-5 space-y-4">
            <div className="flex items-start justify-between gap-2">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-slate-900 border border-slate-800 text-cyan-400">
                  <Wrench className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-sm font-bold text-white tracking-tight">{tp.toolName}</h3>
                  <div className="text-xs font-mono text-cyan-400">{tp.toolId}</div>
                </div>
              </div>

              <div className="flex flex-col items-end gap-1">
                {getRiskBadge(tp.riskClass)}
                {getDriftBadge(tp.driftStatus)}
              </div>
            </div>

            <div className="p-3 rounded-lg bg-slate-950/60 border border-slate-800/80 font-mono text-[11px] space-y-1.5">
              <div className="flex items-center justify-between text-slate-400">
                <span>Provider:</span>
                <span className="text-slate-200">{tp.provider}</span>
              </div>
              <div className="flex items-center justify-between text-slate-400">
                <span>Server Endpoint:</span>
                <span className="text-slate-200">{tp.server}</span>
              </div>
              <div className="flex items-center justify-between text-slate-400">
                <span className="flex items-center gap-1"><Fingerprint className="w-3 h-3" /> Fingerprint:</span>
                <span className="text-slate-500 truncate max-w-[160px]" title={tp.schemaFingerprint}>
                  {tp.schemaFingerprint.slice(0, 16)}...
                </span>
              </div>
            </div>

            <div className="grid grid-cols-3 gap-2 pt-2 border-t border-slate-800 text-center font-mono">
              <div className="p-2 rounded bg-slate-900/40 border border-slate-800/60">
                <div className="text-[10px] text-slate-500">SUCCESS RATE</div>
                <div className="text-xs font-bold text-emerald-400 mt-0.5">{(tp.successRate * 100).toFixed(1)}%</div>
              </div>
              <div className="p-2 rounded bg-slate-900/40 border border-slate-800/60">
                <div className="text-[10px] text-slate-500">P95 LATENCY</div>
                <div className="text-xs font-bold text-slate-200 mt-0.5">{tp.p95LatencyMs}ms</div>
              </div>
              <div className="p-2 rounded bg-slate-900/40 border border-slate-800/60">
                <div className="text-[10px] text-slate-500">INVOCATIONS</div>
                <div className="text-xs font-bold text-slate-200 mt-0.5">{tp.sampleCount.toLocaleString()}</div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
