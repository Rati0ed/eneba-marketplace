import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router'
import { fetchProductFilters, fetchProducts, type ProductFilters, type ProductList } from '../api'
import ProductCard, { ProductCardSkeleton } from '../components/ProductCard'
import FilterSidebar from '../components/FilterSidebar'
import Pagination from '../components/Pagination'
import './StorePage.css'

const SORT_OPTIONS = [
  { value: 'popular', label: 'Best sellers' },
  { value: 'price_asc', label: 'Price: low to high' },
  { value: 'price_desc', label: 'Price: high to low' },
  { value: 'discount', label: 'Biggest discount' },
  { value: 'newest', label: 'Newest' },
]

// Multi-value filters: the URL can contain each of these several times (?platform=Steam&platform=Xbox).
const LIST_FILTERS = ['platform', 'region', 'genre']

// The store page. The URL is the single source of truth for every filter:
//   /store?platform=Steam&min_price=10&sort=price_asc&page=2
// Clicking a filter only changes the URL; changing the URL triggers a new API request.
// That's why back/forward buttons, refresh and shared links all keep the filters.
export default function StorePage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const queryString = searchParams.toString() // e.g. "platform=Steam&sort=price_asc"

  const [retryCount, setRetryCount] = useState(0) // bumping this re-runs the effect ("Try again")
  const [filters, setFilters] = useState<ProductFilters | null>(null)
  const [filtersOpen, setFiltersOpen] = useState(false) // mobile only: show/hide the sidebar

  // `loaded` remembers the last finished request: WHICH request it was (key), its result and error.
  // We don't store "loading" at all: we're loading whenever the last finished request is not
  // the one for the current URL. Values you can calculate shouldn't be separate state,
  // because two states can get out of sync; a calculation can't.
  const requestKey = `${queryString}#${retryCount}`
  const [loaded, setLoaded] = useState<{ key: string; result: ProductList | null; error: string | null }>({
    key: '',
    result: null,
    error: null,
  })
  const loading = loaded.key !== requestKey
  const result = loaded.result // while loading, this is still the PREVIOUS result (shown faded)
  const error = loading ? null : loaded.error

  // Load products whenever the URL query changes (or "Try again" is clicked).
  useEffect(() => {
    const controller = new AbortController()

    fetchProducts(queryString, controller.signal)
      .then((data) => setLoaded({ key: requestKey, result: data, error: null }))
      .catch((err: Error) => {
        if (controller.signal.aborted) return
        // Keep the previous result (prev.result) so the page doesn't go blank on an error.
        setLoaded((prev) => ({ key: requestKey, result: prev.result, error: err.message }))
      })

    // Cleanup: if the URL changes again before this request finishes, cancel it.
    // Without this, a slow OLD response could arrive last and overwrite the NEW results.
    return () => controller.abort()
  }, [queryString, requestKey])

  // Load the filter options once.
  useEffect(() => {
    const controller = new AbortController()
    fetchProductFilters(controller.signal)
      .then(setFilters)
      .catch(() => {
        /* the sidebar keeps showing "Loading filters…"; products still work */
      })
    return () => controller.abort()
  }, [])

  // ---------- Helpers that change the URL ----------

  // Copy the current params, let `change` edit the copy, then save it to the URL.
  // Any filter change goes back to page 1 (page 3 of the OLD results means nothing now).
  function updateParams(change: (params: URLSearchParams) => void, resetPage = true) {
    const next = new URLSearchParams(searchParams)
    change(next)
    if (resetPage) next.delete('page')
    setSearchParams(next)
  }

  function toggleFilter(key: string, value: string) {
    updateParams((params) => {
      const values = params.getAll(key)
      params.delete(key)
      // Re-add every value except the clicked one; add the clicked one if it wasn't there.
      for (const v of values) if (v !== value) params.append(key, v)
      if (!values.includes(value)) params.append(key, value)
    })
  }

  function setPrice(min: string, max: string) {
    updateParams((params) => {
      if (min) params.set('min_price', min)
      else params.delete('min_price')
      if (max) params.set('max_price', max)
      else params.delete('max_price')
    })
  }

  function setSort(sort: string) {
    updateParams((params) => params.set('sort', sort))
  }

  function setPage(page: number) {
    updateParams((params) => params.set('page', String(page)), false)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  // Remove every filter but keep the search text (if any).
  function clearFilters() {
    const q = searchParams.get('q')
    setSearchParams(q ? { q } : {})
  }

  // ---------- Values read from the URL ----------

  const selected: Record<string, string[]> = {}
  for (const key of LIST_FILTERS) selected[key] = searchParams.getAll(key)
  const minPrice = searchParams.get('min_price') ?? ''
  const maxPrice = searchParams.get('max_price') ?? ''
  const query = searchParams.get('q') ?? ''

  // Chips above the grid: one per active filter, each removable with ×.
  const chips: { label: string; remove: () => void }[] = []
  for (const key of LIST_FILTERS) {
    for (const value of selected[key]) chips.push({ label: value, remove: () => toggleFilter(key, value) })
  }
  if (minPrice || maxPrice) {
    chips.push({
      label: `€${minPrice || '0'} – ${maxPrice ? '€' + maxPrice : 'any'}`,
      remove: () => setPrice('', ''),
    })
  }

  return (
    <main className="page store">
      <div className="store-heading">
        <h1>{query ? `Results for “${query}”` : 'Store'}</h1>
        {result && !error && (
          <span className="store-count">
            {result.total} {result.total === 1 ? 'product' : 'products'}
          </span>
        )}
      </div>

      <div className="store-layout">
        <aside className={filtersOpen ? 'store-sidebar open' : 'store-sidebar'}>
          <FilterSidebar
            filters={filters}
            selected={selected}
            minPrice={minPrice}
            maxPrice={maxPrice}
            onToggle={toggleFilter}
            onPriceChange={setPrice}
          />
        </aside>

        <section className="store-results">
          <div className="store-toolbar">
            <button className="btn btn-secondary store-filters-toggle" onClick={() => setFiltersOpen(!filtersOpen)}>
              {filtersOpen ? 'Hide filters' : 'Filters'}
              {chips.length > 0 && ` (${chips.length})`}
            </button>

            <label className="sort-select">
              <span>Sort by</span>
              <select value={result?.sort ?? searchParams.get('sort') ?? 'popular'} onChange={(e) => setSort(e.target.value)}>
                {SORT_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>
          </div>

          {chips.length > 0 && (
            <div className="chips">
              {chips.map((chip) => (
                <button key={chip.label} className="chip" onClick={chip.remove} aria-label={`Remove filter ${chip.label}`}>
                  {chip.label} <span aria-hidden="true">×</span>
                </button>
              ))}
              <button className="link-button" onClick={clearFilters}>
                Clear filters
              </button>
            </div>
          )}

          {error ? (
            <div className="empty-state">
              <h2>Something went wrong</h2>
              <p>{error}</p>
              <p className="muted">If the site was idle, the free server may be waking up (up to a minute).</p>
              <div className="empty-actions">
                <button className="btn btn-primary" onClick={() => setRetryCount(retryCount + 1)}>
                  Try again
                </button>
                <button className="btn btn-secondary" onClick={clearFilters}>
                  Clear filters
                </button>
              </div>
            </div>
          ) : loading && !result ? (
            // First load: skeleton cards. Array.from({length: 12}) makes a list of 12 to map over.
            <div className="product-grid">
              {Array.from({ length: 12 }, (_, i) => (
                <ProductCardSkeleton key={i} />
              ))}
            </div>
          ) : result && result.items.length === 0 ? (
            <div className="empty-state">
              <h2>No products found</h2>
              <p className="muted">Try removing some filters or widening the price range.</p>
              <div className="empty-actions">
                <button className="btn btn-primary" onClick={clearFilters}>
                  Clear filters
                </button>
              </div>
            </div>
          ) : (
            result && (
              <>
                {/* While NEW results load, keep showing the old ones, faded. Less flicker than skeletons. */}
                <div className={loading ? 'product-grid refreshing' : 'product-grid'}>
                  {result.items.map((product) => (
                    <ProductCard key={product.id} product={product} />
                  ))}
                </div>
                <Pagination page={result.page} totalPages={result.total_pages} onChange={setPage} />
              </>
            )
          )}
        </section>
      </div>
    </main>
  )
}
