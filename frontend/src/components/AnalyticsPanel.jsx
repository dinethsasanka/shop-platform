export default function AnalyticsPanel({ stats }) {
  const { data, error } = stats

  return (
    <section className="panel panel-analytics" aria-labelledby="analytics-heading">
      <h2 id="analytics-heading">Analytics</h2>
      {error && <p className="result error">Totals not loaded: {error}</p>}
      {data === null ? (
        !error && <p className="muted">Loading totals…</p>
      ) : (
        <div className={error ? 'analytics stale' : 'analytics'}>
          <table className="totals">
            <tbody>
              <TotalRow label="Orders placed" value={data.total_orders} />
              <TotalRow label="Units ordered" value={data.units_ordered} />
              <TotalRow label="Confirmed" value={data.confirmed_orders} />
              <TotalRow label="Out of stock" value={data.out_of_stock_orders} />
            </tbody>
          </table>

          <div>
            <h3>Orders per product</h3>
            {Object.keys(data.orders_by_product).length === 0 ? (
              <p className="muted">No orders yet.</p>
            ) : (
              <table className="per-product">
                <thead>
                  <tr>
                    <th scope="col">Product</th>
                    <th scope="col">Orders</th>
                  </tr>
                </thead>
                <tbody>
                  {Object.entries(data.orders_by_product).map(([productId, count]) => (
                    <tr key={productId}>
                      <td>{productId}</td>
                      <td>{count}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}
    </section>
  )
}

// A small component used only inside this file: one row of the totals table.
function TotalRow({ label, value }) {
  return (
    <tr>
      <th scope="row">{label}</th>
      <td>{value}</td>
    </tr>
  )
}
