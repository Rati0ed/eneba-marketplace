import { Link } from 'react-router'
import type { ProductCard as Product } from '../api'
import { useApiData } from '../hooks/useApiData'
import ProductCard, { ProductCardSkeleton } from './ProductCard'

type Props = {
  title: string
  endpoint: string // e.g. "/api/home/deals", from the homepage block
  linkUrl: string | null // "Show all" target, or null for no link
}

// A titled, sideways-scrolling row of product cards (Best sellers, Top deals, ...).
export default function ProductRow({ title, endpoint, linkUrl }: Props) {
  const products = useApiData<{ items: Product[] }>(endpoint)

  // Only follow links inside our own site ("/store?..."), never "https://..." or "//evil.com".
  const showLink = linkUrl !== null && linkUrl.startsWith('/') && !linkUrl.startsWith('//')

  if (products.data && products.data.items.length === 0) return null // nothing to show: hide the whole block

  return (
    <section>
      <div className="row-header">
        <h2>{title}</h2>
        {showLink && <Link to={linkUrl}>Show all ›</Link>}
      </div>

      {products.error ? (
        <p className="row-error">
          Couldn't load this section.{' '}
          <button className="link-button" onClick={products.retry}>
            Try again
          </button>
        </p>
      ) : (
        <div className="row-track">
          {products.data
            ? products.data.items.map((product) => <ProductCard key={product.id} product={product} />)
            : Array.from({ length: 6 }, (_, i) => <ProductCardSkeleton key={i} />)}
        </div>
      )}
    </section>
  )
}
