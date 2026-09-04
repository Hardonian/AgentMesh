import Link from "next/link";
import { 
  LayoutDashboard, 
  Bot, 
  FileCode, 
  Wrench, 
  ShieldCheck, 
  GitBranch, 
  Activity, 
  CheckSquare, 
  History, 
  KeyRound,
  Layers
} from "lucide-react";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const navItems = [
    { label: "Overview", href: "/dashboard", icon: LayoutDashboard },
    { label: "Agents & Passports", href: "/dashboard/agents", icon: Bot },
    { label: "Agent Contracts", href: "/dashboard/contracts", icon: FileCode },
    { label: "Tools & MCPGuard", href: "/dashboard/tools", icon: Wrench },
    { label: "Policy Studio", href: "/dashboard/policies", icon: ShieldCheck },
    { label: "Capability Routing", href: "/dashboard/routes", icon: GitBranch },
    { label: "Waterfall Traces", href: "/dashboard/traces", icon: Activity },
    { label: "Approvals Inbox", href: "/dashboard/approvals", icon: CheckSquare },
    { label: "Canaries & Rollouts", href: "/dashboard/canaries", icon: Layers },
    { label: "Audit Trail", href: "/dashboard/audit", icon: History },
    { label: "API Credentials", href: "/dashboard/settings", icon: KeyRound },
  ];

  return (
    <div className="flex min-h-screen bg-[#090d16] text-slate-100">
      {/* Sidebar */}
      <aside className="w-64 border-r border-slate-800/80 bg-slate-950/60 p-4 flex flex-col justify-between shrink-0">
        <div>
          <div className="flex items-center space-x-3 px-2 py-3 mb-6 border-b border-slate-800/60">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-tr from-emerald-500 to-cyan-500 flex items-center justify-center font-bold text-slate-950">
              M
            </div>
            <div>
              <div className="font-bold text-base tracking-tight text-white leading-none">Agent<span className="text-emerald-400">Mesh</span></div>
              <div className="text-[10px] text-slate-500 font-mono mt-1">Control Plane</div>
            </div>
          </div>

          <nav className="space-y-1">
            {navItems.map((item) => {
              const Icon = item.icon;
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className="flex items-center gap-3 px-3 py-2 text-sm rounded-lg text-slate-400 hover:text-white hover:bg-slate-900/80 transition"
                >
                  <Icon className="w-4 h-4 text-emerald-400" />
                  <span>{item.label}</span>
                </Link>
              );
            })}
          </nav>
        </div>

        <div className="p-3 rounded-lg bg-slate-900/50 border border-slate-800/80 text-xs">
          <div className="flex items-center justify-between text-slate-400 mb-1">
            <span>Cluster Status:</span>
            <span className="text-emerald-400 font-mono font-medium">HEALTHY</span>
          </div>
          <div className="flex items-center justify-between text-slate-400">
            <span>Tenant:</span>
            <span className="text-slate-200 font-mono">default</span>
          </div>
        </div>
      </aside>

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col min-w-0 overflow-auto">
        <header className="h-16 border-b border-slate-800/80 bg-slate-950/30 px-8 flex items-center justify-between">
          <div className="text-sm font-medium text-slate-400">
            Environment: <span className="text-emerald-400 font-mono">local-dev</span>
          </div>
          <div className="flex items-center gap-3">
            <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-mono bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
              Data Plane Online :9090
            </span>
          </div>
        </header>

        <main className="p-8 flex-1">
          {children}
        </main>
      </div>
    </div>
  );
}
