import React, { useState, useEffect, useRef } from 'react';
import { 
  Activity, LayoutDashboard, GitMerge, Search, BarChart2, FileText, 
  Settings, TerminalSquare, AlertTriangle, CheckCircle, Copy, 
  Cpu, Network, Server, Play, Zap, ArrowRight, XCircle, AlertCircle,
  ChevronRight, Terminal as TerminalIcon, Sparkles, ShieldAlert,
  ChevronDown, Hexagon, Database, Clock
} from 'lucide-react';

// --- CUSTOM STYLES (Animations & Scrollbars) ---
const customStyles = `
  /* Custom Scrollbar for a premium feel */
  ::-webkit-scrollbar {
    width: 6px;
    height: 6px;
  }
  ::-webkit-scrollbar-track {
    background: transparent;
  }
  ::-webkit-scrollbar-thumb {
    background: #374151;
    border-radius: 3px;
  }
  ::-webkit-scrollbar-thumb:hover {
    background: #4B5563;
  }

  /* Pipeline Data Flow Animation */
  @keyframes packet-flow {
    0% { transform: translateX(0) scale(1); opacity: 1; }
    80% { opacity: 1; }
    100% { transform: translateX(100px) scale(0.5); opacity: 0; }
  }
  .packet {
    animation: packet-flow 1.5s infinite linear;
  }
  .packet:nth-child(2) { animation-delay: 0.5s; }
  .packet:nth-child(3) { animation-delay: 1.0s; }
  
  .glass-panel {
    background: rgba(17, 24, 39, 0.7);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
  }
`;

// --- MOCK DATA ---
const INITIAL_PROBLEMS = [
  { id: 1, type: 'High Cardinality', service: 'payment-service', impact: 'High', severity: 'critical', fix: 'Drop attribute `user.session_id`' },
  { id: 2, type: 'Broken Trace Chain', service: 'auth-service', impact: 'Medium', severity: 'warning', fix: 'Inject trace context in headers' },
  { id: 3, type: 'Coverage Gap', service: 'billing-worker', impact: 'Medium', severity: 'warning', fix: 'Add OTel auto-instrumentation' },
];

const MOCK_AI_YAML = `processors:
  filter/payments:
    metrics:
      exclude:
        match_type: strict
        metric_names:
          - payment.processing.duration
  attributes/cardinality:
    actions:
      - key: user.session_id
        action: delete`;

// --- MAIN APPLICATION COMPONENT ---
export default function App() {
  const [activeTab, setActiveTab] = useState('dashboard');
  const [isTerminalOpen, setIsTerminalOpen] = useState(true);
  const [healthScore, setHealthScore] = useState(94);
  const [signalsPerSec, setSignalsPerSec] = useState(142050);
  const [terminalInput, setTerminalInput] = useState('');
  const [logs, setLogs] = useState([
    { id: 1, time: '10:45:01.000', level: 'INFO', service: 'telemetry-health', msg: 'Pipeline analysis started. 4 nodes active.' },
    { id: 2, time: '10:45:01.050', level: 'INFO', service: 'telemetry-health', msg: 'Listening on port 4317 (gRPC) and 4318 (HTTP).' }
  ]);
  
  const terminalEndRef = useRef(null);

  // Auto-scroll terminal
  useEffect(() => {
    if (terminalEndRef.current) {
      terminalEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, isTerminalOpen]);

  // Terminal submission handler
  const handleTerminalSubmit = (e) => {
    e.preventDefault();
    if (!terminalInput.trim()) return;

    const cmd = terminalInput.trim().toLowerCase();
    
    // Echo command
    const newLogs = [...logs, { id: Date.now(), time: new Date().toISOString().substring(11, 23), level: 'CMD', service: 'user', msg: `> ${cmd}` }];
    setLogs(newLogs);
    setTerminalInput('');

    // Simulate backend response based on command
    setTimeout(() => {
      let responseLogs = [];
      let newScore = healthScore;
      let newSignals = signalsPerSec;

      const timestamp = () => new Date().toISOString().substring(11, 23);

      if (cmd.includes('payment')) {
        responseLogs = [
          { id: Date.now()+1, time: timestamp(), level: 'WARN', service: 'payment-service', msg: 'High cardinality detected on metric: payment.duration' },
          { id: Date.now()+2, time: timestamp(), level: 'ERROR', service: 'payment-service', msg: 'Span dropped: attribute count exceeds limit (4096)' },
          { id: Date.now()+3, time: timestamp(), level: 'INFO', service: 'telemetry-health', msg: 'Updating pipeline quality score...' }
        ];
        newScore = 78; // Drop score
        newSignals += 15000; // Spike in signals
      } else if (cmd.includes('auth')) {
        responseLogs = [
          { id: Date.now()+1, time: timestamp(), level: 'ERROR', service: 'auth-service', msg: 'Missing traceparent header in outbound request' },
          { id: Date.now()+2, time: timestamp(), level: 'WARN', service: 'auth-service', msg: 'Trace chain broken at edge proxy' }
        ];
        newScore = 85;
      } else {
        responseLogs = [
          { id: Date.now()+1, time: timestamp(), level: 'INFO', service: cmd, msg: `Fetching live telemetry streams for ${cmd}...` },
          { id: Date.now()+2, time: timestamp(), level: 'INFO', service: cmd, msg: 'No anomalies detected in the last 60 seconds.' }
        ];
        newScore = Math.min(100, healthScore + 2);
      }

      setLogs(prev => [...prev, ...responseLogs]);
      setHealthScore(newScore);
      setSignalsPerSec(newSignals);

    }, 600);
  };

  return (
    <div className="flex h-screen bg-[#0B0F17] text-gray-300 font-sans overflow-hidden">
      <style>{customStyles}</style>

      {/* --- LEFT SIDEBAR --- */}
      <aside className="w-64 bg-[#111827] border-r border-[#1F2937] flex flex-col z-20">
        <div className="h-16 flex items-center px-6 border-b border-[#1F2937]">
          <Hexagon className="text-orange-500 w-6 h-6 mr-3" />
          <span className="font-semibold text-white tracking-wide">TelemetryHealth</span>
        </div>
        
        <nav className="flex-1 overflow-y-auto py-4 px-3 space-y-1">
          <NavItem icon={<LayoutDashboard />} label="Dashboard" active={activeTab === 'dashboard'} onClick={() => setActiveTab('dashboard')} />
          <NavItem icon={<Activity />} label="Telemetry Health" />
          <NavItem icon={<Network />} label="Live Pipeline" />
          <NavItem icon={<GitMerge />} label="Trace Explorer" />
          
          <div className="pt-4 pb-2 px-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Signals</div>
          <NavItem icon={<BarChart2 />} label="Metrics" />
          <NavItem icon={<FileText />} label="Logs" />
          
          <div className="pt-4 pb-2 px-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Intelligence</div>
          <NavItem icon={<Sparkles />} label="AI Recommendations" />
          <NavItem icon={<ShieldAlert />} label="Remediation Center" />
          <NavItem icon={<Network />} label="Behavior Graph" />
          
          <div className="pt-4 pb-2 px-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Configuration</div>
          <NavItem icon={<Settings />} label="Settings" />
        </nav>

        <div className="p-4 border-t border-[#1F2937]">
          <div className="flex items-center space-x-3 text-sm">
            <div className="w-8 h-8 rounded bg-gray-800 flex items-center justify-center border border-gray-700">
              <span className="text-white font-medium text-xs">JD</span>
            </div>
            <div>
              <div className="text-white font-medium">Platform Eng</div>
              <div className="text-gray-500 text-xs">Production Cluster</div>
            </div>
          </div>
        </div>
      </aside>

      {/* --- MAIN WORKSPACE --- */}
      <main className="flex-1 flex flex-col min-w-0 relative">
        {/* Top Header */}
        <header className="h-16 flex items-center justify-between px-8 border-b border-[#1F2937] bg-[#0B0F17]/80 backdrop-blur-sm z-10">
          <div className="flex items-center space-x-2 text-sm">
            <span className="text-gray-500">Clusters</span>
            <ChevronRight className="w-4 h-4 text-gray-600" />
            <span className="text-gray-500">us-east-1-prod</span>
            <ChevronRight className="w-4 h-4 text-gray-600" />
            <span className="text-white font-medium">Dashboard</span>
          </div>
          
          <div className="flex items-center space-x-4">
            <div className="flex items-center space-x-2 bg-[#111827] border border-[#1F2937] rounded-md px-3 py-1.5 text-sm">
              <Clock className="w-4 h-4 text-gray-500" />
              <span>Last 15 minutes</span>
              <ChevronDown className="w-4 h-4 text-gray-500" />
            </div>
            <button 
              onClick={() => setIsTerminalOpen(!isTerminalOpen)}
              className={`p-2 rounded-md transition-colors ${isTerminalOpen ? 'bg-orange-500/10 text-orange-500 border border-orange-500/20' : 'bg-[#111827] text-gray-400 border border-[#1F2937] hover:text-white'}`}
            >
              <TerminalIcon className="w-5 h-5" />
            </button>
          </div>
        </header>

        {/* Scrollable Dashboard Content */}
        <div className={`flex-1 overflow-y-auto p-8 transition-all duration-300 ${isTerminalOpen ? 'pb-72' : ''}`}>
          <div className="max-w-7xl mx-auto space-y-6">
            
            {/* Top Row: KPIs */}
            <div className="grid grid-cols-5 gap-4">
              {/* Main Score Card */}
              <div className="col-span-1 bg-[#111827] border border-[#1F2937] rounded-lg p-5 flex flex-col justify-between">
                <div className="text-gray-400 text-sm font-medium">Telemetry Health</div>
                <div className="flex items-end space-x-2 mt-2">
                  <span className={`text-5xl font-bold tracking-tight transition-colors duration-500 ${healthScore > 90 ? 'text-green-500' : healthScore > 75 ? 'text-orange-500' : 'text-red-500'}`}>
                    {healthScore}
                  </span>
                  <span className="text-gray-500 mb-1">/ 100</span>
                </div>
                <div className="mt-4 w-full bg-gray-800 h-1.5 rounded-full overflow-hidden">
                  <div 
                    className={`h-full transition-all duration-1000 ${healthScore > 90 ? 'bg-green-500' : healthScore > 75 ? 'bg-orange-500' : 'bg-red-500'}`}
                    style={{ width: `${healthScore}%` }}
                  />
                </div>
              </div>

              {/* Other KPIs */}
              <KPICard title="Pipeline Status" value="Healthy" icon={<CheckCircle className="text-green-500" />} subtitle="All 4 nodes up" />
              <KPICard title="Collector Load" value="42%" icon={<Cpu className="text-gray-400" />} subtitle="Peak: 68%" />
              <KPICard title="Signals / sec" value={(signalsPerSec).toLocaleString()} icon={<Activity className="text-blue-500" />} subtitle="↑ 12% from avg" />
              <KPICard title="Active Alerts" value={healthScore < 90 ? "3" : "0"} icon={<AlertTriangle className={healthScore < 90 ? "text-orange-500" : "text-gray-500"} />} subtitle="Requires attention" />
            </div>

            {/* Second Row: Animated Pipeline Flow */}
            <div className="bg-[#111827] border border-[#1F2937] rounded-lg p-6">
              <div className="flex justify-between items-center mb-6">
                <h2 className="text-white font-semibold flex items-center">
                  <Network className="w-5 h-5 mr-2 text-gray-400" /> Live Pipeline Flow
                </h2>
                <div className="flex items-center space-x-4 text-xs">
                  <span className="flex items-center"><span className="w-2 h-2 rounded-full bg-blue-500 mr-2"></span>Traces</span>
                  <span className="flex items-center"><span className="w-2 h-2 rounded-full bg-green-500 mr-2"></span>Metrics</span>
                  <span className="flex items-center"><span className="w-2 h-2 rounded-full bg-purple-500 mr-2"></span>Logs</span>
                </div>
              </div>
              
              <div className="relative py-8 px-4 flex items-center justify-between">
                {/* Connecting Lines */}
                <div className="absolute top-1/2 left-20 right-20 h-0.5 bg-gray-800 -translate-y-1/2 z-0"></div>
                
                {/* Animated Packets Container */}
                <div className="absolute top-1/2 left-24 right-24 h-0.5 -translate-y-1/2 z-0 overflow-hidden">
                   <div className="relative w-full h-full">
                     <div className="absolute top-[-3px] left-0 w-2 h-2 rounded-full bg-blue-500 packet"></div>
                     <div className="absolute top-[-3px] left-0 w-2 h-2 rounded-full bg-green-500 packet"></div>
                     <div className="absolute top-[-3px] left-0 w-2 h-2 rounded-full bg-purple-500 packet shadow-[0_0_8px_#a855f7]"></div>
                   </div>
                </div>

                {/* Nodes */}
                <PipelineNode icon={<Server />} name="Applications" metric="14k RPS" />
                <PipelineNode icon={<Hexagon />} name="OTel Collector" metric="42% CPU" status={healthScore < 90 ? 'warning' : 'healthy'} />
                <PipelineNode icon={<ShieldAlert />} name="TelemetryHealth" metric="99.9% Quality" active highlight />
                <PipelineNode icon={<Database />} name="SigNoz" metric="1.2 TB/day" />
              </div>
            </div>

            {/* Third Row: Problems & AI Recommendations */}
            <div className="grid grid-cols-3 gap-6">
              
              {/* Problems Detected Table */}
              <div className="col-span-2 bg-[#111827] border border-[#1F2937] rounded-lg flex flex-col">
                <div className="p-5 border-b border-[#1F2937] flex justify-between items-center">
                  <h2 className="text-white font-semibold flex items-center">
                    <AlertCircle className="w-5 h-5 mr-2 text-red-400" /> Problems Detected
                  </h2>
                  <span className="bg-red-500/10 text-red-500 text-xs px-2 py-1 rounded border border-red-500/20">3 Issues</span>
                </div>
                <div className="p-0 overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead className="bg-[#0B0F17] text-gray-500 border-b border-[#1F2937]">
                      <tr>
                        <th className="px-5 py-3 font-medium">Issue Type</th>
                        <th className="px-5 py-3 font-medium">Service / Source</th>
                        <th className="px-5 py-3 font-medium">Severity</th>
                        <th className="px-5 py-3 font-medium">Action</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-[#1F2937]">
                      {INITIAL_PROBLEMS.map(prob => (
                        <tr key={prob.id} className="hover:bg-gray-800/30 transition-colors group cursor-pointer">
                          <td className="px-5 py-4 text-gray-200 font-medium">{prob.type}</td>
                          <td className="px-5 py-4 font-mono text-xs text-gray-400">{prob.service}</td>
                          <td className="px-5 py-4">
                            <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium border ${
                              prob.severity === 'critical' ? 'bg-red-500/10 text-red-400 border-red-500/20' : 'bg-orange-500/10 text-orange-400 border-orange-500/20'
                            }`}>
                              {prob.severity.toUpperCase()}
                            </span>
                          </td>
                          <td className="px-5 py-4 text-orange-500 opacity-0 group-hover:opacity-100 transition-opacity">
                            View Details →
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>

              {/* AI Recommendation Panel */}
              <div className="col-span-1 bg-[#111827] border border-[#1F2937] rounded-lg flex flex-col relative overflow-hidden">
                <div className="absolute top-0 left-0 right-0 h-1 bg-gradient-to-r from-orange-500 to-amber-400"></div>
                <div className="p-5 border-b border-[#1F2937]">
                  <h2 className="text-white font-semibold flex items-center">
                    <Sparkles className="w-5 h-5 mr-2 text-orange-400" /> AI Recommendation
                  </h2>
                  <p className="text-xs text-gray-400 mt-1">For: High Cardinality in payment-service</p>
                </div>
                
                <div className="p-5 flex-1 flex flex-col space-y-4">
                  <div>
                    <div className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">Root Cause</div>
                    <p className="text-sm text-gray-300 leading-relaxed">
                      The attribute <code className="text-orange-400 bg-gray-900 px-1 rounded">user.session_id</code> on metric <code className="text-gray-300 bg-gray-900 px-1 rounded">payment.duration</code> is generating 15,000+ unique timeseries/min, exceeding Collector limits.
                    </p>
                  </div>
                  
                  <div className="flex-1 flex flex-col">
                    <div className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2 flex justify-between items-center">
                      <span>Suggested YAML Fix</span>
                      <button className="text-gray-400 hover:text-white"><Copy className="w-3.5 h-3.5" /></button>
                    </div>
                    <div className="bg-[#0B0F17] border border-[#1F2937] rounded p-3 flex-1 overflow-y-auto">
                      <pre className="font-mono text-xs text-blue-300 leading-relaxed">
                        {MOCK_AI_YAML}
                      </pre>
                    </div>
                  </div>
                  
                  <button className="w-full bg-white text-black font-semibold py-2 rounded flex items-center justify-center hover:bg-gray-200 transition-colors">
                    <Zap className="w-4 h-4 mr-2" />
                    Apply to Collector
                  </button>
                </div>
              </div>

            </div>
          </div>
        </div>

        {/* --- BOTTOM RAW LOG TERMINAL --- */}
        <div 
          className={`absolute bottom-0 left-0 right-0 bg-[#0A0D12] border-t border-[#1F2937] transition-all duration-300 ease-in-out flex flex-col z-30 shadow-[0_-10px_40px_rgba(0,0,0,0.5)]
            ${isTerminalOpen ? 'h-72 translate-y-0' : 'h-72 translate-y-full'}`}
        >
          {/* Terminal Header */}
          <div className="h-10 flex items-center justify-between px-4 bg-[#111827] border-b border-[#1F2937]">
            <div className="flex items-center space-x-3">
              <TerminalIcon className="w-4 h-4 text-orange-500" />
              <span className="text-sm font-medium text-white tracking-wide">Live Pipeline Terminal</span>
              <span className="text-xs text-gray-500 bg-gray-800 px-2 py-0.5 rounded ml-2">Warp-inspired</span>
            </div>
            <button onClick={() => setIsTerminalOpen(false)} className="text-gray-400 hover:text-white">
              <XCircle className="w-5 h-5" />
            </button>
          </div>

          {/* Terminal Output */}
          <div className="flex-1 overflow-y-auto p-4 font-mono text-sm space-y-1.5 scroll-smooth">
            {logs.map((log) => (
              <div key={log.id} className="flex group hover:bg-white/5 px-2 py-0.5 -mx-2 rounded transition-colors">
                <span className="text-gray-500 w-24 shrink-0 select-none">{log.time}</span>
                <span className={`w-14 shrink-0 font-bold select-none
                  ${log.level === 'ERROR' ? 'text-red-400' : 
                    log.level === 'WARN' ? 'text-amber-400' : 
                    log.level === 'CMD' ? 'text-blue-400' : 'text-gray-400'}`}
                >
                  {log.level === 'CMD' ? '' : `[${log.level}]`}
                </span>
                <span className="text-gray-500 w-36 shrink-0 truncate px-2 select-none">[{log.service}]</span>
                <span className={`flex-1 break-all ${
                  log.level === 'ERROR' ? 'text-red-300' : 
                  log.level === 'WARN' ? 'text-amber-300' : 
                  log.level === 'CMD' ? 'text-blue-300 font-semibold' : 'text-gray-300'
                }`}>
                  {log.msg}
                </span>
              </div>
            ))}
            <div ref={terminalEndRef} />
          </div>

          {/* Terminal Input */}
          <div className="p-3 bg-[#0B0F17] border-t border-[#1F2937] flex items-center">
            <span className="text-orange-500 font-bold mr-3 ml-2 font-mono">{'>'}</span>
            <form onSubmit={handleTerminalSubmit} className="flex-1">
              <input
                type="text"
                value={terminalInput}
                onChange={(e) => setTerminalInput(e.target.value)}
                placeholder="Type 'payment-service' or 'auth-service' to query logs..."
                className="w-full bg-transparent text-white font-mono focus:outline-none placeholder-gray-600"
                autoComplete="off"
                spellCheck="false"
              />
            </form>
          </div>
        </div>
      </main>
    </div>
  );
}

// --- SUB-COMPONENTS ---

function NavItem({ icon, label, active, onClick }) {
  return (
    <button 
      onClick={onClick}
      className={`w-full flex items-center space-x-3 px-3 py-2 rounded-md transition-colors text-sm font-medium
        ${active ? 'bg-orange-500/10 text-orange-500' : 'text-gray-400 hover:text-white hover:bg-gray-800/50'}`}
    >
      <div className={`w-5 h-5 ${active ? 'text-orange-500' : 'text-gray-400'}`}>
        {React.cloneElement(icon, { size: 18 })}
      </div>
      <span>{label}</span>
    </button>
  );
}

function KPICard({ title, value, subtitle, icon }) {
  return (
    <div className="col-span-1 bg-[#111827] border border-[#1F2937] rounded-lg p-5 flex flex-col justify-between">
      <div className="flex justify-between items-start">
        <div className="text-gray-400 text-sm font-medium">{title}</div>
        <div>{icon}</div>
      </div>
      <div className="mt-4">
        <div className="text-2xl font-bold text-white">{value}</div>
        <div className="text-xs text-gray-500 mt-1">{subtitle}</div>
      </div>
    </div>
  );
}

function PipelineNode({ icon, name, metric, status = 'healthy', active = false, highlight = false }) {
  const isWarning = status === 'warning';
  
  return (
    <div className={`relative z-10 flex flex-col items-center p-4 rounded-xl border transition-all duration-300 w-40
      ${highlight 
        ? 'bg-orange-500/5 border-orange-500/30 shadow-[0_0_20px_rgba(249,115,22,0.1)]' 
        : 'bg-[#0B0F17] border-[#1F2937]'}
    `}>
      <div className={`w-12 h-12 rounded-lg flex items-center justify-center mb-3
        ${highlight ? 'bg-orange-500 text-white' : 'bg-gray-800 text-gray-300 border border-gray-700'}
        ${isWarning ? 'border-orange-500/50 shadow-[0_0_15px_rgba(249,115,22,0.2)]' : ''}
      `}>
        {React.cloneElement(icon, { size: 24 })}
      </div>
      <div className={`font-semibold text-sm mb-1 ${highlight ? 'text-orange-400' : 'text-gray-200'}`}>
        {name}
      </div>
      <div className="text-xs font-mono text-gray-500 flex items-center">
        {isWarning && <AlertTriangle className="w-3 h-3 text-orange-500 mr-1" />}
        {metric}
      </div>
    </div>
  );
}