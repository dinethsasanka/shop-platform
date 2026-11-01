import { useState } from 'react'
import { placeOrder } from '../api.js'

export default function OrderForm({ products }) {
  // Each input shows a state value and updates it when the user types.
  const [productId, setProductId] = useState('')
  const [quantity, setQuantity] = useState('1')
  const [customer, setCustomer] = useState('')
  // Result of the last attempt: { kind: 'success' or 'error', text }
  const [result, setResult] = useState(null)
  const [sending, setSending] = useState(false)

  async function handleSubmit(event) {
    event.preventDefault() // stop the browser from reloading the page
    setSending(true)
    setResult(null)
    try {
      const order = await placeOrder({
        product_id: productId,
        quantity: Number(quantity),
        customer: customer.trim(),
      })
      setResult({
        kind: 'success',
        text: `Order ${order.id.slice(0, 8)} placed. Stock is checked in the background.`,
      })
      setCustomer('')
    } catch (err) {
      setResult({ kind: 'error', text: `Order not placed: ${err.message}` })
    } finally {
      setSending(false)
    }
  }

  return (
    <section className="panel panel-order" aria-labelledby="order-heading">
      <h2 id="order-heading">Place an order</h2>
      <form onSubmit={handleSubmit}>
        <label htmlFor="product">Product</label>
        <select
          id="product"
          value={productId}
          onChange={(e) => setProductId(e.target.value)}
          required
        >
          <option value="" disabled>
            {products.length > 0 ? 'Choose a product' : 'Loading products…'}
          </option>
          {products.map((id) => (
            <option key={id} value={id}>
              {id}
            </option>
          ))}
        </select>

        <label htmlFor="quantity">Quantity</label>
        <input
          id="quantity"
          type="number"
          min="1"
          step="1"
          value={quantity}
          onChange={(e) => setQuantity(e.target.value)}
          required
        />

        <label htmlFor="customer">Customer name</label>
        <input
          id="customer"
          type="text"
          autoComplete="off"
          value={customer}
          onChange={(e) => setCustomer(e.target.value)}
          required
        />

        <button type="submit" disabled={sending}>
          {sending ? 'Placing order…' : 'Place order'}
        </button>
      </form>

      <p className={result ? `result ${result.kind}` : 'result'} role="status">
        {result ? result.text : ''}
      </p>
    </section>
  )
}
