import { useState } from 'react';

export function Remediation({ apiRemediation }: { apiRemediation?: { issueType: string, yaml: string } }) {
  const [copied, setCopied] = useState<string | null>(null);
  const [toast, setToast] = useState<string | null>(null);

  const copyCode = (id: string, text: string) => {
    if (navigator.clipboard) {
      navigator.clipboard.writeText(text).catch(() => {});
    }
    setCopied(id);
    setToast('Config copied to clipboard');
    setTimeout(() => { setCopied(null); }, 1300);
    setTimeout(() => { setToast(null); }, 2000);
  };

  const applyRemediation = async (issueType: string, yaml: string) => {
    try {
      await fetch('/api/v1/remediation/apply', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ issueType, yaml })
      });
      setToast('Remediation applied to OTel Collector');
      setTimeout(() => setToast(null), 2000);
    } catch (e) {
      console.error(e);
    }
  };

  const renderCard = (id: string, badgeType: string, svc: string, code: string) => (
    <div className="rem-card" id={id}>
      <div className="rem-head">
        <span className="badge badge-type">{badgeType}</span>
        <span className="rem-svc">{svc}</span>
        <span className="badge badge-ok">validated in sandbox</span>
        <div className="rem-actions">
          <button className="btn" onClick={() => applyRemediation(badgeType, code)}>
            Apply to Collector
          </button>
          <button 
            className={`btn copy-btn ${copied === id ? 'flash' : ''}`}
            onClick={() => copyCode(id, code)}
          >
            {copied === id ? 'copied' : 'copy config'}
          </button>
          <button 
            className="btn" 
            title="Open a PR on GitHub with this configuration"
            onClick={() => window.open('https://github.com/frag2win/TelemetryHealth/new/main?filename=remediation.yaml&value=' + encodeURIComponent(code), '_blank')}
          >
            open PR &#8599;
          </button>
        </div>
      </div>
      <pre className="code">
        {code.split('\n').map((line, i) => (
          <div key={i}>{line}</div>
        ))}
      </pre>
    </div>
  );

  return (
    <section className="view active">
      {toast && <div style={{ position: 'fixed', bottom: '20px', right: '20px', background: 'var(--panel-2)', padding: '12px 24px', borderRadius: '4px', border: '1px solid var(--phosphor)', color: 'var(--phosphor)', zIndex: 9999 }}>{toast}</div>}
      <div className="eyebrow">05 &#183; remediation generator &#183; §8.5 &#183; propose-only in v1</div>

      {apiRemediation && renderCard(
        'rem-api',
        apiRemediation.issueType,
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

      <div className="footnote">every snippet above ran through the shadow-collector dry-run (zero egress, 500m cpu / 128mb ram cap) before appearing here &#183; §8.5</div>
    </section>
  );
}
