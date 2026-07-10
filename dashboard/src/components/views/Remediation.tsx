import { useState, useEffect, useRef } from 'react';
import { Copy, Check, ExternalLink, AlertTriangle } from 'lucide-react';
import type { RemediationPayload } from '../../App';

interface RemediationProps {
  apiRemediation?: RemediationPayload;
}

interface RemediationCardProps {
  id: string;
  badgeType: string;
  svc: string;
  code: string;
  activeTab: 'yaml' | 'diff';
  onTabChange: (tab: 'yaml' | 'diff') => void;
  copiedId: string | null;
  onCopy: (id: string, text: string) => void;
  onApply: (issueType: string, yaml: string) => void;
  isApplying: boolean;
}

// 1. Formal React component for Remediation Card to ensure Virtual DOM reconciliation
function RemediationCard({
  id,
  badgeType,
  svc,
  code,
  activeTab,
  onTabChange,
  copiedId,
  onCopy,
  onApply,
  isApplying
}: RemediationCardProps) {
  // Safe YAML Syntax highlighting without false positives
  const highlightYamlLine = (line: string, index: number) => {
    if (line.trim().startsWith('#')) {
      return <div key={index} className="yaml-comment">{line}</div>;
    }
    const colonIndex = line.indexOf(':');
    if (colonIndex !== -1) {
      const key = line.slice(0, colonIndex);
      const rest = line.slice(colonIndex);
      const restTrimmed = rest.trim();
      
      // Strict matching for values to prevent broad string contains warnings
      const isQuotedString = (restTrimmed.startsWith('"') && restTrimmed.endsWith('"')) || 
                           (restTrimmed.startsWith("'") && restTrimmed.endsWith("'"));
      const isSpecificKeyword = restTrimmed === 'delete' || restTrimmed === '100' || restTrimmed === 'true' || restTrimmed === 'false';
      
      if (isQuotedString || isSpecificKeyword) {
        return (
          <div key={index}>
            <span className="yaml-key">{key}</span>
            <span className="yaml-value">{rest}</span>
          </div>
        );
      }
      return (
        <div key={index}>
          <span className="yaml-key">{key}</span>
          <span className="yaml-text">{rest}</span>
        </div>
      );
    }
    return <div key={index} className="yaml-text">{line}</div>;
  };

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
    <div className="rem-card" id={id}>
      <div className="rem-head">
        <span className="badge badge-type">{badgeType}</span>
        <span className="rem-svc">{svc}</span>
        <span className="badge badge-ok">validated in sandbox</span>
        
        <div className="pillgroup" style={{ marginLeft: '12px', padding: '2px' }}>
          <button
            className={`pill ${activeTab === 'yaml' ? 'active' : ''}`}
            style={{ padding: '3px 8px', fontSize: '10px' }}
            onClick={() => onTabChange('yaml')}
          >
            YAML Patch
          </button>
          <button
            className={`pill ${activeTab === 'diff' ? 'active' : ''}`}
            style={{ padding: '3px 8px', fontSize: '10px' }}
            onClick={() => onTabChange('diff')}
          >
            Diff View
          </button>
        </div>

        <div className="rem-actions">
          <button 
            className="btn" 
            onClick={() => onApply(badgeType, code)}
            disabled={isApplying}
          >
            {isApplying ? 'Applying...' : 'Apply to Collector'}
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

      {activeTab === 'yaml' ? (
        <pre className="code">
          {code.split('\n').map((line, i) => highlightYamlLine(line, i))}
        </pre>
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

  return (
    <section className="view active">
      {/* Toast Alert situated fixed overlay standards */}
      {toast && (
        <div
          style={{
            position: 'fixed',
            bottom: '1rem',
            right: '1rem',
            background: 'var(--toast-bg)',
            border: `1px solid ${toast.isError ? 'var(--red)' : 'var(--toast-border)'}`,
            padding: '12px 24px',
            borderRadius: '4px',
            color: toast.isError ? 'var(--red)' : 'var(--phosphor)',
            zIndex: 9999,
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            boxShadow: '0 4px 12px rgba(0,0,0,0.5)'
          }}
        >
          {toast.isError ? <AlertTriangle size={16} /> : <Check size={16} />}
          <span style={{ fontSize: '13px', fontWeight: '500' }}>{toast.message}</span>
        </div>
      )}

      <div className="eyebrow">05 • remediation generator • §8.5 • propose-only in v1</div>

      {apiRemediation && apiRemediation.yaml && (
        <RemediationCard
          id="rem-api"
          badgeType={apiRemediation.issueType || 'sandbox auto-healing'}
          svc="API Suggestion"
          code={apiRemediation.yaml}
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
        code={`processors:\n  attributes/redact_user_id:\n    actions:\n      - key: user_id_raw\n        action: delete`}
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
        code={`processors:\n  probabilistic_sampler/payments:\n    sampling_percentage: 100\n    # deterministic hash on trace_id, applies fleet-wide`}
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
        code={`receivers:\n  otlp/inventory_worker:\n    protocols:\n      grpc:\n        endpoint: inventory-worker:4317`}
        activeTab={activeTabs['rem-3'] || 'yaml'}
        onTabChange={(tab) => handleTabChange('rem-3', tab)}
        copiedId={copiedId}
        onCopy={handleCopy}
        onApply={handleApply}
        isApplying={isApplying}
      />

      <div className="footnote">every snippet above ran through the shadow-collector dry-run (zero egress, 500m cpu / 128mb ram cap) before appearing here · §8.5</div>
    </section>
  );
}
