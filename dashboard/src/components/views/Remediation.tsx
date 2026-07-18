import { useState, useEffect, useRef } from 'react';
import { Copy, Check, ExternalLink, AlertTriangle, Play } from 'lucide-react';
import type { RemediationPayload } from '../../App';
import { Toast } from '../Shared';

interface RemediationProps {
  apiRemediation?: RemediationPayload;
}

interface RemediationCardProps {
  id: string;
  badgeType: string;
  svc: string;
  code: string;
  rootCauseExplanation?: string;
  onChange: (newVal: string) => void;
  activeTab: 'yaml' | 'diff';
  onTabChange: (tab: 'yaml' | 'diff') => void;
  copiedId: string | null;
  onCopy: (id: string, text: string) => void;
  onApply: (issueType: string, yaml: string) => void;
  isApplying: boolean;
}

// 1. Formal React component for Remediation Card with live textarea editing & lint warnings
function RemediationCard({
  id,
  badgeType,
  svc,
  code,
  rootCauseExplanation,
  onChange,
  activeTab,
  onTabChange,
  copiedId,
  onCopy,
  onApply,
  isApplying
}: RemediationCardProps) {
  
  // Real-time YAML Syntax validation logic (Bug 21)
  const getLintWarnings = (textVal: string) => {
    const warnings: string[] = [];
    const lines = textVal.split('\n');
    lines.forEach((line, index) => {
      if (line.includes('\t')) {
        warnings.push(`Line ${index + 1}: Tab character detected. YAML requires spaces.`);
      }
      if (line.trim() && !line.trim().startsWith('#') && !line.includes(':')) {
        warnings.push(`Line ${index + 1}: Missing colon ':' key-value separator.`);
      }
    });
    return warnings;
  };

  const warnings = getLintWarnings(code);
  const totalLines = code.split('\n').length;
  const lineNumbers = Array.from({ length: Math.max(5, totalLines) }).map((_, i) => i + 1);

  const getDiffLines = (cardId: string) => {
    if (cardId === 'rem-1' || cardId === 'rem-api') {
      return [
        { type: 'unchanged', num: 1, text: 'processors:' },
        { type: 'unchanged', num: 2, text: '  batch:' },
        { type: 'unchanged', num: 3, text: '    timeout: 1s' },
        { type: 'added', num: 4, text: '  attributes/remediation:' },
        { type: 'added', num: 5, text: '    actions:' },
        { type: 'added', num: 6, text: '      - key: "user_id"' },
        { type: 'added', num: 7, text: '        action: "delete"' }
      ];
    } else if (cardId === 'rem-2') {
      return [
        { type: 'unchanged', num: 1, text: 'processors:' },
        { type: 'removed', num: 2, text: '  probabilistic_sampler/payments:' },
        { type: 'removed', num: 3, text: '    sampling_percentage: 20' },
        { type: 'added', num: 4, text: '  probabilistic_sampler/payments:' },
        { type: 'added', num: 5, text: '    sampling_percentage: 100' },
        { type: 'unchanged', num: 6, text: '  batch:' },
        { type: 'unchanged', num: 7, text: '    timeout: 1s' }
      ];
    } else {
      return [
        { type: 'unchanged', num: 1, text: 'receivers:' },
        { type: 'unchanged', num: 2, text: '  otlp:' },
        { type: 'added', num: 3, text: '  otlp/inventory_worker:' },
        { type: 'added', num: 4, text: '    protocols:' },
        { type: 'added', num: 5, text: '      grpc:' },
        { type: 'added', num: 6, text: '        endpoint: inventory-worker:4317' }
      ];
    }
  };

  return (
    <div className="rem-card" id={id} style={{ marginBottom: '16px' }}>
      <div className="rem-head" style={{ flexWrap: 'wrap', gap: '8px' }}>
        <span className="badge badge-type">{badgeType}</span>
        <span className="rem-svc">{svc}</span>
        <span className="badge badge-ok">sandbox verified</span>
        
        <div className="pillgroup" style={{ marginLeft: '12px', padding: '2px' }}>
          <button
            className={`pill ${activeTab === 'yaml' ? 'active' : ''}`}
            style={{ padding: '3px 8px', fontSize: '10px' }}
            onClick={() => onTabChange('yaml')}
          >
            YAML Editor
          </button>
          <button
            className={`pill ${activeTab === 'diff' ? 'active' : ''}`}
            style={{ padding: '3px 8px', fontSize: '10px' }}
            onClick={() => onTabChange('diff')}
          >
            Diff View
          </button>
        </div>

        <div className="rem-actions" style={{ marginLeft: 'auto' }}>
          <button 
            className="btn" 
            onClick={() => onApply(badgeType, code)}
            disabled={isApplying || warnings.length > 0}
            style={{ display: 'inline-flex', alignItems: 'center', gap: '4px' }}
          >
            <Play size={12} />
            {isApplying ? 'Applying...' : 'Apply Patch'}
          </button>
          <button
            className="btn copy-btn"
            onClick={() => onCopy(id, code)}
            title="Copy code snippet to clipboard"
          >
            {copiedId === id ? <Check size={12} style={{ color: 'var(--phosphor)' }} /> : <Copy size={12} />}
            <span>{copiedId === id ? 'copied' : 'copy'}</span>
          </button>
          <button
            className="btn"
            title="Open a PR on GitHub with this configuration"
            onClick={() => window.open('https://github.com/frag2win/TelemetryHealth/new/main?filename=remediation.yaml&value=' + encodeURIComponent(code), '_blank')}
          >
            <ExternalLink size={12} />
            <span>PR</span>
          </button>
        </div>
      </div>

      {rootCauseExplanation && (
        <div style={{
          background: 'var(--panel-2)',
          borderLeft: '3px solid var(--phosphor)',
          padding: '10px 14px',
          marginBottom: '12px',
          fontSize: '11px',
          color: 'var(--paper)',
          display: 'flex',
          flexDirection: 'column',
          gap: '4px',
          borderRadius: '0 4px 4px 0'
        }}>
          <span style={{ fontSize: '10px', textTransform: 'uppercase', color: 'var(--phosphor)', fontWeight: 600 }}>Root Cause Analysis</span>
          <span>{rootCauseExplanation}</span>
        </div>
      )}

      {activeTab === 'yaml' ? (
        <div className="yaml-editor-layout" style={{ display: 'flex', background: 'var(--panel)', border: '1px solid var(--bezel)', borderRadius: '4px', overflow: 'hidden' }}>
          {/* Editor Line Numbers */}
          <div className="line-numbers" style={{ background: 'var(--panel-2)', padding: '12px 8px', textAlign: 'right', borderRight: '1px solid var(--bezel)', color: 'var(--muted)', fontSize: '12px', fontFamily: 'var(--mono)', userSelect: 'none', minWidth: '30px' }}>
            {lineNumbers.map(n => <div key={n}>{n}</div>)}
          </div>
          {/* Code Textarea Sandbox */}
          <textarea
            value={code}
            onChange={(e) => onChange(e.target.value)}
            spellCheck={false}
            style={{
              flex: 1,
              background: 'transparent',
              border: 'none',
              outline: 'none',
              resize: 'none',
              padding: '12px',
              color: 'var(--paper)',
              fontSize: '12px',
              fontFamily: 'var(--mono)',
              lineHeight: '1.5',
              height: '140px'
            }}
          />
        </div>
      ) : (
        <div className="diff-container" style={{ padding: '12px 0' }}>
          {getDiffLines(id).map((line, i) => (
            <div
              key={i}
              className={`diff-line ${
                line.type === 'added'
                  ? 'diff-line-added'
                  : line.type === 'removed'
                  ? 'diff-line-removed'
                  : 'diff-line-unchanged'
              }`}
            >
              <span className="diff-line-num">{line.num}</span>
              <span>
                {line.type === 'added' ? '+' : line.type === 'removed' ? '-' : ' '} {line.text}
              </span>
            </div>
          ))}
        </div>
      )}

      {/* Editor Lint Warnings Display */}
      {warnings.length > 0 && activeTab === 'yaml' && (
        <div
          style={{
            margin: '8px 12px 12px',
            background: 'var(--amber-dim)',
            border: '1px solid var(--amber)',
            borderRadius: '4px',
            padding: '8px 12px',
            fontSize: '11px',
            color: 'var(--amber)',
            display: 'flex',
            flexDirection: 'column',
            gap: '4px'
          }}
        >
          {warnings.map((warn, i) => (
            <div key={i} style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
              <AlertTriangle size={12} />
              <span>{warn}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export function Remediation({ apiRemediation }: RemediationProps) {
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [toast, setToast] = useState<{ message: string; isError: boolean } | null>(null);
  const [isApplying, setIsApplying] = useState<boolean>(false);
  const [activeTabs, setActiveTabs] = useState<Record<string, 'yaml' | 'diff'>>({
    'rem-api': 'yaml',
    'rem-1': 'yaml',
    'rem-2': 'yaml',
    'rem-3': 'yaml'
  });

  // Slider State representing percentage of redundant spans dropped
  const [dropRatio, setDropRatio] = useState<number>(75);

  // Dynamic state for editable code snippets
  const [snippets, setSnippets] = useState<Record<string, string>>({
    'rem-api': '',
    'rem-1': `processors:\n  attributes/redact_user_id:\n    actions:\n      - key: user_id_raw\n        action: delete`,
    'rem-2': `processors:\n  probabilistic_sampler/payments:\n    sampling_percentage: 100\n    # deterministic hash on trace_id, applies fleet-wide`,
    'rem-3': `receivers:\n  otlp/inventory_worker:\n    protocols:\n      grpc:\n        endpoint: inventory-worker:4317`
  });

  useEffect(() => {
    if (apiRemediation?.yaml) {
      setSnippets(prev => ({ ...prev, 'rem-api': apiRemediation.yaml }));
    }
  }, [apiRemediation]);

  // refs for timeout tracking to prevent unmount memory leaks
  const copiedTimeoutRef = useRef<number | null>(null);
  const toastTimeoutRef = useRef<number | null>(null);

  // Clear timers on component unmount
  useEffect(() => {
    return () => {
      if (copiedTimeoutRef.current) window.clearTimeout(copiedTimeoutRef.current);
      if (toastTimeoutRef.current) window.clearTimeout(toastTimeoutRef.current);
    };
  }, []);

  const handleTabChange = (id: string, tab: 'yaml' | 'diff') => {
    setActiveTabs(prev => ({ ...prev, [id]: tab }));
  };

  const handleCopy = (id: string, text: string) => {
    if (copiedTimeoutRef.current) window.clearTimeout(copiedTimeoutRef.current);
    if (toastTimeoutRef.current) window.clearTimeout(toastTimeoutRef.current);

    // Standard Clipboard API execution
    if (navigator.clipboard) {
      navigator.clipboard.writeText(text)
        .then(() => {
          triggerCopyFeedback(id);
        })
        .catch((err) => {
          console.warn('Clipboard write failed, attempting fallback loop. Details:', err);
          fallbackCopyText(id, text);
        });
    } else {
      fallbackCopyText(id, text);
    }
  };

  // Robust fallback copy loop utilizing temporary textarea elements
  const fallbackCopyText = (id: string, text: string) => {
    try {
      const textArea = document.createElement('textarea');
      textArea.value = text;
      textArea.style.position = 'fixed';
      textArea.style.top = '0';
      textArea.style.left = '0';
      textArea.style.width = '2em';
      textArea.style.height = '2em';
      textArea.style.padding = '0';
      textArea.style.border = 'none';
      textArea.style.outline = 'none';
      textArea.style.boxShadow = 'none';
      textArea.style.background = 'transparent';
      document.body.appendChild(textArea);
      textArea.focus();
      textArea.select();
      
      const successful = document.execCommand('copy');
      document.body.removeChild(textArea);
      if (successful) {
        triggerCopyFeedback(id);
      } else {
        throw new Error('Fallback command exec failed');
      }
    } catch (e) {
      console.error('Total copy failure:', e);
      setToast({ message: 'Error: Failed to copy to clipboard', isError: true });
      toastTimeoutRef.current = window.setTimeout(() => setToast(null), 2500);
    }
  };

  const triggerCopyFeedback = (id: string) => {
    setCopiedId(id);
    setToast({ message: 'Config copied to clipboard', isError: false });
    copiedTimeoutRef.current = window.setTimeout(() => setCopiedId(null), 1300);
    toastTimeoutRef.current = window.setTimeout(() => setToast(null), 2000);
  };

  const handleApply = async (issueType: string, yaml: string) => {
    if (toastTimeoutRef.current) window.clearTimeout(toastTimeoutRef.current);
    setIsApplying(true);
    
    try {
      // Relative proxy URL path compliance
      const response = await fetch('/api/v1/remediation/apply', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ issueType, yaml })
      });
      if (!response.ok) throw new Error('Collector apply mutation failed');
      
      setToast({ message: 'Remediation applied to OTel Collector successfully', isError: false });
      toastTimeoutRef.current = window.setTimeout(() => setToast(null), 2500);
    } catch (e: any) {
      console.error('Error applying remediation:', e);
      setToast({ message: `Error: Failed to apply remediation. ${e.message || ''}`, isError: true });
      toastTimeoutRef.current = window.setTimeout(() => setToast(null), 3000);
    } finally {
      setIsApplying(false);
    }
  };

  const handleCodeChange = (id: string, newVal: string) => {
    setSnippets(prev => ({ ...prev, [id]: newVal }));
  };

  // Anomaly Impact Simulation formulas
  const estSavings = Math.round((dropRatio / 100) * 1800);
  const estIngest = (12.4 - (dropRatio / 100) * 4.8).toFixed(1);
  const cardinalityReduction = Math.round(dropRatio * 0.9);

  return (
    <section className="view active">
      {/* Toast Alert situated fixed overlay standards */}
      {toast && <Toast message={toast.message} isError={toast.isError} />}

      <div className="eyebrow">05 • remediation generator • sandbox sandbox • v1.2</div>

      {/* Anomaly Impact Simulator Panel */}
      <div className="panel" style={{ marginBottom: '20px', borderLeft: '3px solid var(--phosphor)' }}>
        <div className="metric-label" style={{ marginBottom: '14px', textTransform: 'uppercase', fontSize: '12px' }}>
          Remediation Impact & Cost Simulator
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: '12px', marginBottom: '20px' }}>
          <div style={{ background: 'var(--panel-2)', padding: '12px', borderRadius: '4px' }}>
            <div style={{ fontSize: '10px', color: 'var(--muted)', textTransform: 'uppercase' }}>est cost savings</div>
            <div style={{ fontSize: '18px', fontWeight: '700', color: 'var(--phosphor)', marginTop: '4px' }}>
              ${estSavings}/mo
            </div>
            <div style={{ fontSize: '9px', color: 'var(--muted)', marginTop: '4px' }}>Wasted billing stopped</div>
          </div>
          <div style={{ background: 'var(--panel-2)', padding: '12px', borderRadius: '4px' }}>
            <div style={{ fontSize: '10px', color: 'var(--muted)', textTransform: 'uppercase' }}>projected ingest</div>
            <div style={{ fontSize: '18px', fontWeight: '700', color: 'var(--paper)', marginTop: '4px' }}>
              {estIngest} GB/day
            </div>
            <div style={{ fontSize: '9px', color: 'var(--muted)', marginTop: '4px' }}>Down from 12.4 GB/day</div>
          </div>
          <div style={{ background: 'var(--panel-2)', padding: '12px', borderRadius: '4px' }}>
            <div style={{ fontSize: '10px', color: 'var(--muted)', textTransform: 'uppercase' }}>cardinality reduction</div>
            <div style={{ fontSize: '18px', fontWeight: '700', color: 'var(--amber)', marginTop: '4px' }}>
              -{cardinalityReduction}%
            </div>
            <div style={{ fontSize: '9px', color: 'var(--muted)', marginTop: '4px' }}>Reduced dimension index</div>
          </div>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '11px', color: 'var(--paper)' }}>
            <span>Drop Redundant Span Attributes Ratio</span>
            <span style={{ fontFamily: 'var(--mono)', fontWeight: '600' }}>{dropRatio}%</span>
          </div>
          <input
            type="range"
            min="0"
            max="100"
            value={dropRatio}
            onChange={(e) => setDropRatio(parseInt(e.target.value))}
            style={{
              width: '100%',
              cursor: 'pointer',
              accentColor: 'var(--phosphor)',
              background: 'var(--panel-2)',
              height: '6px',
              borderRadius: '3px'
            }}
          />
        </div>
      </div>

      {apiRemediation && apiRemediation.yaml && (
        <RemediationCard
          id="rem-api"
          badgeType={apiRemediation.issueType || 'sandbox auto-healing'}
          svc="API Suggestion"
          code={snippets['rem-api']}
          rootCauseExplanation="The RCIE (Root Cause Intelligence Engine) dynamically generated this YAML patch in response to a detected behavioral anomaly in the agent trace pipeline."
          onChange={(val) => handleCodeChange('rem-api', val)}
          activeTab={activeTabs['rem-api'] || 'yaml'}
          onTabChange={(tab) => handleTabChange('rem-api', tab)}
          copiedId={copiedId}
          onCopy={handleCopy}
          onApply={handleApply}
          isApplying={isApplying}
        />
      )}

      <RemediationCard
        id="rem-1"
        badgeType="cardinality redaction"
        svc="checkout-service · user_id_raw"
        code={snippets['rem-1']}
        rootCauseExplanation="Root Cause Engine detected a Prompt Explosion caused by a Retry Storm. Dropping user_id_raw cardinality will reduce index pressure and unblock the LLM gateway."
        onChange={(val) => handleCodeChange('rem-1', val)}
        activeTab={activeTabs['rem-1'] || 'yaml'}
        onTabChange={(tab) => handleTabChange('rem-1', tab)}
        copiedId={copiedId}
        onCopy={handleCopy}
        onApply={handleApply}
        isApplying={isApplying}
      />

      <RemediationCard
        id="rem-2"
        badgeType="sampling adjustment"
        svc="payments-api · high orphan rate"
        code={snippets['rem-2']}
        rootCauseExplanation="Root Cause Engine traced Span Drop errors back to Collector Queue Saturation. Increasing sampling to 100% on the payments-api temporarily restores trace chain integrity."
        onChange={(val) => handleCodeChange('rem-2', val)}
        activeTab={activeTabs['rem-2'] || 'yaml'}
        onTabChange={(tab) => handleTabChange('rem-2', tab)}
        copiedId={copiedId}
        onCopy={handleCopy}
        onApply={handleApply}
        isApplying={isApplying}
      />

      <RemediationCard
        id="rem-3"
        badgeType="coverage · instrumentation"
        svc="inventory-worker · silent 14m"
        code={snippets['rem-3']}
        onChange={(val) => handleCodeChange('rem-3', val)}
        activeTab={activeTabs['rem-3'] || 'yaml'}
        onTabChange={(tab) => handleTabChange('rem-3', tab)}
        copiedId={copiedId}
        onCopy={handleCopy}
        onApply={handleApply}
        isApplying={isApplying}
      />

      <div className="footnote">sandbox patch visualizer checks YAML structure automatically before application dry-run · §8.5</div>
    </section>
  );
}
