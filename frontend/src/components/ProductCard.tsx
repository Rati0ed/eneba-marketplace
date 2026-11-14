import { Link } from 'react-router'
import { formatPrice, imageUrl, type ProductCard as Product } from '../api'
import './ProductCard.css'

// Props = the inputs of a component, like function parameters.
// `{ product }: { product: Product }` takes the `product` prop and says its type.
export default function ProductCard({ product }: { product: Product }) {
  const soldOut = product.lowest_price_cents === null

  return (
    <Link to={`/product/${product.slug}`} className={soldOut ? 'card sold-out' : 'card'}>
      <div className="card-image">
        {/* alt="" because the title is written right below; screen readers don't need it twice. */}
        <img src={imageUrl(product.image_url)} alt="" loading="lazy" width={300} height={400} />
      </div>

      <div className="card-body">
        <h3 className="card-title">{product.title}</h3>
        <div className="card-region">{product.region}</div>

        {/* lowest_price_cents is null when sold out. The ternary (a ? b : c) is if/else inside JSX. */}
        {product.lowest_price_cents === null ? (
          <div className="card-price-row">
            <span className="card-sold-out">Sold out</span>
          </div>
        ) : (
          <>
            <div className="card-from">
              From
              {product.discount_percent > 0 && (
                <>
                  <s className="card-msrp">{formatPrice(product.msrp_cents)}</s>
                  <span className="card-discount">-{product.discount_percent}%</span>
                </>
              )}
            </div>
            <div className="card-price">{formatPrice(product.lowest_price_cents)}</div>
          </>
        )}
      </div>
    </Link>
  )
}

// A grey placeholder with the same shape as a card, shown while products load.
// The page doesn't "jump" when real cards arrive, because the layout is already there.
export function ProductCardSkeleton() {
  return (
    <div className="card card-skeleton" aria-hidden="true">
      <div className="card-image skeleton" />
      <div className="card-body">
        <div className="skeleton skeleton-line" />
        <div className="skeleton skeleton-line short" />
        <div className="skeleton skeleton-line price" />
      </div>
    </div>
  )
}
