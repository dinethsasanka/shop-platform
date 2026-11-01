export default function InventoryPanel({ stock }) {
  const { data, error } = stock

  return (
    <section className="panel panel-inventory" aria-labelledby="inventory-heading">
      <h2 id="inventory-heading">Inventory</h2>
      {error && <p className="result error">Stock not loaded: {error}</p>}
      {data === null ? (
        !error && <p className="muted">Loading stock…</p>
      ) : (
        <ul className={error ? 'bins stale' : 'bins'}>
          {Object.entries(data).map(([productId, quantity]) => (
            <li key={productId} className={quantity === 0 ? 'bin empty' : 'bin'}>
              <span className="bin-id">{productId}</span>
              <span className="bin-qty">{quantity}</span>
              <span className="bin-note">{quantity === 0 ? 'Out of stock' : 'in stock'}</span>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
