import { useEffect, useRef, useState, type FormEvent, type KeyboardEvent } from 'react'
import { Link, useNavigate } from 'react-router'
import { fetchSuggestions, formatPrice, imageUrl, type SearchSuggestions } from '../api'
import Highlight from './Highlight'
import './SearchBar.css'

const MIN_CHARS = 2 // don't search for a single letter
const DEBOUNCE_MS = 250 // wait this long after the last keystroke before asking the API

// The header search: a text box with a live dropdown of the best matches.
//   - typing waits 250 ms after the last key ("debounce"), then asks /api/search/suggest
//   - ↑ / ↓ move through results, Enter opens the highlighted product,
//     Enter with nothing highlighted opens the store with ?q=..., Esc closes
//   - clicking outside closes the dropdown
export default function SearchBar({ initialQuery }: { initialQuery: string }) {
  const navigate = useNavigate()
  const inputRef = useRef<HTMLInputElement>(null) // direct access to the <input>, e.g. to blur it

  const [text, setText] = useState(initialQuery)
  const [open, setOpen] = useState(false)
  const [activeIndex, setActiveIndex] = useState(-1) // -1 = nothing highlighted

  // Like on the store page: remember WHICH query the stored results belong to.
  const [loaded, setLoaded] = useState<{ query: string; data: SearchSuggestions | null; failed: boolean }>({
    query: '',
    data: null,
    failed: false,
  })

  const query = text.trim()
  const wantsResults = query.length >= MIN_CHARS
  const loading = wantsResults && loaded.query !== query
  const data = loaded.data // may be from the previous query while the new one loads (shown faded)
  const items = data?.items ?? []
  const showDropdown = open && wantsResults

  // Debounced search: every keystroke changes `query`, which re-runs this effect.
  // The cleanup cancels the PREVIOUS timer, so only the last keystroke's timer survives
  // the 250 ms and actually calls the API. Typing "elden" = 1 request, not 5.
  useEffect(() => {
    if (!wantsResults) return
    const controller = new AbortController()
    const timer = setTimeout(() => {
      fetchSuggestions(query, controller.signal)
        .then((result) => setLoaded({ query, data: result, failed: false }))
        .catch(() => {
          if (!controller.signal.aborted) setLoaded({ query, data: null, failed: true })
        })
    }, DEBOUNCE_MS)

    return () => {
      clearTimeout(timer) // typed again within 250 ms: forget this search
      controller.abort() // a request already started: cancel it
    }
  }, [query, wantsResults])

  function close() {
    setOpen(false)
    setActiveIndex(-1)
  }

  // Enter with nothing highlighted, or "See all results": the store page with ?q=...
  function searchInStore() {
    close()
    inputRef.current?.blur() // hides the phone keyboard
    navigate(query ? '/store?' + new URLSearchParams({ q: query }) : '/store')
  }

  function handleSubmit(event: FormEvent) {
    event.preventDefault() // a form submit would reload the page
    searchInStore()
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault() // otherwise the cursor jumps to the start/end of the text
      setOpen(true)
      if (items.length === 0) return
      // % (modulo) wraps around: after the last item comes the first again.
      const step = event.key === 'ArrowDown' ? 1 : -1
      setActiveIndex((current) => (current + step + items.length) % items.length)
    } else if (event.key === 'Enter' && activeIndex >= 0 && items[activeIndex]) {
      event.preventDefault() // don't submit the form: open the highlighted product instead
      close()
      inputRef.current?.blur()
      navigate(`/product/${items[activeIndex].slug}`)
    } else if (event.key === 'Escape') {
      event.preventDefault() // browsers clear a type="search" box on Esc; we only want to close the list
      close()
    }
  }

  // Ids connect the input to the list for screen readers (the ARIA "combobox" pattern).
  const listId = 'search-results'
  const optionId = (i: number) => `search-option-${i}`

  return (
    <form className="search" role="search" onSubmit={handleSubmit}>
      <input
        ref={inputRef}
        type="search"
        className="search-input"
        placeholder="Search games, gift cards…"
        value={text}
        onChange={(e) => {
          setText(e.target.value)
          setOpen(true)
          setActiveIndex(-1) // new text = new results, old highlight means nothing
        }}
        onFocus={() => setOpen(true)}
        onClick={() => setOpen(true)} // after Esc the input still has focus, so onFocus won't fire again
        onBlur={close}
        onKeyDown={handleKeyDown}
        maxLength={100}
        autoComplete="off"
        role="combobox"
        aria-label="Search products"
        aria-expanded={showDropdown}
        aria-controls={listId}
        aria-autocomplete="list"
        aria-activedescendant={activeIndex >= 0 ? optionId(activeIndex) : undefined}
      />
      <button type="submit" className="search-button" aria-label="Search">
        {/* A magnifying glass drawn with SVG: a circle and a line */}
        <svg width="20" height="20" viewBox="0 0 24 24" aria-hidden="true">
          <circle cx="11" cy="11" r="7" fill="none" stroke="currentColor" strokeWidth="2.5" />
          <line x1="16.5" y1="16.5" x2="21" y2="21" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" />
        </svg>
      </button>

      {showDropdown && (
        // onMouseDown + preventDefault: clicking inside the dropdown must NOT blur the input,
        // or onBlur would close the dropdown before the click on a result registers.
        <div className="search-dropdown" onMouseDown={(e) => e.preventDefault()}>
          {loaded.failed && !loading ? (
            <p className="search-message">Search is unavailable right now. Try again in a moment.</p>
          ) : items.length === 0 && loading ? (
            <p className="search-message">Searching…</p>
          ) : items.length === 0 && data ? (
            <div className="search-message">
              No results for “{query}”.
              {data.suggestion && (
                <>
                  {' '}
                  Did you mean{' '}
                  <Link to={`/product/${data.suggestion.slug}`} onClick={close}>
                    {data.suggestion.title}
                  </Link>
                  ?
                </>
              )}
            </div>
          ) : (
            <>
              <ul id={listId} role="listbox" className={loading ? 'search-results stale' : 'search-results'}>
                {items.map((product, i) => (
                  <li key={product.id} id={optionId(i)} role="option" aria-selected={i === activeIndex}>
                    <Link
                      to={`/product/${product.slug}`}
                      className={i === activeIndex ? 'search-result active' : 'search-result'}
                      onClick={close}
                      onMouseEnter={() => setActiveIndex(i)}
                      tabIndex={-1} // keyboard users move with ↑/↓, not Tab
                    >
                      <img src={imageUrl(product.image_url)} alt="" width={36} height={48} />
                      <span className="search-result-text">
                        <span className="search-result-title">
                          <Highlight text={product.title} query={query} />
                        </span>
                        <span className="search-result-meta">
                          {product.platform} · {product.region}
                        </span>
                      </span>
                      <span className="search-result-price">
                        {product.lowest_price_cents === null ? 'Sold out' : formatPrice(product.lowest_price_cents)}
                      </span>
                    </Link>
                  </li>
                ))}
              </ul>
              {data && data.total > 0 && (
                <button type="button" className="search-all" onClick={searchInStore}>
                  See all {data.total} results for “{query}”
                </button>
              )}
            </>
          )}
        </div>
      )}
    </form>
  )
}
