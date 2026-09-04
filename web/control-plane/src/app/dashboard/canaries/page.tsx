import { Layers, ArrowRight, RotateCcw, CheckCircle, AlertTriangle, TrendingUp } from "lucide-react";

export default function CanariesRolloutsPage() {
  const activeCanaries = [
    {
      id: "canary_procurement-agent_v1.1.0",
      agentId: "procurement-agent",
      baselineVersion: "v1.0.0",
      candidateVersion: "v1.1.0",
      trafficWeight: 25,
      status: "CANARY_HEALTHY",
      shadowMode: false,
      baseline: {
        successRate: "99.8%",
        p95Latency: "8,400ms",
        avgCost: "$0.0078",
      },
      candidate: {
        successRate: "99.9%",
        p95Latency: "6,200ms",
        avgCost: "$0.0054",
      },
      improvements: {
        latency: "-26% faster",
        cost: "-30% cheaper",
      },
    },
  ];

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold text-white tracking-tight">Progressive Delivery & Canaries</h1>
        <p className="text-sm text-slate-400">
          Safely evaluate, shadow, canary, and automatically promote or roll back AI agent revisions.
        </p>
      </div>

      <div className="space-y-6">
        {activeCanaries.map((c) => (
          <div key={c.id} className="glass-panel p-6">
            <div className="flex flex-col md:flex-row md:items-center justify-between pb-6 border-b border-slate-800 gap-4">
              <div>
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-xs font-mono text-slate-500">CANARY ID:</span>
                  <span className="text-xs font-mono text-white font-bold">{c.id}</span>
                  <span className="px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 text-[10px] font-bold">
                    {c.status}
                  </span>
                </div>
                <div className="text-lg font-bold text-white">
                  Agent: <span className="text-emerald-400 font-mono">{c.agentId}</span>
                </div>
                <div className="text-xs text-slate-400 font-mono mt-1">
                  Baseline: {c.baselineVersion} <ArrowRight className="inline w-3 h-3 mx-1 text-slate-500" /> Candidate: {c.candidateVersion}
                </div>
              </div>

              <div className="flex items-center gap-3">
                <button className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-bold text-xs flex items-center gap-1.5 transition shadow-lg shadow-emerald-500/10">
                  <TrendingUp className="w-4 h-4" /> Promote to 50%
                </button>
                <button className="px-4 py-2 rounded-lg bg-red-500/20 hover:bg-red-500/30 text-red-400 border border-red-500/30 font-bold text-xs flex items-center gap-1.5 transition">
                  <RotateCcw className="w-4 h-4" /> Emergency Rollback
                </button>
              </div>
            </div>

            {/* Traffic Split Visualizer */}
            <div className="pt-6 pb-4">
              <div className="flex justify-between text-xs font-mono mb-2">
                <span className="text-slate-400">Baseline ({100 - c.trafficWeight}% Traffic)</span>
                <span className="text-emerald-400 font-bold">Candidate ({c.trafficWeight}% Traffic)</span>
              </div>
              <div className="w-full h-4 bg-slate-950 rounded-full overflow-hidden flex border border-slate-800">
                <div
                  className="bg-slate-700 h-full transition-all duration-500"
                  style={{ width: `${100 - c.trafficWeight}%` }}
                />
                <div
                  className="bg-emerald-500 h-full transition-all duration-500"
                  style={{ width: `${c.trafficWeight}%` }}
                />
              </div>
            </div>

            {/* Metric Deltas */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-4 text-xs font-mono">
              <div className="p-4 rounded-lg bg-slate-950/60 border border-slate-800">
                <div className="text-slate-500 uppercase text-[10px] font-bold mb-3">Baseline Performance ({c.baselineVersion})</div>
                <div className="grid grid-cols-3 gap-2">
                  <div>
                    <div className="text-slate-500 text-[10px]">SUCCESS</div>
                    <div className="text-base font-bold text-white">{c.baseline.successRate}</div>
                  </div>
                  <div>
                    <div className="text-slate-500 text-[10px]">P95 LATENCY</div>
                    <div className="text-base font-bold text-white">{c.baseline.p95Latency}</div>
                  </div>
                  <div>
                    <div className="text-slate-500 text-[10px]">AVG COST</div>
                    <div className="text-base font-bold text-white">{c.baseline.avgCost}</div>
                  </div>
                </div>
              </div>

              <div className="p-4 rounded-lg bg-emerald-950/20 border border-emerald-500/30">
                <div className="text-emerald-400 uppercase text-[10px] font-bold mb-3 flex items-center justify-between">
                  <span>Candidate Performance ({c.candidateVersion})</span>
                  <span className="text-[10px] text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded">
                    {c.improvements.latency} • {c.improvements.cost}
                  </span>
                </div>
                <div className="grid grid-cols-3 gap-2">
                  <div>
                    <div className="text-slate-500 text-[10px]">SUCCESS</div>
                    <div className="text-base font-bold text-emerald-400">{c.candidate.successRate}</div>
                  </div>
                  <div>
                    <div className="text-slate-500 text-[10px]">P95 LATENCY</div>
                    <div className="text-base font-bold text-emerald-400">{c.candidate.p95Latency}</div>
                  </div>
                  <div>
                    <div className="text-slate-500 text-[10px]">AVG COST</div>
                    <div className="text-base font-bold text-emerald-400">{c.candidate.avgCost}</div>
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
