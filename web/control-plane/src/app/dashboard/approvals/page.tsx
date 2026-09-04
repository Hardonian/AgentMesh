import { CheckSquare, AlertTriangle, Check, X, Shield, Lock } from "lucide-react";

export default function ApprovalsInboxPage() {
  const pendingApprovals = [
    {
      id: "appr_1ff77cc4",
      agentId: "procurement-agent",
      tool: "bigquery.delete",
      action: "delete",
      reason: "Human approval required by rule 'Require Approval for BigQuery Delete'",
      parametersHash: "4a8c9e01f2...5b88",
      createdAt: "2 minutes ago",
      expiresIn: "13 minutes",
      parameters: {
        dataset: "raw_archive",
        table: "q3_logs",
        retentionPolicy: "purge_immediate",
      },
    },
  ];

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold text-white tracking-tight">Human Approvals Inbox</h1>
        <p className="text-sm text-slate-400">
          Cryptographically bound action authorizations for sensitive and destructive tool executions.
        </p>
      </div>

      <div className="p-4 rounded-lg bg-amber-950/20 border border-amber-500/30 text-xs text-amber-300 flex items-center gap-3">
        <AlertTriangle className="w-5 h-5 text-amber-400 shrink-0" />
        <div>
          <strong className="font-semibold">Security Boundary:</strong> Approvals cryptographically bind to the exact agent, tool, parameter hash, and policy version. Any parameter tampering immediately invalidates the authorization token.
        </div>
      </div>

      <div className="space-y-4">
        {pendingApprovals.map((req) => (
          <div key={req.id} className="glass-panel p-6 border-amber-500/20">
            <div className="flex flex-col md:flex-row md:items-center justify-between pb-4 border-b border-slate-800 gap-4">
              <div>
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-xs font-mono text-slate-500">REQUEST ID:</span>
                  <span className="text-xs font-mono font-bold text-white">{req.id}</span>
                  <span className="px-2 py-0.5 rounded bg-amber-500/10 text-amber-400 border border-amber-500/20 text-[10px] font-bold">
                    PENDING APPROVAL
                  </span>
                </div>
                <div className="text-base font-bold text-white">
                  Agent <span className="text-emerald-400 font-mono">{req.agentId}</span> requested execution of <span className="text-purple-400 font-mono">{req.tool}</span>
                </div>
                <div className="text-xs text-slate-400 mt-1">{req.reason}</div>
              </div>

              <div className="flex items-center gap-3">
                <button className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-bold text-xs flex items-center gap-1.5 transition shadow-lg shadow-emerald-500/10">
                  <Check className="w-4 h-4" /> Approve Execution
                </button>
                <button className="px-4 py-2 rounded-lg bg-red-500/20 hover:bg-red-500/30 text-red-400 border border-red-500/30 font-bold text-xs flex items-center gap-1.5 transition">
                  <X className="w-4 h-4" /> Reject
                </button>
              </div>
            </div>

            <div className="pt-4 grid grid-cols-1 md:grid-cols-2 gap-4 text-xs font-mono">
              <div className="p-3 rounded bg-slate-950 border border-slate-800">
                <div className="text-slate-500 mb-2 font-bold uppercase text-[10px] flex items-center gap-1">
                  <Lock className="w-3 h-3 text-emerald-400" /> Immutable Parameters Hash
                </div>
                <div className="text-slate-300 break-all">{req.parametersHash}</div>
                <div className="text-[10px] text-slate-500 mt-2">Expires in: {req.expiresIn}</div>
              </div>

              <div className="p-3 rounded bg-slate-950 border border-slate-800">
                <div className="text-slate-500 mb-2 font-bold uppercase text-[10px]">Action Parameters Payload</div>
                <pre className="text-emerald-400 text-[11px] overflow-x-auto">
                  {JSON.stringify(req.parameters, null, 2)}
                </pre>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
