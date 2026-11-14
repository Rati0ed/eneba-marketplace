// Everything the frontend needs to talk to our Go API lives in this file:
// the base URL, the TypeScript types of the JSON we receive, and small fetch helpers.

// Vite reads VITE_* variables from .env files (locally) or from Vercel's settings
// (in production) and inserts their values into the code at BUILD time.
// The fallback lets `npm run dev` work without any .env file.
// .replace(/\/$/, '') removes a trailing slash, so "https://x.com/" + "/api" doesn't become "//api".
export const API_URL = (import.meta.env.VITE_API_URL || 'http://localhost:8080').replace(/\/$/, '')

// ---------- Types: the shapes of the JSON our API sends ----------
// They must match the Go structs (backend/handlers/cards.go, products.go).
// TypeScript then checks that we only use fields that exist.
// Python comparison: like a TypedDict / dataclass describing a dict.

export type ProductCard = {
  id: number
  slug: string
  title: string
  platform: string
  region: string
  genre: string
  image_url: string
  msrp_cents: number
  lowest_price_cents: number | null // null = every offer is sold out
  discount_percent: number // 0 = no discount
  offer_count: number
}

export type ProductList = {
  items: ProductCard[]
  total: number
  page: number
  page_size: number
  total_pages: number
  sort: string
  suggestion: ProductCard | null // "did you mean": only set when a search found nothing
}

export type SearchSuggestions = {
  items: ProductCard[] // best matches, max 6
  total: number // how many products match in total
  suggestion: ProductCard | null // "did you mean", when nothing matched
}

export type FilterOption = { value: string; count: number }

export type ProductFilters = {
  platforms: FilterOption[]
  regions: FilterOption[]
  genres: FilterOption[]
  price: { min: number; max: number } // whole euros
}

// ---------- Fetch helpers ----------

// apiGet fetches `path` from our API and returns the parsed JSON.
// <T> is a "generic": the caller says what type of JSON it expects, e.g. apiGet<ProductList>.
// `signal` lets the caller cancel the request (see the useEffect cleanups in the pages).
async function apiGet<T>(path: string, signal?: AbortSignal): Promise<T> {
  // `await` pauses this function until the network answers, without freezing the page.
  const response = await fetch(API_URL + path, { signal })

  if (!response.ok) {
    // Our backend always sends errors as {"error": "..."}; use that message if we can read it.
    let message = `Request failed (${response.status})`
    try {
      const body = await response.json()
      if (body.error) message = body.error
    } catch {
      // The body wasn't JSON. Keep the generic message.
    }
    throw new Error(message)
  }
  return (await response.json()) as T
}

// The store page passes its URL query straight through (e.g. "platform=Steam&sort=price_asc"),
// because the store URL and the API use the SAME parameter names. One source of truth: the URL.
export function fetchProducts(query: string, signal?: AbortSignal): Promise<ProductList> {
  return apiGet<ProductList>('/api/products' + (query ? '?' + query : ''), signal)
}

export function fetchProductFilters(signal?: AbortSignal): Promise<ProductFilters> {
  return apiGet<ProductFilters>('/api/products/filters', signal)
}

export function fetchSuggestions(query: string, signal?: AbortSignal): Promise<SearchSuggestions> {
  // URLSearchParams encodes the text safely: "gta 5 & more" -> "q=gta+5+%26+more".
  // Gluing raw text into a URL would break on characters like & ? #.
  return apiGet<SearchSuggestions>('/api/search/suggest?' + new URLSearchParams({ q: query }), signal)
}

// ---------- Small formatting helpers ----------

// The API stores image paths like "/api/covers/x.svg"; images must load from the API's domain.
export function imageUrl(path: string): string {
  return API_URL + path
}

// 1479 -> "€14.79". Intl.NumberFormat formats money correctly for a locale.
const euroFormat = new Intl.NumberFormat('en-IE', { style: 'currency', currency: 'EUR' })
export function formatPrice(cents: number): string {
  return euroFormat.format(cents / 100)
}
