import Link from "next/link";
import { 
  Cpu, 
  Layers, 
  ShieldCheck, 
  GitFork, 
  Cloud, 
  Activity, 
  ArrowLeft, 
  CheckCircle2, 
  ExternalLink 
} from "lucide-react";

export default function GoogleDeepeningPage() {
  const pillars = [
    {
      title: "Google ADK Graph Intelligence",
      description: "Static AST inspection of Google ADK Go agents without untrusted code execution. Generates canonical AgentGraphs, analyzes 9 risk dimensions, and enforces anti-privilege escalation delegation taint propagation.",
      badge: "Deep Integration",
      icon: GitFork,
    },
    {
      title: "Gemini & Vertex AI Multi-Model Routing",
      description: "Dynamic routing across Gemini 1.5 Pro, Flash, and Vertex AI endpoints with location, quota, cost, and latency-aware fallback. Never silently falls back across model families without policy authorization.",
      badge: "Production Routing",
      icon: Cpu,
    },
    {
      title: "Google-Managed MCP Governance",
      description: "Native discovery and policy templates for Google-managed MCP services (BigQuery, Cloud Storage, Maps, GKE). Enforces read-only, write approvals, and regional isolation policies.",
      badge: "MCP Gateway",
      icon: Layers,
    },
    {
      title: "Cloud Run & GKE Production Patterns",
      description: "Declarative deployment plans, Workload Identity federation without static JSON keys, and optional Kubernetes operator reconciliation for AgentMeshAgent, AgentMeshPolicy, and AgentMeshRoute CRDs.",
      badge: "Enterprise Ready",
      icon: Cloud,
    },
    {
      title: "Cloud Trace & OpenTelemetry",
      description: "Distributed trace context propagation across A2A, MCP, and model calls mapping cleanly into Google Cloud Observability without vendor lock-in.",
      badge: "Observability",
      icon: Activity,
    },
    {
      title: "Google-Aligned, Vendor-Neutral",
      description: "While engineered for the deepest integration with Google's agent ecosystem, AgentMesh remains independently useful and fully operable on-premises or across any cloud provider.",
      badge: "Architecture",
      icon: ShieldCheck,
    }
  ];

  return (
    <div className="min-h-screen bg-[#090d16] text-slate-100 p-8 max-w-6xl mx-auto space-y-12">
      <div className="flex items-center justify-between border-b border-slate-800/80 pb-6">
        <div className="flex items-center gap-3">
          <Link href="/dashboard" className="p-2 rounded-lg bg-slate-900 text-slate-400 hover:text-white transition">
            <ArrowLeft className="w-4 h-4" />
          </Link>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-2xl font-bold text-white tracking-tight">Google-Native AgentMesh Architecture</h1>
              <span className="px-2.5 py-0.5 rounded-full text-xs font-mono bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                Phase 2 Complete
              </span>
            </div>
            <p className="text-xs text-slate-400 mt-0.5">
              Production intelligence and governance layer for Google ADK, Gemini, Vertex AI, and Google Cloud MCP.
            </p>
          </div>
        </div>

        <Link 
          href="/dashboard/graphs"
          className="px-4 py-2 rounded-lg bg-emerald-500 text-slate-950 font-bold text-xs font-mono hover:bg-emerald-400 transition"
        >
          View ADK Graphs
        </Link>
      </div>

      {/* Pillars Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {pillars.map((p, idx) => {
          const Icon = p.icon;
          return (
            <div key={idx} className="glass-panel p-6 space-y-3">
              <div className="flex items-center justify-between">
                <div className="p-2.5 rounded-lg bg-slate-900 text-emerald-400 border border-slate-800">
                  <Icon className="w-5 h-5" />
                </div>
                <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-slate-800/80 text-cyan-400 border border-slate-700/60">
                  {p.badge}
                </span>
              </div>
              <h3 className="text-base font-bold text-white tracking-tight">{p.title}</h3>
              <p className="text-xs text-slate-400 leading-relaxed">{p.description}</p>
            </div>
          );
        })}
      </div>

      {/* Reference Architecture Diagram Callout */}
      <div className="glass-panel p-8 space-y-4">
        <h2 className="text-lg font-bold text-white flex items-center gap-2">
          <CheckCircle2 className="w-5 h-5 text-emerald-400" />
          Production Google Cloud Deployment Flow
        </h2>
        <div className="p-4 rounded-xl bg-slate-950 border border-slate-800 font-mono text-xs text-slate-300 overflow-x-auto">
          <code>
            ADK Go Agents (Cloud Run / GKE) <br />
            &nbsp;&nbsp;&nbsp;&nbsp;↓ (Workload Identity / mTLS) <br />
            AgentMesh Proxy & Control Plane <br />
            &nbsp;&nbsp;&nbsp;&nbsp;↓ (Deterministic Policy & Semantic Auth) <br />
            [ Gemini 1.5 / Vertex AI ] &nbsp;&nbsp;&nbsp;&nbsp; [ Google Managed MCP (BigQuery/GCS) ] &nbsp;&nbsp;&nbsp;&nbsp; [ A2A Mesh Agents ] <br />
            &nbsp;&nbsp;&nbsp;&nbsp;↓ (OpenTelemetry) <br />
            Google Cloud Trace & Cloud Logging (Audit Correlated)
          </code>
        </div>
      </div>
    </div>
  );
}
