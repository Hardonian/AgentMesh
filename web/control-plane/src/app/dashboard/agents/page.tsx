import { Bot, Shield, CheckCircle2, Award, Zap, AlertCircle } from "lucide-react";

export default function AgentsPassportPage() {
  const agents = [
    {
      id: "procurement-agent",
      name: "Corporate Procurement Agent",
      version: "1.0.0",
      runtime: "Go 1.26",
      framework: "Google ADK Go",
      status: "HEALTHY",
      evidenceStatus: "MEASURED",
      confidence: "98%",
      declared: {
        capabilities: ["vendor_search", "quote_analysis", "purchase_request"],
        tools: ["bigquery.read", "internal.erp.quote"],
        targetSuccessRate: "99.5%",
        targetP95: "12,000ms",
      },
      measured: {
        samples: 284,
        successRate: "99.8%",
        p95Latency: "8,400ms",
        avgCost: "$0.0078",
        toolCallSuccess: "99.9%",
        compliance: "COMPLIANT",
      },
    },
    {
      id: "finance-agent",
      name: "Financial Analysis Agent",
      version: "1.1.0",
      runtime: "Go 1.26",
      framework: "Google ADK Go",
      status: "HEALTHY",
      evidenceStatus: "MEASURED",
      confidence: "92%",
      declared: {
        capabilities: ["financial_reconciliation", "budget_check"],
        tools: ["bigquery.read", "payment.execute"],
        targetSuccessRate: "99.0%",
        targetP95: "5,000ms",
      },
      measured: {
        samples: 142,
        successRate: "99.3%",
        p95Latency: "3,100ms",
        avgCost: "$0.0045",
        toolCallSuccess: "99.6%",
        compliance: "COMPLIANT",
      },
    },
    {
      id: "research-agent",
      name: "Web & Academic Research Agent",
      version: "0.9.0",
      runtime: "Go 1.26",
      framework: "Custom A2A",
      status: "HEALTHY",
      evidenceStatus: "INFERRED",
      confidence: "45%",
      declared: {
        capabilities: ["market_search", "paper_summarize"],
        tools: ["google.search", "web.scrape"],
        targetSuccessRate: "95.0%",
        targetP95: "15,000ms",
      },
      measured: {
        samples: 8,
        successRate: "96.0%",
        p95Latency: "11,200ms",
        avgCost: "$0.0120",
        toolCallSuccess: "97.5%",
        compliance: "COMPLIANT",
      },
    },
  ];

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight">Agent Registry & Passports</h1>
          <p className="text-sm text-slate-400">
            Declared contracts verified against empirical operational evidence.
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs font-mono text-emerald-400 bg-emerald-500/10 px-3 py-1.5 rounded-lg border border-emerald-500/20">
          <Award className="w-4 h-4" /> AgentMesh Passport Verified
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6">
        {agents.map((agent) => (
          <div key={agent.id} className="glass-panel p-6">
            <div className="flex flex-col md:flex-row md:items-center justify-between pb-4 border-b border-slate-800 gap-4">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-lg bg-slate-900 border border-slate-800 flex items-center justify-center text-emerald-400">
                  <Bot className="w-5 h-5" />
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <h2 className="text-lg font-bold text-white">{agent.name}</h2>
                    <span className="text-xs font-mono text-slate-400">({agent.id})</span>
                    <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-slate-800 text-slate-300">
                      v{agent.version}
                    </span>
                  </div>
                  <div className="text-xs text-slate-500 font-mono mt-0.5">
                    Runtime: {agent.runtime} • Framework: {agent.framework}
                  </div>
                </div>
              </div>

              <div className="flex items-center gap-3">
                <div className="text-right">
                  <div className="text-xs text-slate-400">Evidence Status</div>
                  <div className="flex items-center gap-1.5 justify-end mt-0.5">
                    <span className={`w-2 h-2 rounded-full ${agent.evidenceStatus === "MEASURED" ? "bg-emerald-400" : "bg-amber-400"}`} />
                    <span className="text-xs font-mono font-bold text-white">{agent.evidenceStatus}</span>
                    <span className="text-[10px] text-slate-500">({agent.confidence} conf.)</span>
                  </div>
                </div>
              </div>
            </div>

            {/* Passport Body: Declared vs Measured */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 pt-6">
              {/* Declared Claims */}
              <div className="p-4 rounded-lg bg-slate-950/60 border border-slate-800">
                <div className="text-xs font-mono font-bold text-slate-400 uppercase tracking-wider mb-3 flex items-center gap-1.5">
                  <Shield className="w-3.5 h-3.5 text-blue-400" /> Declared Contract Specifications
                </div>
                <div className="space-y-3 text-xs">
                  <div>
                    <span className="text-slate-500">Capabilities:</span>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {agent.declared.capabilities.map((c) => (
                        <span key={c} className="px-2 py-0.5 rounded bg-blue-500/10 text-blue-300 font-mono text-[10px]">
                          {c}
                        </span>
                      ))}
                    </div>
                  </div>
                  <div>
                    <span className="text-slate-500">Allowed Tools:</span>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {agent.declared.tools.map((t) => (
                        <span key={t} className="px-2 py-0.5 rounded bg-slate-800 text-slate-300 font-mono text-[10px]">
                          {t}
                        </span>
                      ))}
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-2 pt-2 border-t border-slate-800/80 font-mono text-[11px]">
                    <div>
                      <span className="text-slate-500">Target SLO: </span>
                      <span className="text-slate-300">{agent.declared.targetSuccessRate}</span>
                    </div>
                    <div>
                      <span className="text-slate-500">Target P95: </span>
                      <span className="text-slate-300">{agent.declared.targetP95}</span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Measured Evidence */}
              <div className="p-4 rounded-lg bg-emerald-950/20 border border-emerald-500/30">
                <div className="text-xs font-mono font-bold text-emerald-400 uppercase tracking-wider mb-3 flex items-center gap-1.5">
                  <Zap className="w-3.5 h-3.5 text-emerald-400" /> Measured Operational Evidence
                </div>
                <div className="grid grid-cols-2 gap-4 text-xs font-mono">
                  <div>
                    <div className="text-slate-500 text-[10px]">SUCCESS RATE</div>
                    <div className="text-lg font-bold text-emerald-400">{agent.measured.successRate}</div>
                    <div className="text-[10px] text-slate-500">Target: {agent.declared.targetSuccessRate}</div>
                  </div>
                  <div>
                    <div className="text-slate-500 text-[10px]">P95 LATENCY</div>
                    <div className="text-lg font-bold text-white">{agent.measured.p95Latency}</div>
                    <div className="text-[10px] text-slate-500">Target: {agent.declared.targetP95}</div>
                  </div>
                  <div>
                    <div className="text-slate-500 text-[10px]">AVG TASK COST</div>
                    <div className="text-base font-bold text-emerald-400">{agent.measured.avgCost}</div>
                    <div className="text-[10px] text-slate-500">{agent.measured.samples} samples</div>
                  </div>
                  <div>
                    <div className="text-slate-500 text-[10px]">POLICY STATUS</div>
                    <div className="flex items-center gap-1 text-emerald-400 font-bold mt-1">
                      <CheckCircle2 className="w-3.5 h-3.5" /> {agent.measured.compliance}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
