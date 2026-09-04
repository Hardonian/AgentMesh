import Link from "next/link";
import { Shield, Cpu, Network, ArrowRight, CheckCircle2, Lock, Terminal, Activity, Layers } from "lucide-react";

export default function LandingPage() {
  return (
    <div className="min-h-screen flex flex-col justify-between selection:bg-emerald-500 selection:text-black">
      {/* Header */}
      <header className="border-b border-slate-800/80 bg-slate-950/40 backdrop-blur-md sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-6 h-16 flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-tr from-emerald-500 to-cyan-500 flex items-center justify-center font-bold text-slate-950">
              M
            </div>
            <span className="font-bold text-xl tracking-tight text-white">Agent<span className="text-emerald-400">Mesh</span></span>
            <span className="text-xs px-2 py-0.5 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 font-mono">v1.0.0</span>
          </div>
          <nav className="flex items-center space-x-6">
            <Link href="/dashboard" className="text-sm text-slate-300 hover:text-white transition">Dashboard</Link>
            <Link href="/dashboard/agents" className="text-sm text-slate-300 hover:text-white transition">Passports</Link>
            <Link href="/dashboard/traces" className="text-sm text-slate-300 hover:text-white transition">Traces</Link>
            <Link href="/dashboard/approvals" className="text-sm text-slate-300 hover:text-white transition">Approvals</Link>
            <Link href="/dashboard" className="px-4 py-2 text-sm font-medium bg-emerald-500 hover:bg-emerald-400 text-slate-950 rounded-lg transition font-semibold shadow-lg shadow-emerald-500/20">
              Open Control Plane
            </Link>
          </nav>
        </div>
      </header>

      {/* Hero Section */}
      <main className="max-w-7xl mx-auto px-6 pt-20 pb-16 flex-1">
        <div className="text-center max-w-4xl mx-auto mb-16">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-950/50 border border-emerald-500/30 text-emerald-400 text-xs font-mono mb-6">
            <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
            OPEN-CORE GO-NATIVE DATA & CONTROL PLANE
          </div>
          <h1 className="text-5xl md:text-6xl font-extrabold tracking-tight text-white mb-6 leading-tight">
            The open control plane for <span className="gradient-text">A2A and MCP agents</span>.
          </h1>
          <p className="text-xl text-slate-400 mb-10 max-w-2xl mx-auto leading-relaxed">
            Identity, deterministic policy, capability routing, reliability, and progressive delivery for production AI agent systems.
          </p>

          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link href="/dashboard" className="w-full sm:w-auto px-8 py-3.5 bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-bold rounded-xl flex items-center justify-center gap-2 transition shadow-xl shadow-emerald-500/20">
              Launch Dashboard <ArrowRight className="w-4 h-4" />
            </Link>
            <a href="#quickstart" className="w-full sm:w-auto px-8 py-3.5 bg-slate-900/80 hover:bg-slate-800 text-white font-medium border border-slate-700/60 rounded-xl flex items-center justify-center gap-2 transition">
              <Terminal className="w-4 h-4 text-emerald-400" /> Start Locally (<code className="text-xs font-mono text-emerald-400">make dev</code>)
            </a>
          </div>
        </div>

        {/* Architecture Visualizer */}
        <div className="glass-panel p-8 max-w-5xl mx-auto mb-20 shadow-2xl relative overflow-hidden border border-slate-700/50">
          <div className="absolute top-0 right-0 w-96 h-96 bg-emerald-500/5 rounded-full blur-3xl -z-10" />
          <div className="flex items-center justify-between border-b border-slate-800 pb-4 mb-8">
            <div className="flex items-center gap-2">
              <Layers className="w-5 h-5 text-emerald-400" />
              <span className="font-semibold text-white">AgentMesh Topographical Control Architecture</span>
            </div>
            <span className="text-xs text-slate-400 font-mono">Google ADK • Gemini • Vertex AI • Multi-Vendor</span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 text-center">
            {/* Layer 1: Agents */}
            <div className="p-6 rounded-xl bg-slate-900/60 border border-slate-800 flex flex-col items-center">
              <div className="w-12 h-12 rounded-lg bg-blue-500/10 border border-blue-500/20 flex items-center justify-center text-blue-400 mb-3">
                <Cpu className="w-6 h-6" />
              </div>
              <h3 className="font-bold text-white mb-1">Autonomous Agents</h3>
              <p className="text-xs text-slate-400 mb-3">Google ADK Go • LangGraph • Custom</p>
              <div className="w-full space-y-1.5 text-left text-xs font-mono bg-slate-950 p-2.5 rounded border border-slate-800/80 text-slate-300">
                <div className="text-blue-400 font-semibold">Protocols:</div>
                <div>• A2A (Agent-to-Agent)</div>
                <div>• MCP (Model Context Protocol)</div>
              </div>
            </div>

            {/* Layer 2: AgentMesh Data Plane */}
            <div className="p-6 rounded-xl bg-gradient-to-b from-emerald-950/40 to-slate-900/80 border border-emerald-500/40 flex flex-col items-center shadow-lg relative">
              <div className="absolute -top-3 px-3 py-0.5 bg-emerald-500 text-slate-950 text-[10px] font-bold rounded-full uppercase tracking-wider">
                Control & Data Plane
              </div>
              <div className="w-12 h-12 rounded-lg bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-center text-emerald-400 mb-3">
                <Shield className="w-6 h-6" />
              </div>
              <h3 className="font-bold text-white mb-1">AgentMesh Proxy</h3>
              <p className="text-xs text-emerald-400 mb-3">Sub-millisecond Policy & Routing</p>
              <div className="w-full space-y-1 text-left text-xs font-mono bg-slate-950/80 p-2.5 rounded border border-emerald-500/20 text-slate-300">
                <div className="flex items-center gap-1.5"><CheckCircle2 className="w-3 h-3 text-emerald-400" /> A2A Firewall</div>
                <div className="flex items-center gap-1.5"><CheckCircle2 className="w-3 h-3 text-emerald-400" /> MCPGuard Policy</div>
                <div className="flex items-center gap-1.5"><CheckCircle2 className="w-3 h-3 text-emerald-400" /> Anti-Escalation</div>
                <div className="flex items-center gap-1.5"><CheckCircle2 className="w-3 h-3 text-emerald-400" /> Circuit Breaker</div>
                <div className="flex items-center gap-1.5"><CheckCircle2 className="w-3 h-3 text-emerald-400" /> HITL Approvals</div>
              </div>
            </div>

            {/* Layer 3: Tools & Models */}
            <div className="p-6 rounded-xl bg-slate-900/60 border border-slate-800 flex flex-col items-center">
              <div className="w-12 h-12 rounded-lg bg-purple-500/10 border border-purple-500/20 flex items-center justify-center text-purple-400 mb-3">
                <Network className="w-6 h-6" />
              </div>
              <h3 className="font-bold text-white mb-1">Tools, Models & Cloud</h3>
              <p className="text-xs text-slate-400 mb-3">Managed MCP • BigQuery • Gemini</p>
              <div className="w-full space-y-1.5 text-left text-xs font-mono bg-slate-950 p-2.5 rounded border border-slate-800/80 text-slate-300">
                <div className="text-purple-400 font-semibold">Downstream Targets:</div>
                <div>• BigQuery / Cloud Run / GKE</div>
                <div>• Gemini 2.0 / Vertex AI</div>
                <div>• Peer A2A Agents</div>
              </div>
            </div>
          </div>
        </div>

        {/* Feature Grid */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-20">
          <div className="glass-panel p-6">
            <Shield className="w-8 h-8 text-emerald-400 mb-4" />
            <h3 className="text-lg font-bold text-white mb-2">A2A Firewall & MCPGuard</h3>
            <p className="text-sm text-slate-400 leading-relaxed">
              Deterministic, non-LLM policy enforcement. Granular allow/deny, data classification tagging, and automatic cycle termination across delegation stacks.
            </p>
          </div>

          <div className="glass-panel p-6">
            <Activity className="w-8 h-8 text-cyan-400 mb-4" />
            <h3 className="text-lg font-bold text-white mb-2">Agent Passport & Evidence</h3>
            <p className="text-sm text-slate-400 leading-relaxed">
              Separates declared contract specifications from empirical operational metrics. Measured success rates, P95 latencies, and tool reliability scorecards.
            </p>
          </div>

          <div className="glass-panel p-6">
            <Lock className="w-8 h-8 text-purple-400 mb-4" />
            <h3 className="text-lg font-bold text-white mb-2">Progressive Delivery & Canary</h3>
            <p className="text-sm text-slate-400 leading-relaxed">
              Stage agent revisions (1% to 100% traffic splits). Automatically roll back to last-known-good when latency regressions or error spikes breach SLOs.
            </p>
          </div>
        </div>

        {/* Quickstart Terminal */}
        <div id="quickstart" className="glass-panel p-8 max-w-3xl mx-auto text-left">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-red-500/80 inline-block" />
              <span className="w-3 h-3 rounded-full bg-yellow-500/80 inline-block" />
              <span className="w-3 h-3 rounded-full bg-green-500/80 inline-block" />
              <span className="text-xs font-mono text-slate-400 ml-2">Quickstart in 60 seconds</span>
            </div>
          </div>
          <pre className="bg-slate-950 p-4 rounded-lg font-mono text-sm text-emerald-400 overflow-x-auto leading-relaxed border border-slate-800">
{`# 1. Run local zero-dependency control plane & proxy
make dev

# 2. Validate your first agent contract
agentmesh contract validate agent.contract.yaml

# 3. Test policy evaluation dry-run
agentmesh route explain financial_research`}
          </pre>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-900 bg-slate-950/80 py-8 text-center text-xs text-slate-500">
        <p>© 2026 AgentMesh. Apache 2.0 Open-Core Infrastructure Platform. Google Cloud First • Vendor Neutral.</p>
      </footer>
    </div>
  );
}
