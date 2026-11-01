import { getStats, getStock } from './api.js'
import { usePolling } from './usePolling.js'
import OrderForm from './components/OrderForm.jsx'
import InventoryPanel from './components/InventoryPanel.jsx'
import AnalyticsPanel from './components/AnalyticsPanel.jsx'

const REFRESH_MS = 2000

export default function App() {
  // App loads the data. The panels only display what they receive as props.
  const stock = usePolling(getStock, REFRESH_MS)
  const stats = usePolling(getStats, REFRESH_MS)

  const products = stock.data ? Object.keys(stock.data) : []
  const nothingLoads = stock.error !== null && stats.error !== null

  return (
    <div className="page">
      <header className="masthead">
        <h1>Shop Platform</h1>
        <p>Place an order and watch the services update the stock and the totals.</p>
      </header>

      {nothingLoads && (
        <p className="banner" role="alert">
          Stock and totals can’t be loaded. Check that the containers are running.
        </p>
      )}

      <main className="layout">
        <OrderForm products={products} />
        <InventoryPanel stock={stock} />
        <AnalyticsPanel stats={stats} />
      </main>

      <footer className="footnote">
        Stock and totals refresh every {REFRESH_MS / 1000} seconds.
      </footer>
    </div>
  )
}
