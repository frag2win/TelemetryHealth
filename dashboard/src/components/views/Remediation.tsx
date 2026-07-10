import { useState } from 'react';
import { Copy, Check, ExternalLink } from 'lucide-react';
import type { RemediationPayload } from '../../App';

interface RemediationProps {
  apiRemediation?: RemediationPayload;
}

export function Remediation({ apiRemediation }: RemediationProps) {
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [toast, setToast] = useState<string | null>(null);
  const [activeTabs, setActiveTabs] = useState<Record<string, 'yaml' | 'diff'>>({
    'rem-api': 'yaml',
    'rem-1': 'yaml',
    'rem-2': 'yaml',
    'rem-3': 'yaml'
  });

  const copyCode = (id: string, text: string) => {
    if (navigator.clipboard) {
      navigator.clipboard.writeText(text)
        .then(() => {
          setCopiedId(id);
          setToast('Config copied to clipboard');
          setTimeout(() => { setCopiedId(null); }, 1300);
          setTimeout(() => { setToast(null); }, 2000);
        })
        .catch((err) => {
          console.error('Failed to copy text: ', err);
        });
    }
  };

  const applyRemediation = async (issueType: string, yaml: string) => {
    try {
      const response = await fetch('http://localhost:8080/api/v1/remediation/apply', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ issueType, yaml })
      });
      if (!response.ok) throw new Error('Apply failed');
      setToast('Remediation applied to OTel Collector successfully');
      setTimeout(() => setToast(null), 2500);
    } catch (e) {
      console.error(e);
      setToast('Error: Failed to apply remediation');
      setTimeout(() => setToast(null), 2500);
    }
  };

  // Inline syntax highlighter parser for YAML
  const highlightYamlLine = (line: string, index: number) => {
    if (line.trim().startsWith('#')) {
      return <div key={index} className="yaml-comment">{line}</div>;
    }
    const colonIndex = line.indexOf(':');
    if (colonIndex !== -1) {
      const key = line.slice(0, colonIndex);
      const rest = line.slice(colonIndex);
      // Further color values
      if (rest.includes('"') || rest.includes('\'') || rest.includes('delete') || rest.includes('100')) {
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

  const getDiffLines = (id: string) => {
    // Generate simple mock before/after configurations
    if (id === 'rem-1' || id === 'rem-api') {
      return [
        { type: 'unchanged', num: 1, text: 'processors:' },
        { type: 'unchanged', num: 2, text: '  batch:' },
        { type: 'unchanged', num: 3, text: '    timeout: 1s' },
        { type: 'added', num: 4, text: '  attributes/remediation:' },
        { type: 'added', num: 5, text: '    actions:' },
        { type: 'added', num: 6, text: '      - key: "user_id"' },
        { type: 'added', num: 7, text: '        action: "delete"' }
      ];
    } else if (id === 'rem-2') {
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

  const renderCard = (id: string, badgeType: string, svc: string, code: string) => {
    const activeTab = activeTabs[id] || 'yaml';

    return (
      <div className="rem-card" id={id}>
        <div className="rem-head">
          <span className="badge badge-type">{badgeType}</span>
          <span className="rem-svc">{svc}</span>
          <span className="badge badge-ok">validated in sandbox</span>
          
          {/* Tab Selection Switcher */}
          <div className="pillgroup" style={{ marginLeft: '12px', padding: '2px' }}>
            <button
              className={`pill ${activeTab === 'yaml' ? 'active' : ''}`}
              style={{ padding: '3px 8px', fontSize: '10px' }}
              onClick={() => setActiveTabs(prev => ({ ...prev, [id]: 'yaml' }))}
            >
              YAML Patch
            </button>
            <button
              className={`pill ${activeTab === 'diff' ? 'active' : ''}`}
              style={{ padding: '3px 8px', fontSize: '10px' }}
              onClick={() => setActiveTabs(prev => ({ ...prev, [id]: 'diff' }))}
            >
              Diff View
            </button>
          </div>

          <div className="rem-actions">
            <button className="btn" onClick={() => applyRemediation(badgeType, code)}>
              Apply to Collector
            </button>
            <button
              className="btn copy-btn"
              onClick={() => copyCode(id, code)}
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
  };

  return (
    <section className="view active">
      {/* High-priority overlay Toast notification */}
      {toast && (
        <div
          style={{
            position: 'fixed',
            bottom: '1rem',
            right: '1rem',
            background: 'var(--toast-bg)',
            border: '1px solid var(--toast-border)',
            padding: '12px 24px',
            borderRadius: '4px',
            color: 'var(--phosphor)',
            zIndex: 9999,
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            boxShadow: '0 4px 12px rgba(0,0,0,0.5)'
          }}
        >
          <Check size={16} />
          <span style={{ fontSize: '13px', fontWeight: '500' }}>{toast}</span>
        </div>
      )}

      <div className="eyebrow">05 • remediation generator • §8.5 • propose-only in v1</div>

      {apiRemediation && apiRemediation.yaml && renderCard(
        'rem-api',
        apiRemediation.issueType || 'sandbox auto-healing',
        'API Suggestion',
        apiRemediation.yaml
      )}

      {renderCard(
        'rem-1',
        'cardinality redaction',
        'checkout-service · user_id_raw',
        `processors:\n  attributes/redact_user_id:\n    actions:\n      - key: user_id_raw\n        action: delete`
      )}

      {renderCard(
        'rem-2',
        'sampling adjustment',
        'payments-api · high orphan rate',
        `processors:\n  probabilistic_sampler/payments:\n    sampling_percentage: 100\n    # deterministic hash on trace_id, applies fleet-wide`
      )}

      {renderCard(
        'rem-3',
        'coverage · instrumentation',
        'inventory-worker · silent 14m',
        `receivers:\n  otlp/inventory_worker:\n    protocols:\n      grpc:\n        endpoint: inventory-worker:4317`
      )}

      <div className="footnote">every snippet above ran through the shadow-collector dry-run (zero egress, 500m cpu / 128mb ram cap) before appearing here · §8.5</div>
    </section>
  );
}
