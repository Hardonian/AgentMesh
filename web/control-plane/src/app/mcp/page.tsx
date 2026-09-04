import Link from "next/link";
import { ArrowLeft, CheckCircle2, Layers, ShieldCheck, Wrench } from "lucide-react";

export default function MCPGovernanceOverviewPage() {
  return (
    <div className="min-h-screen bg-[#090d16] text-slate-100 p-8 max-w-6xl mx-auto space-y-10">
      <div className="flex items-center justify-between border-b border-slate-800/80 pb-6">
        <div className="flex items-center gap-3">
          <Link href="/dashboard" className="p-2 rounded-lg bg-slate-900 text-slate-400 hover:text-white transition">
            <ArrowLeft className="w-4 h-4" />
          </Link>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-2xl font-bold text-white tracking-tight">Governed MCP Tool Gateway</h1>
              <span className="px-2.5 py-0.5 rounded-full text-xs font-mono bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">
                JSON-RPC 2.0 / 2024-11-05
              </span>
            </div>
            <p className="text-xs text-slate-400 mt-0.5">
              Deterministic authorization, stable tool fingerprints, schema drift detection, and Google-managed MCP policies.
            </p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="glass-panel p-6 space-y-2">
          <h3 className="text-sm font-bold text-white flex items-center gap-2">
            <ShieldCheck className="w-4 h-4 text-emerald-400" />
            Risk Classification
          </h3>
          <p className="text-xs text-slate-400">
            Categorizes tools into READ, WRITE, DESTRUCTIVE, and FINANCIAL. High-risk actions require explicit admin policy or human-in-the-loop approvals.
          </p>
        </div>
        <div className="glass-panel p-6 space-y-2">
          <h3 className="text-sm font-bold text-white flex items-center gap-2">
            <Wrench className="w-4 h-4 text-cyan-400" />
            Schema Drift Safety
          </h3>
          <p className="text-xs text-slate-400">
            Detects added required fields (BREAKING) or removed properties (POTENTIALLY_BREAKING), flagging affected agent policies and evaluation suites as stale.
          </p>
        </div>
        <div className="glass-panel p-6 space-y-2">
          <h3 className="text-sm font-bold text-white flex items-center gap-2">
            <Layers className="w-4 h-4 text-amber-400" />
            Google Managed MCP
          </h3>
          <p className="text-xs text-slate-400">
            Governs Google Cloud MCP endpoints (BigQuery, Cloud Storage, Maps, GKE) with turnkey templates for read-only analytics, write approvals, and regional isolation.
          </p>
        </div>
      </div>
    </div>
  );
}
