

export function Coverage() {
  return (
    <section className="view active">
      <div className="eyebrow">04 &#183; coverage &#183; sampling gap detector &#183; §8.3</div>
      <div className="panel panel-tight">
        <table>
          <thead>
            <tr><th>service</th><th>last seen</th><th>expected baseline</th><th>grace period</th><th>status</th></tr>
          </thead>
          <tbody>
            <tr>
              <td className="mono-cell">inventory-worker</td>
              <td className="mono-cell">14m ago</td>
              <td className="mono-cell">continuous</td>
              <td className="mono-cell" style={{ color: 'var(--red)' }}>breached</td>
              <td><span className="status-chip"><span className="rled a"></span>silent</span></td>
            </tr>
            <tr>
              <td className="mono-cell">fraud-scoring</td>
              <td className="mono-cell">22m ago</td>
              <td className="mono-cell">continuous</td>
              <td className="mono-cell" style={{ color: 'var(--red)' }}>breached</td>
              <td><span className="status-chip"><span className="rled r"></span>down</span></td>
            </tr>
            <tr>
              <td className="mono-cell">notification-svc</td>
              <td className="mono-cell">3s ago</td>
              <td className="mono-cell">continuous</td>
              <td className="mono-cell">&mdash;</td>
              <td><span className="status-chip"><span className="rled p"></span>healthy</span></td>
            </tr>
            <tr>
              <td className="mono-cell">legacy-batch-job</td>
              <td className="mono-cell">2d ago</td>
              <td className="mono-cell">weekly cadence</td>
              <td className="mono-cell">&mdash;</td>
              <td><span className="status-chip"><span className="rled p"></span>healthy</span></td>
            </tr>
          </tbody>
        </table>
      </div>
      <div className="footnote">baseline derived from rolling 30-day emission pattern &#183; grace period default 10m</div>
    </section>
  );
}
