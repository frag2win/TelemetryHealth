

export function Cardinality() {
  return (
    <section className="view active">
      <div className="eyebrow">02 &#183; cardinality detector &#183; §8.1</div>

      <div className="progress-wrap">
        <span className="progress-label">key-space</span>
        <div className="progress-track">
          <div className="progress-fill" style={{ width: '62%' }}></div>
          <div className="progress-cap" style={{ left: '100%' }}></div>
        </div>
        <span className="progress-label">62 / 100 tracked keys</span>
      </div>

      <div className="panel panel-tight">
        <table>
          <thead>
            <tr><th>service</th><th>attribute key</th><th>unique (hll est.)</th><th>trend</th><th>status</th><th></th></tr>
          </thead>
          <tbody>
            <tr>
              <td className="mono-cell">checkout-service</td>
              <td className="mono-cell">user_id_raw</td>
              <td className="mono-cell">~14,382</td>
              <td><svg width="60" height="20"><polyline points="0,16 15,14 30,10 45,6 60,3" fill="none" stroke="#E5484D" strokeWidth="1.5"/></svg></td>
              <td><span className="status-chip"><span className="rled r"></span>breach</span></td>
              <td><button className="btn">redact</button></td>
            </tr>
            <tr>
              <td className="mono-cell">payments-api</td>
              <td className="mono-cell">raw_url</td>
              <td className="mono-cell">~8,120</td>
              <td><svg width="60" height="20"><polyline points="0,15 15,13 30,12 45,8 60,4" fill="none" stroke="#E5484D" strokeWidth="1.5"/></svg></td>
              <td><span className="status-chip"><span className="rled r"></span>breach</span></td>
              <td><button className="btn">redact</button></td>
            </tr>
            <tr>
              <td className="mono-cell">inventory-worker</td>
              <td className="mono-cell">session_token</td>
              <td className="mono-cell">~640</td>
              <td><svg width="60" height="20"><polyline points="0,10 15,11 30,9 45,10 60,10" fill="none" stroke="#F5A623" strokeWidth="1.5"/></svg></td>
              <td><span className="status-chip"><span className="rled a"></span>watch</span></td>
              <td><button className="btn">redact</button></td>
            </tr>
            <tr>
              <td className="mono-cell">auth-service</td>
              <td className="mono-cell">request_id</td>
              <td className="mono-cell">~92</td>
              <td><svg width="60" height="20"><polyline points="0,10 15,10 30,11 45,10 60,10" fill="none" stroke="#5CE1A5" strokeWidth="1.5"/></svg></td>
              <td><span className="status-chip"><span className="rled p"></span>normal</span></td>
              <td><button className="btn btn-ghost btn-disabled">redact</button></td>
            </tr>
          </tbody>
        </table>
      </div>
      <div className="footnote">key-space anomaly detected on checkout-service — dynamic key pattern user_id_1042: active. tracking capped, truncation fallback active.</div>
    </section>
  );
}
