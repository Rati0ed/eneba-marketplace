import { useEffect, useState } from 'react'
import { fetchProducts, formatPrice, imageUrl, type ProductCard } from './api'

// A React component is a function that returns what to show (JSX: HTML-like syntax inside TypeScript).
// Whenever its state changes, React calls the function again and updates the page.
export default function App() {
  // useState creates a value React remembers between renders, plus a function to change it.
  // Changing it with the setter makes React re-render the component.
  const [products, setProducts] = useState<ProductCard[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // useEffect runs code AFTER the component appears on screen: the place for side effects
  // like network requests. The empty list [] at the end means "run only once, on first show".
  useEffect(() => {
    // AbortController can cancel the request. If the component disappears before the
    // answer arrives, the cleanup function below cancels it, so we never update
    // state of a component that is gone. (In development React runs effects twice
    // on purpose to catch missing cleanups; this handles that too.)
    const controller = new AbortController()

    fetchProducts(controller.signal)
      .then((items) => setProducts(items))
      .catch((err: Error) => {
        if (!controller.signal.aborted) setError(err.message)
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false)
      })

    return () => controller.abort() // cleanup function
  }, [])

  return (
    <main className="page">
      <h1>Store</h1>

      {/* Three possible states: loading, error, or the list. */}
      {loading && <p className="status">Loading products…</p>}

      {error && (
        <p className="status error">
          Could not load products: {error}. The API may be waking up (free hosting sleeps when idle); try again in a
          minute.
        </p>
      )}

      {!loading && !error && (
        <>
          <p className="status">{products.length} products</p>
          <ul className="product-list">
            {/* .map turns each product into a list item. `key` helps React track items between renders. */}
            {products.map((product) => (
              <li key={product.id} className="product-row">
                <img src={imageUrl(product.image_url)} alt="" width={60} height={80} loading="lazy" />
                <div className="product-info">
                  <div className="product-title">{product.title}</div>
                  <div className="product-meta">
                    {product.platform} · {product.region} · {product.offer_count} offers
                  </div>
                </div>
                <div className="product-price">
                  {product.lowest_price_cents === null ? (
                    <span className="sold-out">Sold out</span>
                  ) : (
                    <>
                      <span className="from">From</span> {formatPrice(product.lowest_price_cents)}
                    </>
                  )}
                </div>
              </li>
            ))}
          </ul>
        </>
      )}
    </main>
  )
}
