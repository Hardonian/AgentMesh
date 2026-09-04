"use client";

import { useState } from "react";
import { 
  GitFork, 
  ShieldAlert, 
  ShieldCheck, 
  Zap, 
  Clock, 
  DollarSign, 
  AlertTriangle, 
  Filter, 
  Search, 
  ArrowRight, 
  Layers, 
  Wrench, 
  Cpu, 
  CheckCircle2, 
  XCircle,
  HelpCircle
} from "lucide-react";

interface GraphNode {
  id: string;
  name: string;
  type: "AGENT" | "WORKFLOW_STEP" | "TOOL" | "MODEL" | "HITL_APPROVAL";
  target?: string;
  latencyMs: number;
  costUSD: number;
  status: "HEALTHY" | "DEGRADED";
  riskSeverity?: "LOW" | "MEDIUM" | "HIGH" | "CRITICAL";
  riskNote?: string;
}

interface GraphEdge {
  from: string;
  to: string;
  label?: string;
  policyAllowed: boolean;
  policyReason?: string;
}

interface GraphData {
  id: string;
  agentId: string;
  version: string;
  hash: string;
  entrypoint: string;
  overallRisk: "LOW" | "MEDIUM" | "HIGH" | "CRITICAL";
  policyCompliant: boolean;
  nodes: GraphNode[];
  edges: GraphEdge[];
}

const mockGraphs: Record<string, GraphData> = {
  "graph_procurement_agent": {
    id: "graph_procurement_agent",
    agentId: "procurement-agent",
    version: "1.0.0",
    hash: "a4f89d31b2e106c59b48c772e0a293817f0932c64e81fa214309a90184b2cd5e",
    entrypoint: "entry_intent_router",
    overallRisk: "MEDIUM",
    policyCompliant: true,
    nodes: [
      { id: "entry_intent_router", name: "Intent Router", type: "WORKFLOW_STEP", latencyMs: 45, costUSD: 0.0002, status: "HEALTHY" },
      { id: "gemini_flash", name: "Gemini 1.5 Flash", type: "MODEL", target: "models/gemini-1.5-flash", latencyMs: 190, costUSD: 0.0008, status: "HEALTHY" },
      { id: "vendor_search", name: "Vendor Search Tool", type: "TOOL", target: "bigquery.read", latencyMs: 310, costUSD: 0.0040, status: "HEALTHY", riskSeverity: "LOW", riskNote: "Read-only analytics query" },
      { id: "quote_evaluator", name: "Quote Evaluator Agent", type: "AGENT", target: "analyst-agent", latencyMs: 520, costUSD: 0.0035, status: "HEALTHY" },
      { id: "procurement_approval", name: "Procurement Lead HITL", type: "HITL_APPROVAL", latencyMs: 12000, costUSD: 0.0, status: "HEALTHY", riskSeverity: "MEDIUM", riskNote: "Required for expenditures > $500" },
      { id: "erp_po_create", name: "ERP PO Create Tool", type: "TOOL", target: "sap.po.create", latencyMs: 420, costUSD: 0.0010, status: "HEALTHY", riskSeverity: "HIGH", riskNote: "Write capability with financial consequence" },
    ],
    edges: [
      { from: "entry_intent_router", to: "gemini_flash", label: "parse_request", policyAllowed: true },
      { from: "gemini_flash", to: "vendor_search", label: "query_vendors", policyAllowed: true },
      { from: "vendor_search", to: "quote_evaluator", label: "delegate_analysis", policyAllowed: true },
      { from: "quote_evaluator", to: "procurement_approval", label: "quote > $500", policyAllowed: true },
      { from: "procurement_approval", to: "erp_po_create", label: "on_approved", policyAllowed: true },
    ]
  },
  "graph_finance_agent": {
    id: "graph_finance_agent",
    agentId: "finance-agent",
    version: "1.1.0",
    hash: "6b38c01e51f8a8479e3c27da509b552309f482a229cb0184c281048b6d3910c2",
    entrypoint: "reconcile_entry",
    overallRisk: "HIGH",
    policyCompliant: false,
    nodes: [
      { id: "reconcile_entry", name: "Reconciliation Ingestion", type: "WORKFLOW_STEP", latencyMs: 30, costUSD: 0.0001, status: "HEALTHY" },
      { id: "bq_query", name: "Ledger Query Tool", type: "TOOL", target: "bigquery.read", latencyMs: 280, costUSD: 0.0030, status: "HEALTHY" },
      { id: "delegated_researcher", name: "Research Assistant", type: "AGENT", target: "research-agent", latencyMs: 840, costUSD: 0.0062, status: "HEALTHY" },
      { id: "gmail_send", name: "Gmail Dispatch Tool", type: "TOOL", target: "gmail.send", latencyMs: 210, costUSD: 0.0005, status: "DEGRADED", riskSeverity: "CRITICAL", riskNote: "Forbidden indirect privilege escalation path detected" },
    ],
    edges: [
      { from: "reconcile_entry", to: "bq_query", label: "fetch_ledger", policyAllowed: true },
      { from: "bq_query", to: "delegated_researcher", label: "lookup_discrepancy", policyAllowed: true },
      { from: "delegated_researcher", to: "gmail_send", label: "notify_vendor", policyAllowed: false, policyReason: "Policy 'corp-anti-exfiltration' forbids indirect delegation from finance-agent to external communication tools" },
    ]
  }
};

export default function GraphVisualizerPage() {
  const [selectedGraphId, setSelectedGraphId] = useState<string>("graph_procurement_agent");
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>("vendor_search");
  const [filterType, setFilterType] = useState<string>("ALL");

  const graph = mockGraphs[selectedGraphId] || mockGraphs["graph_procurement_agent"];
  const selectedNode = graph.nodes.find((n) => n.id === selectedNodeId) || graph.nodes[0];

  const filteredNodes = filterType === "ALL" 
    ? graph.nodes 
    : graph.nodes.filter((n) => n.type === filterType);

  const getNodeColor = (type: GraphNode["type"]) => {
    switch (type) {
      case "AGENT": return "border-emerald-500/50 bg-emerald-950/30 text-emerald-300";
      case "TOOL": return "border-cyan-500/50 bg-cyan-950/30 text-cyan-300";
      case "MODEL": return "border-amber-500/50 bg-amber-950/30 text-amber-300";
      case "HITL_APPROVAL": return "border-rose-500/50 bg-rose-950/30 text-rose-300";
      default: return "border-slate-700 bg-slate-900/60 text-slate-300";
    }
  };

  const getRiskBadge = (level?: string) => {
    switch (level) {
      case "CRITICAL": return <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-rose-500/20 text-rose-400 border border-rose-500/40 font-bold">CRITICAL RISK</span>;
      case "HIGH": return <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-orange-500/20 text-orange-400 border border-orange-500/40">HIGH RISK</span>;
      case "MEDIUM": return <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-amber-500/20 text-amber-400 border border-amber-500/40">MEDIUM RISK</span>;
      default: return <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-emerald-500/20 text-emerald-400 border border-emerald-500/40">LOW RISK</span>;
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-white tracking-tight">ADK AgentGraph Intelligence</h1>
            <span className="px-2.5 py-0.5 rounded-full text-xs font-mono bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">
              ADK AST Normalizer v2.0
            </span>
          </div>
          <p className="text-sm text-slate-400 mt-1">
            Static operational topology, taint-propagated delegation safety, and indirect privilege escalation analysis.
          </p>
        </div>

        {/* Graph Switcher */}
        <div className="flex items-center gap-2">
          <label className="text-xs font-mono text-slate-400">Target Graph:</label>
          <select 
            value={selectedGraphId} 
            onChange={(e) => {
              setSelectedGraphId(e.target.value);
              setSelectedNodeId(null);
            }}
            className="bg-slate-900 border border-slate-700 text-slate-200 text-xs rounded-lg px-3 py-1.5 focus:outline-none focus:border-emerald-500 font-mono"
          >
            <option value="graph_procurement_agent">procurement-agent (v1.0.0)</option>
            <option value="graph_finance_agent">finance-agent (v1.1.0 - Policy Violation)</option>
          </select>
        </div>
      </div>

      {/* Graph Summary Banner */}
      <div className="glass-panel p-4 flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-6">
          <div>
            <div className="text-[10px] text-slate-500 font-mono">CANONICAL GRAPH ID</div>
            <div className="text-sm font-mono text-slate-200 font-medium">{graph.id}</div>
          </div>
          <div>
            <div className="text-[10px] text-slate-500 font-mono">AGENT ID</div>
            <div className="text-sm font-mono text-emerald-400">{graph.agentId}</div>
          </div>
          <div>
            <div className="text-[10px] text-slate-500 font-mono">DETERMINISTIC HASH</div>
            <div className="text-xs font-mono text-slate-400 truncate max-w-[200px]" title={graph.hash}>{graph.hash}</div>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1.5 px-3 py-1 rounded-lg bg-slate-900/80 border border-slate-800 text-xs font-mono">
            <span>Graph Risk:</span>
            {getRiskBadge(graph.overallRisk)}
          </div>
          <div className="flex items-center gap-1.5 px-3 py-1 rounded-lg bg-slate-900/80 border border-slate-800 text-xs font-mono">
            <span>Policy Status:</span>
            {graph.policyCompliant ? (
              <span className="flex items-center gap-1 text-emerald-400 font-bold">
                <CheckCircle2 className="w-3.5 h-3.5" /> COMPLIANT
              </span>
            ) : (
              <span className="flex items-center gap-1 text-rose-400 font-bold">
                <XCircle className="w-3.5 h-3.5" /> VIOLATION DETECTED
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Visualizer Canvas & Node Inspector Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Canvas / Topology Viewer */}
        <div className="lg:col-span-2 glass-panel p-6 flex flex-col justify-between min-h-[520px]">
          <div>
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <GitFork className="w-4 h-4 text-emerald-400" />
                <span className="text-sm font-bold text-white">Execution Topology Flow</span>
              </div>
              <div className="flex items-center gap-2">
                <Filter className="w-3.5 h-3.5 text-slate-500" />
                <select 
                  value={filterType} 
                  onChange={(e) => setFilterType(e.target.value)}
                  className="bg-slate-900/80 border border-slate-800 text-slate-300 text-xs rounded px-2 py-1 focus:outline-none"
                >
                  <option value="ALL">All Nodes</option>
                  <option value="TOOL">Tools Only</option>
                  <option value="AGENT">Delegated Agents</option>
                  <option value="HITL_APPROVAL">Approvals</option>
                  <option value="MODEL">Models</option>
                </select>
              </div>
            </div>

            {/* Simulated Directed Graph Visual Flow */}
            <div className="space-y-4 py-4">
              {filteredNodes.map((node, index) => {
                const isSelected = node.id === selectedNodeId;
                const edgeOut = graph.edges.find((e) => e.from === node.id);

                return (
                  <div key={node.id} className="relative">
                    {/* Node Card */}
                    <div 
                      onClick={() => setSelectedNodeId(node.id)}
                      className={`cursor-pointer border rounded-xl p-4 transition-all duration-200 flex items-center justify-between ${
                        isSelected 
                          ? "ring-2 ring-emerald-400 border-emerald-400/80 bg-slate-900/90 shadow-lg shadow-emerald-500/10" 
                          : "border-slate-800/80 bg-slate-950/50 hover:border-slate-700"
                      }`}
                    >
                      <div className="flex items-center gap-3">
                        <div className={`p-2 rounded-lg border text-xs font-mono font-bold ${getNodeColor(node.type)}`}>
                          {node.type === "TOOL" && <Wrench className="w-4 h-4" />}
                          {node.type === "AGENT" && <GitFork className="w-4 h-4" />}
                          {node.type === "MODEL" && <Cpu className="w-4 h-4" />}
                          {node.type === "HITL_APPROVAL" && <ShieldAlert className="w-4 h-4" />}
                          {node.type === "WORKFLOW_STEP" && <Layers className="w-4 h-4" />}
                        </div>
                        <div>
                          <div className="text-sm font-semibold text-white flex items-center gap-2">
                            {node.name}
                            {node.id === graph.entrypoint && (
                              <span className="text-[10px] bg-emerald-500/20 text-emerald-400 px-1.5 py-0.5 rounded font-mono">
                                ENTRYPOINT
                              </span>
                            )}
                          </div>
                          <div className="text-xs text-slate-400 font-mono">
                            id: {node.id} {node.target ? `-> ${node.target}` : ""}
                          </div>
                        </div>
                      </div>

                      <div className="flex items-center gap-4">
                        <div className="text-right">
                          <div className="text-xs font-mono text-slate-300 flex items-center gap-1 justify-end">
                            <Clock className="w-3 h-3 text-slate-500" /> {node.latencyMs}ms
                          </div>
                          <div className="text-[11px] font-mono text-slate-500 flex items-center gap-1 justify-end">
                            <DollarSign className="w-3 h-3 text-slate-600" /> ${node.costUSD.toFixed(4)}
                          </div>
                        </div>
                        {node.riskSeverity && getRiskBadge(node.riskSeverity)}
                      </div>
                    </div>

                    {/* Edge Connector Indicator */}
                    {edgeOut && index < filteredNodes.length - 1 && (
                      <div className="flex items-center justify-center my-1.5">
                        <div className="flex items-center gap-2 px-2.5 py-0.5 rounded bg-slate-900 border border-slate-800 text-[10px] font-mono text-slate-400">
                          <span>{edgeOut.label || "invokes"}</span>
                          {edgeOut.policyAllowed ? (
                            <CheckCircle2 className="w-3 h-3 text-emerald-400" />
                          ) : (
                            <span className="text-rose-400 font-bold flex items-center gap-1">
                              <XCircle className="w-3 h-3" /> POLICY DENIED
                            </span>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </div>

          {/* Performance Critical Path Footer */}
          <div className="pt-4 border-t border-slate-800/80 flex items-center justify-between text-xs text-slate-400">
            <span className="font-mono">Critical Path Duration: <strong className="text-white">1,240ms</strong></span>
            <span className="font-mono">Total Estimated Spend: <strong className="text-emerald-400">$0.0095</strong></span>
          </div>
        </div>

        {/* Node & Risk Inspector Panel */}
        <div className="glass-panel p-6 flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between pb-4 border-b border-slate-800">
              <h2 className="text-sm font-bold text-white flex items-center gap-2">
                <ShieldCheck className="w-4 h-4 text-emerald-400" /> Node & Policy Inspector
              </h2>
              <span className="text-xs font-mono text-slate-500">{selectedNode.id}</span>
            </div>

            <div className="mt-4 space-y-4">
              <div>
                <label className="text-[10px] font-mono text-slate-500 uppercase">Node Title & Target</label>
                <div className="text-base font-bold text-white">{selectedNode.name}</div>
                <div className="text-xs font-mono text-cyan-400 mt-0.5">{selectedNode.target || "Local Execution Unit"}</div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="p-3 rounded-lg bg-slate-900/60 border border-slate-800">
                  <div className="text-[10px] text-slate-500 font-mono">NODE TYPE</div>
                  <div className="text-xs font-bold text-slate-200 mt-1">{selectedNode.type}</div>
                </div>
                <div className="p-3 rounded-lg bg-slate-900/60 border border-slate-800">
                  <div className="text-[10px] text-slate-500 font-mono">EXECUTION SLO</div>
                  <div className="text-xs font-bold text-emerald-400 mt-1">{selectedNode.latencyMs}ms latency</div>
                </div>
              </div>

              {/* Risk Finding */}
              {selectedNode.riskNote && (
                <div className="p-3.5 rounded-lg bg-amber-500/10 border border-amber-500/20 text-xs">
                  <div className="flex items-center gap-1.5 text-amber-400 font-bold mb-1">
                    <AlertTriangle className="w-4 h-4" /> Static Graph Finding
                  </div>
                  <div className="text-slate-300 leading-relaxed">{selectedNode.riskNote}</div>
                </div>
              )}

              {/* Policy Explanation */}
              <div className="p-3.5 rounded-lg bg-slate-900/90 border border-slate-800 text-xs space-y-2">
                <div className="text-slate-400 font-semibold flex items-center justify-between">
                  <span>Delegation Taint Analysis:</span>
                  <span className="text-emerald-400 font-mono">ATTRIBUTED</span>
                </div>
                <p className="text-slate-400 leading-relaxed text-[11px]">
                  Principal authorization context propagates through all inbound delegation edges. Privilege reduction is strictly enforced; delegation cannot widen tool access.
                </p>
              </div>
            </div>
          </div>

          <div className="mt-6 pt-4 border-t border-slate-800">
            <button 
              onClick={() => alert(`Simulating static execution path for node ${selectedNode.id}`)}
              className="w-full py-2 px-3 rounded-lg bg-emerald-500/20 text-emerald-400 border border-emerald-500/40 text-xs font-mono font-bold hover:bg-emerald-500/30 transition text-center"
            >
              Simulate Policy Decision for Node
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
