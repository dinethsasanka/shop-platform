// All requests go to /api/... on the page's own origin.
// In development, the Vite server forwards them to the gateway (see vite.config.js).

async function request(path, options) {
  const response = await fetch(path, options)
  if (!response.ok) {
    // The services send their error messages as plain text.
    // Other error bodies (for example an HTML error page) are not shown.
    const type = response.headers.get('Content-Type') || ''
    const text = type.startsWith('text/plain') ? (await response.text()).trim() : ''
    throw new Error(`${text || 'server error'} (HTTP ${response.status})`)
  }
  return response.json()
}

export function getStock() {
  return request('/api/stock')
}

export function getStats() {
  return request('/api/stats')
}

export function placeOrder(order) {
  return request('/api/orders', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(order),
  })
}
