import { Bot, ShieldCheck, Activity, CheckSquare, ArrowUpRight, CheckCircle2, Clock } from "lucide-react";
import Link from "next/link";

export default function DashboardOverviewPage() {
  const stats = [
    { label: "Active Agents", value: "4", change: "+1 today", icon: Bot, color: "text-emerald-400" },
    { label: "Policy Decisions", value: "1,248", change: "99.2% Allowed", icon: ShieldCheck, color: "text-blue-400" },
    { label: "Proxy P95 Latency", value: "1.4ms", change: "Sub-millisecond core", icon: Activity, color: "text-cyan-400" },
    { label: "Pending Approvals", value: "1", change: "Requires review", icon: CheckSquare, color: "text-amber-400" },
  ];

  const recentTraces = [
    { id: "tr_99a81c0f", root: "procurement-agent", action: "Purchase Requisition", duration: "320ms", cost: "$0.0084", status: "SUCCESS" },
    { id: "tr_88b14e2d", root: "analyst-agent", action: "Market Insight Query", duration: "180ms", cost: "$0.0032", status: "SUCCESS" },
    { id: "tr_77c29d11", root: "research-agent", action: "Competitor Analysis", duration: "410ms", cost: "$0.0112", status: "SUCCESS" },
    { id: "tr_66d40a33", root: "procurement-agent", action: "ERP Payment Delete", duration: "45ms", cost: "$0.0000", status: "APPROVAL_REQUIRED" },
  ];

  const circuitBreakers = [
    { target: "bigquery.read", type: "MCP Tool", state: "CLOSED", failures: "0/3", lastProbe: "Just now" },
    { target: "finance-agent", type: "A2A Agent", state: "CLOSED", failures: "0/3", lastProbe: "12s ago" },
    { target: "gemini-1.5-pro", type: "Model Provider", state: "CLOSED", failures: "0/5", lastProbe: "5s ago" },
  ];

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold text-white tracking-tight">System Overview</h1>
        <p className="text-sm text-slate-400">Real-time health, routing telemetry, and policy enforcement metrics.</p>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {stats.map((s, idx) => {
          const Icon = s.icon;
          return (
            <div key={idx} className="glass-panel p-5">
              <div className="flex items-center justify-between mb-3">
                <span className="text-xs font-medium text-slate-400">{s.label}</span>
                <Icon className={`w-4 h-4 ${s.color}`} />
              </div>
              <div className="text-3xl font-extrabold text-white tracking-tight mb-1">{s.value}</div>
              <div className="text-xs text-slate-500 font-mono">{s.change}</div>
            </div>
          );
        })}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Waterfall Traces preview */}
        <div className="lg:col-span-2 glass-panel p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-base font-bold text-white flex items-center gap-2">
              <Activity className="w-4 h-4 text-emerald-400" /> Recent Execution Traces
            </h2>
            <Link href="/dashboard/traces" className="text-xs text-emerald-400 hover:text-emerald-300 flex items-center gap-1 font-mono">
              View Waterfall <ArrowUpRight className="w-3 h-3" />
            </Link>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs font-mono">
              <thead>
                <tr className="border-b border-slate-800 text-slate-500">
                  <th className="pb-3">TRACE ID</th>
                  <th className="pb-3">ROOT AGENT</th>
                  <th className="pb-3">ACTION</th>
                  <th className="pb-3">DURATION</th>
                  <th className="pb-3">COST</th>
                  <th className="pb-3">STATUS</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60">
                {recentTraces.map((t) => (
                  <tr key={t.id} className="hover:bg-slate-900/40 transition">
                    <td className="py-3 text-slate-300">{t.id}</td>
                    <td className="py-3 text-white font-medium">{t.root}</td>
                    <td className="py-3 text-slate-400">{t.action}</td>
                    <td className="py-3 text-slate-400">{t.duration}</td>
                    <td className="py-3 text-emerald-400">{t.cost}</td>
                    <td className="py-3">
                      <span className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                        t.status === "SUCCESS" ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20" : "bg-amber-500/10 text-amber-400 border border-amber-500/20"
                      }`}>
                        {t.status}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Reliability & Circuit Breakers */}
        <div className="glass-panel p-6">
          <h2 className="text-base font-bold text-white mb-4 flex items-center gap-2">
            <ShieldCheck className="w-4 h-4 text-cyan-400" /> Circuit Breakers
          </h2>
          <div className="space-y-3">
            {circuitBreakers.map((cb, idx) => (
              <div key={idx} className="p-3 rounded-lg bg-slate-950/60 border border-slate-800 flex items-center justify-between text-xs font-mono">
                <div>
                  <div className="font-semibold text-white">{cb.target}</div>
                  <div className="text-[10px] text-slate-500">{cb.type}</div>
                </div>
                <div className="text-right">
                  <span className="px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-bold text-[10px]">
                    {cb.state}
                  </span>
                  <div className="text-[10px] text-slate-500 mt-1">{cb.lastProbe}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
