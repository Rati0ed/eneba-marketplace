import { useState } from 'react'
import { Link, useParams } from 'react-router'
import { formatPrice, imageUrl, type Offer, type ProductDetail } from '../api'
import { useApiData } from '../hooks/useApiData'
import { useCart } from '../hooks/useCart'
import './ProductPage.css'

// Short "how to activate" help per platform (UI help text, not catalog data).
const ACTIVATION_HELP: Record<string, string> = {
  Steam: 'Open Steam → Games → Activate a Product on Steam, and enter the key.',
  PlayStation: 'On PlayStation Store, open your profile → Redeem codes, and enter the key.',
  Xbox: 'Go to microsoft.com/redeem or the Xbox Store → Redeem a code, and enter the key.',
  Nintendo: 'Open Nintendo eShop → Redeem Code, and enter the key.',
}

// /product/:slug: product details + every seller's offer, so the buyer can compare and choose.
export default function ProductPage() {
  const { slug = '' } = useParams() // ":slug" from the URL

  // Two independent requests run at the same time (in parallel): details and offers.
  // encodeURIComponent keeps a strange slug from changing the API path.
  const product = useApiData<ProductDetail>(`/api/products/${encodeURIComponent(slug)}`)
  const offers = useApiData<{ items: Offer[] }>(`/api/products/${encodeURIComponent(slug)}/offers`)

  if (product.error?.status === 404) {
    return (
      <main className="page">
        <div className="empty-state">
          <h1>Product not found</h1>
          <p className="muted">It may have been removed, or the link is wrong.</p>
          <Link to="/store" className="btn btn-primary">
            Browse the store
          </Link>
        </div>
      </main>
    )
  }

  if (product.error) {
    return (
      <main className="page">
        <div className="empty-state">
          <h1>Something went wrong</h1>
          <p className="muted">{product.error.message}</p>
          <button className="btn btn-primary" onClick={product.retry}>
            Try again
          </button>
        </div>
      </main>
    )
  }

  const p = product.data
  if (!p) return <ProductPageSkeleton />

  return (
    <main className="page product-page">
      {/* Breadcrumb: where am I? Store > Steam > this product */}
      <nav className="breadcrumb" aria-label="Breadcrumb">
        <Link to="/store">Store</Link> <span aria-hidden="true">›</span>{' '}
        <Link to={'/store?' + new URLSearchParams({ platform: p.platform })}>{p.platform}</Link>{' '}
        <span aria-hidden="true">›</span> <span>{p.title}</span>
      </nav>

      <div className="product-top">
        <img className="product-cover" src={imageUrl(p.image_url)} alt={`${p.title} cover`} width={300} height={400} />

        <div className="product-info">
          <h1>{p.title}</h1>
          <div className="product-tags">
            <span className="tag">{p.platform}</span>
            <span className="tag tag-region">{p.region}</span>
            <span className="tag">{p.genre}</span>
          </div>

          <div className="product-price-box">
            {p.lowest_price_cents === null ? (
              <span className="product-sold-out">Currently sold out</span>
            ) : (
              <>
                <span className="muted">Best price from {p.offer_count} {p.offer_count === 1 ? 'seller' : 'sellers'}</span>
                <div className="product-price-line">
                  <span className="product-price">{formatPrice(p.lowest_price_cents)}</span>
                  {p.discount_percent > 0 && (
                    <>
                      <s className="muted">{formatPrice(p.msrp_cents)}</s>
                      <span className="card-discount">-{p.discount_percent}%</span>
                    </>
                  )}
                </div>
                <a href="#offers" className="btn btn-primary">
                  Compare offers
                </a>
              </>
            )}
          </div>

          <dl className="product-facts">
            <dt>Release date</dt>
            {/* new Date("2022-02-25") -> "25 February 2022" */}
            <dd>{new Date(p.release_date).toLocaleDateString('en-GB', { day: 'numeric', month: 'long', year: 'numeric' })}</dd>
            <dt>Region</dt>
            <dd>{p.region === 'Global' ? 'Works worldwide' : `Can be activated in ${p.region}`}</dd>
            <dt>Delivery</dt>
            <dd>Digital key, instantly after payment</dd>
          </dl>
        </div>
      </div>

      <section className="product-section">
        <h2>About</h2>
        <p>{p.description}</p>
        {ACTIVATION_HELP[p.platform] && (
          <p className="muted">
            <strong>How to activate:</strong> {ACTIVATION_HELP[p.platform]}
          </p>
        )}
      </section>

      <section className="product-section" id="offers">
        <h2>Offers {offers.data && <span className="muted">({offers.data.items.length})</span>}</h2>
        {offers.loading && <p className="muted">Loading offers…</p>}
        {offers.error && (
          <p className="error-text">
            Could not load offers: {offers.error.message}{' '}
            <button className="link-button" onClick={offers.retry}>
              Try again
            </button>
          </p>
        )}
        {offers.data && (
          <ul className="offer-list">
            {offers.data.items.map((offer, i) => (
              <OfferRow key={offer.id} offer={offer} best={i === 0 && offer.stock > 0} />
            ))}
          </ul>
        )}
      </section>
    </main>
  )
}

// One seller's offer with an "Add to cart" button.
function OfferRow({ offer, best }: { offer: Offer; best: boolean }) {
  const cart = useCart()
  const [justAdded, setJustAdded] = useState(false)
  const soldOut = offer.stock === 0
  const inCart = cart.lines.find((line) => line.offer_id === offer.id)?.quantity ?? 0
  const atLimit = inCart >= offer.stock // can't put more keys in the cart than the seller has

  function handleAdd() {
    cart.add(offer.id, offer.stock)
    setJustAdded(true)
    setTimeout(() => setJustAdded(false), 1500) // show "Added ✓" for 1.5 s
  }

  return (
    <li className={soldOut ? 'offer sold-out' : best ? 'offer best' : 'offer'}>
      {/* The first letter of the seller's name as a simple avatar */}
      <div className="offer-avatar" aria-hidden="true">
        {offer.seller_name[0]}
      </div>
      <div className="offer-seller">
        <div className="offer-seller-name">
          {offer.seller_name} {best && <span className="best-tag">Best price</span>}
        </div>
        <div className="muted offer-seller-meta">
          {offer.seller_rating_percent}% positive · {offer.seller_sales_count.toLocaleString('en-GB')} sales
        </div>
      </div>
      <div className={soldOut ? 'offer-stock out' : offer.stock <= 3 ? 'offer-stock low' : 'offer-stock'}>
        {soldOut ? 'Sold out' : offer.stock <= 3 ? `Only ${offer.stock} left` : `${offer.stock} in stock`}
      </div>
      <div className="offer-price">{formatPrice(offer.price_cents)}</div>
      <button className="btn btn-primary offer-button" onClick={handleAdd} disabled={soldOut || atLimit}>
        {soldOut ? 'Sold out' : justAdded ? 'Added ✓' : atLimit ? `In cart (${inCart})` : inCart > 0 ? `Add another (${inCart})` : 'Add to cart'}
      </button>
    </li>
  )
}

function ProductPageSkeleton() {
  return (
    <main className="page product-page" aria-busy="true">
      <div className="product-top">
        <div className="product-cover skeleton" />
        <div className="product-info">
          <div className="skeleton skeleton-line" style={{ height: 32, width: '80%' }} />
          <div className="skeleton skeleton-line short" style={{ marginTop: 16 }} />
          <div className="skeleton skeleton-line" style={{ height: 120, marginTop: 24 }} />
        </div>
      </div>
    </main>
  )
}
