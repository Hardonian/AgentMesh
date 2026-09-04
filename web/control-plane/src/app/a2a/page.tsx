import Link from "next/link";
import { ArrowLeft, CheckCircle2, Award, Zap, Activity, ShieldCheck } from "lucide-react";

export default function A2APage() {
  const matrix = [
    { runtime: "Google ADK (Go)", discovery: "COMPATIBLE", card: "COMPATIBLE", invoke: "COMPATIBLE", stream: "COMPATIBLE", cancel: "COMPATIBLE", auth: "COMPATIBLE" },
    { runtime: "Custom Go A2A", discovery: "COMPATIBLE", card: "COMPATIBLE", invoke: "COMPATIBLE", stream: "PARTIAL", cancel: "COMPATIBLE", auth: "COMPATIBLE" },
    { runtime: "Python LangGraph A2A", discovery: "COMPATIBLE", card: "COMPATIBLE", invoke: "COMPATIBLE", stream: "COMPATIBLE", cancel: "PARTIAL", auth: "COMPATIBLE" },
  ];

  return (
    <div className="min-h-screen bg-[#090d16] text-slate-100 p-8 max-w-6xl mx-auto space-y-10">
      <div className="flex items-center justify-between border-b border-slate-800/80 pb-6">
        <div className="flex items-center gap-3">
          <Link href="/dashboard" className="p-2 rounded-lg bg-slate-900 text-slate-400 hover:text-white transition">
            <ArrowLeft className="w-4 h-4" />
          </Link>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-2xl font-bold text-white tracking-tight">AgentMesh A2A Compatibility Lab</h1>
              <span className="px-2.5 py-0.5 rounded-full text-xs font-mono bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">
                A2A Protocol v0.3.0
              </span>
            </div>
            <p className="text-xs text-slate-400 mt-0.5">
              Objective protocol conformance verification across discovery, Agent Cards, streaming, and cancellation.
            </p>
          </div>
        </div>
      </div>

      {/* Interoperability Matrix */}
      <div className="glass-panel p-6 space-y-4">
        <h2 className="text-base font-bold text-white flex items-center gap-2">
          <Award className="w-4 h-4 text-emerald-400" /> Protocol Conformance Matrix
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs font-mono">
            <thead>
              <tr className="border-b border-slate-800 text-slate-400">
                <th className="py-2 pr-4">AGENT RUNTIME</th>
                <th className="py-2 px-4">DISCOVERY</th>
                <th className="py-2 px-4">AGENT CARD</th>
                <th className="py-2 px-4">INVOCATION</th>
                <th className="py-2 px-4">STREAMING</th>
                <th className="py-2 px-4">CANCELLATION</th>
                <th className="py-2 px-4">AUTH / TENANT</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {matrix.map((row) => (
                <tr key={row.runtime} className="hover:bg-slate-900/40">
                  <td className="py-3 pr-4 font-bold text-white">{row.runtime}</td>
                  <td className="py-3 px-4 text-emerald-400">COMPATIBLE</td>
                  <td className="py-3 px-4 text-emerald-400">COMPATIBLE</td>
                  <td className="py-3 px-4 text-emerald-400">COMPATIBLE</td>
                  <td className="py-3 px-4 text-emerald-400">{row.stream}</td>
                  <td className="py-3 px-4 text-emerald-400">{row.cancel}</td>
                  <td className="py-3 px-4 text-emerald-400">COMPATIBLE</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
