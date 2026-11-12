// Everything the frontend needs to talk to our Go API lives in this file:
// the base URL, the TypeScript types of the JSON we receive, and small fetch helpers.

// Vite reads VITE_* variables from .env files (locally) or from Vercel's settings
// (in production) and inserts their values into the code at BUILD time.
// The fallback lets `npm run dev` work without any .env file.
// .replace(/\/$/, '') removes a trailing slash, so "https://x.com/" + "/api" doesn't become "//api".
export const API_URL = (import.meta.env.VITE_API_URL || 'http://localhost:8080').replace(/\/$/, '')

// The shape of one product in GET /api/products. Must match ProductCard in
// backend/handlers/products.go. TypeScript checks that we only use fields that exist.
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
  offer_count: number
}

// apiGet fetches `path` from our API and returns the parsed JSON.
// <T> is a "generic": the caller says what type of JSON it expects, e.g. apiGet<{ items: ProductCard[] }>.
// `signal` lets the caller cancel the request (see App.tsx).
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

export async function fetchProducts(signal?: AbortSignal): Promise<ProductCard[]> {
  const data = await apiGet<{ items: ProductCard[] }>('/api/products', signal)
  return data.items
}

// The API stores image paths like "/api/covers/x.svg"; images must load from the API's domain.
export function imageUrl(path: string): string {
  return API_URL + path
}

// 1479 -> "€14.79". Intl.NumberFormat formats money correctly for a locale.
const euroFormat = new Intl.NumberFormat('en-IE', { style: 'currency', currency: 'EUR' })
export function formatPrice(cents: number): string {
  return euroFormat.format(cents / 100)
}
