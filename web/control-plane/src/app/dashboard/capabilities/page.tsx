"use client";

import { useState } from "react";
import { 
  GitBranch, 
  CheckCircle2, 
  Award, 
  Zap, 
  ShieldCheck, 
  Clock, 
  DollarSign, 
  Search,
  Filter,
  ArrowUpRight
} from "lucide-react";

interface CapabilityRecord {
  id: string;
  name: string;
  description: string;
  requiredTools: string[];
  allowedDataClasses: string[];
  agents: {
    agentId: string;
    version: string;
    evidenceTier: "DECLARED" | "EVALUATED" | "PRODUCTION_OBSERVED";
    qualityScore: number;
    p95LatencyMs: number;
    avgCostUSD: number;
    sampleCount: number;
    confidence: number;
  }[];
}

const mockCapabilities: CapabilityRecord[] = [
  {
    id: "quote_analysis",
    name: "Enterprise Quote & Invoice Analysis",
    description: "Multi-vendor RFP evaluation, line-item extraction, pricing discrepancy reconciliation.",
    requiredTools: ["bigquery.read", "internal.erp.quote"],
    allowedDataClasses: ["INTERNAL", "FINANCIAL"],
    agents: [
      {
        agentId: "procurement-agent",
        version: "1.0.0",
        evidenceTier: "PRODUCTION_OBSERVED",
        qualityScore: 0.98,
        p95LatencyMs: 4200,
        avgCostUSD: 0.0078,
        sampleCount: 428,
        confidence: 0.96,
      },
      {
        agentId: "analyst-agent",
        version: "2.1.0",
        evidenceTier: "EVALUATED",
        qualityScore: 0.94,
        p95LatencyMs: 5100,
        avgCostUSD: 0.0062,
        sampleCount: 50,
        confidence: 0.88,
      }
    ]
  },
  {
    id: "general_research",
    name: "Deep Web & Academic Research",
    description: "Factual synthesis across technical documentation, academic corpora, and indexed search.",
    requiredTools: ["web.search"],
    allowedDataClasses: ["PUBLIC", "INTERNAL"],
    agents: [
      {
        agentId: "research-agent",
        version: "0.9.0",
        evidenceTier: "EVALUATED",
        qualityScore: 0.96,
        p95LatencyMs: 2800,
        avgCostUSD: 0.0042,
        sampleCount: 120,
        confidence: 0.92,
      }
    ]
  },
  {
    id: "financial_reconciliation",
    name: "General Ledger Reconciliation",
    description: "Automated variance detection and ledger cross-matching against bank feeds.",
    requiredTools: ["bigquery.read"],
    allowedDataClasses: ["RESTRICTED", "FINANCIAL"],
    agents: [
      {
        agentId: "finance-agent",
        version: "1.1.0",
        evidenceTier: "PRODUCTION_OBSERVED",
        qualityScore: 0.99,
        p95LatencyMs: 3100,
        avgCostUSD: 0.0045,
        sampleCount: 312,
        confidence: 0.98,
      }
    ]
  }
];

export default function CapabilitiesPage() {
  const [searchTerm, setSearchTerm] = useState("");

  const filtered = mockCapabilities.filter((c) => 
    c.name.toLowerCase().includes(searchTerm.toLowerCase()) || 
    c.id.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const getTierBadge = (tier: string) => {
    switch (tier) {
      case "PRODUCTION_OBSERVED":
        return <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 font-bold">PRODUCTION OBSERVED</span>;
      case "EVALUATED":
        return <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-cyan-500/20 text-cyan-400 border border-cyan-500/30">EVALUATED IN CI</span>;
      default:
        return <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-slate-700/50 text-slate-400 border border-slate-600">DECLARED ONLY</span>;
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-white tracking-tight">Capability Registry & Routing V2</h1>
            <span className="px-2.5 py-0.5 rounded-full text-xs font-mono bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              Evidence-Weighted Ranking
            </span>
          </div>
          <p className="text-sm text-slate-400 mt-1">
            Empirical capability matching separating declared claims from measured production performance.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="w-4 h-4 absolute left-3 top-2.5 text-slate-500" />
            <input 
              type="text"
              placeholder="Filter capabilities..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="bg-slate-900 border border-slate-800 rounded-lg pl-9 pr-3 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-emerald-500 w-64 font-mono"
            />
          </div>
        </div>
      </div>

      {/* Capabilities List */}
      <div className="space-y-4">
        {filtered.map((cap) => (
          <div key={cap.id} className="glass-panel p-6">
            <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4 pb-4 border-b border-slate-800">
              <div>
                <div className="flex items-center gap-3">
                  <h2 className="text-lg font-bold text-white tracking-tight">{cap.name}</h2>
                  <span className="px-2 py-0.5 rounded text-xs font-mono bg-slate-800 text-cyan-400">
                    {cap.id}
                  </span>
                </div>
                <p className="text-xs text-slate-400 mt-1">{cap.description}</p>
              </div>

              <div className="flex items-center gap-2">
                <span className="text-xs text-slate-500 font-mono">Required Tools:</span>
                {cap.requiredTools.map((t) => (
                  <span key={t} className="px-2 py-0.5 rounded bg-slate-900 text-slate-300 border border-slate-800 text-[11px] font-mono">
                    {t}
                  </span>
                ))}
              </div>
            </div>

            {/* Eligible Agents Table */}
            <div className="mt-4">
              <div className="text-xs font-mono text-slate-500 mb-2 uppercase">Verified Candidate Agents</div>
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead>
                    <tr className="border-b border-slate-800 text-slate-400 font-mono">
                      <th className="py-2 pr-4">AGENT ID</th>
                      <th className="py-2 px-4">VERSION</th>
                      <th className="py-2 px-4">EVIDENCE TIER</th>
                      <th className="py-2 px-4">QUALITY SCORE</th>
                      <th className="py-2 px-4">P95 LATENCY</th>
                      <th className="py-2 px-4">AVG COST</th>
                      <th className="py-2 px-4">CONFIDENCE</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/60 font-mono">
                    {cap.agents.map((ag) => (
                      <tr key={ag.agentId} className="hover:bg-slate-900/40 transition">
                        <td className="py-3 pr-4 font-bold text-white flex items-center gap-2">
                          <GitBranch className="w-3.5 h-3.5 text-emerald-400" />
                          {ag.agentId}
                        </td>
                        <td className="py-3 px-4 text-slate-400">{ag.version}</td>
                        <td className="py-3 px-4">{getTierBadge(ag.evidenceTier)}</td>
                        <td className="py-3 px-4 text-emerald-400 font-bold">{(ag.qualityScore * 100).toFixed(1)}%</td>
                        <td className="py-3 px-4 text-slate-300">{ag.p95LatencyMs}ms</td>
                        <td className="py-3 px-4 text-slate-300">${ag.avgCostUSD.toFixed(4)}</td>
                        <td className="py-3 px-4">
                          <span className="px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                            {(ag.confidence * 100).toFixed(0)}%
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
