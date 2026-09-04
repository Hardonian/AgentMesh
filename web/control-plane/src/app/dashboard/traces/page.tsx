import { Activity, Clock, DollarSign, ShieldCheck, CheckCircle2, ChevronRight } from "lucide-react";

export default function WaterfallTracesPage() {
  const activeTrace = {
    id: "tr_99a81c0f-b411",
    rootAgent: "procurement-agent",
    taskId: "task-quote-requisition-992",
    totalDuration: "320ms",
    totalCost: "$0.0084",
    status: "SUCCESS",
    spans: [
      {
        id: "sp_1",
        name: "A2A Task Invocation: procurement-agent",
        type: "AGENT_REQUEST",
        subject: "procurement-agent",
        duration: "320ms",
        cost: "$0.0084",
        policy: "ALLOW",
        status: "SUCCESS",
        widthPct: "100%",
        offsetPct: "0%",
        color: "bg-emerald-500",
      },
      {
        id: "sp_2",
        name: "Delegation Hop: finance-agent (budget_check)",
        type: "DELEGATION",
        subject: "finance-agent",
        duration: "140ms",
        cost: "$0.0021",
        policy: "ALLOW",
        status: "SUCCESS",
        widthPct: "45%",
        offsetPct: "15%",
        color: "bg-blue-500",
      },
      {
        id: "sp_3",
        name: "MCP Tool Call: bigquery.read (SELECT customer_rates)",
        type: "TOOL_CALL",
        subject: "bigquery.read",
        duration: "85ms",
        cost: "$0.0015",
        policy: "ALLOW",
        status: "SUCCESS",
        widthPct: "28%",
        offsetPct: "35%",
        color: "bg-purple-500",
      },
      {
        id: "sp_4",
        name: "Model Generation: gemini-1.5-pro",
        type: "MODEL_CALL",
        subject: "gemini-1.5-pro",
        duration: "95ms",
        cost: "$0.0048",
        policy: "PASS",
        status: "SUCCESS",
        widthPct: "30%",
        offsetPct: "65%",
        color: "bg-cyan-500",
      },
    ],
  };

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold text-white tracking-tight">Agent Execution Traces</h1>
        <p className="text-sm text-slate-400">
          Developer-friendly waterfall visualization across multi-agent hops, tools, and models.
        </p>
      </div>

      {/* Trace Overview Header */}
      <div className="glass-panel p-6">
        <div className="flex flex-col md:flex-row md:items-center justify-between pb-6 border-b border-slate-800 gap-4">
          <div>
            <div className="flex items-center gap-2 mb-1">
              <span className="font-mono text-xs text-slate-500">TRACE ID:</span>
              <span className="font-mono text-xs font-bold text-emerald-400">{activeTrace.id}</span>
              <span className="px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 text-[10px] font-bold">
                {activeTrace.status}
              </span>
            </div>
            <div className="text-lg font-bold text-white">Root Task: {activeTrace.taskId}</div>
          </div>

          <div className="flex items-center gap-6 text-xs font-mono">
            <div>
              <div className="text-slate-500 flex items-center gap-1"><Clock className="w-3.5 h-3.5" /> DURATION</div>
              <div className="text-base font-bold text-white mt-0.5">{activeTrace.totalDuration}</div>
            </div>
            <div>
              <div className="text-slate-500 flex items-center gap-1"><DollarSign className="w-3.5 h-3.5 text-emerald-400" /> TOTAL COST</div>
              <div className="text-base font-bold text-emerald-400 mt-0.5">{activeTrace.totalCost}</div>
            </div>
          </div>
        </div>

        {/* Waterfall Timeline Graph */}
        <div className="pt-6 space-y-4">
          <div className="flex justify-between text-[11px] font-mono text-slate-500 pb-2 border-b border-slate-800/80">
            <span>SPAN / OPERATION</span>
            <span>TIMELINE (0ms → 320ms)</span>
          </div>

          {activeTrace.spans.map((span) => (
            <div key={span.id} className="space-y-1.5 font-mono text-xs">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className={`w-2 h-2 rounded-full ${span.color}`} />
                  <span className="text-slate-200 font-medium">{span.name}</span>
                  <span className="text-[10px] text-slate-500">({span.subject})</span>
                </div>
                <div className="flex items-center gap-3 text-slate-400 text-[11px]">
                  <span>{span.duration}</span>
                  <span className="text-emerald-400 font-semibold">{span.cost}</span>
                  <span className="px-1.5 py-0.2 rounded bg-slate-800 text-[10px] text-slate-300">
                    {span.policy}
                  </span>
                </div>
              </div>

              {/* Progress Bar Container */}
              <div className="w-full h-3 bg-slate-950 rounded-full overflow-hidden relative border border-slate-800/80">
                <div
                  className={`h-full rounded-full ${span.color} opacity-80`}
                  style={{
                    width: span.widthPct,
                    marginLeft: span.offsetPct,
                  }}
                />
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
