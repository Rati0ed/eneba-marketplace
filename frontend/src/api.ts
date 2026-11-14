// Everything the frontend needs to talk to our Go API lives in this file:
// the base URL, the TypeScript types of the JSON we receive, and fetch helpers.

// Vite reads VITE_* variables from .env files (locally) or from Vercel's settings
// (in production) and inserts their values into the code at BUILD time.
// The fallback lets `npm run dev` work without any .env file.
// .replace(/\/$/, '') removes a trailing slash, so "https://x.com/" + "/api" doesn't become "//api".
export const API_URL = (import.meta.env.VITE_API_URL || 'http://localhost:8080').replace(/\/$/, '')

// ---------- Types: the shapes of the JSON our API sends ----------
// They must match the Go structs in backend/handlers/. TypeScript then checks that we
// only use fields that exist. Python comparison: like a TypedDict describing a dict.

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

// "&" combines two types: a ProductDetail has every ProductCard field plus these two.
export type ProductDetail = ProductCard & {
  description: string
  release_date: string // "2022-02-25"
}

export type Offer = {
  id: number
  seller_name: string
  seller_rating_percent: number
  seller_sales_count: number
  price_cents: number
  stock: number // 0 = sold out
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

export type HomeBlock = {
  id: number
  type: string // "banner" | "product_row" (unknown types are skipped, so the API can add new ones safely)
  title: string
  endpoint: string // where the block loads its content, e.g. "/api/home/deals"
  link_url: string | null // "Show all" link
}

export type Banner = {
  id: number
  title: string
  subtitle: string
  image_url: string
  link_url: string
}

export type User = {
  id: number
  email: string
  display_name: string
}

export type AuthResponse = {
  token: string // the JWT; sent back in the Authorization header on protected requests
  user: User
}

// What the browser stores in the cart: just WHICH offer and HOW MANY. No prices:
// prices always come fresh from the server (/api/cart/quote), so they can't be faked.
export type CartLine = {
  offer_id: number
  quantity: number
}

export type QuoteLine = {
  offer_id: number
  quantity: number
  product_slug: string
  product_title: string
  image_url: string
  platform: string
  region: string
  seller_name: string
  unit_price_cents: number
  stock: number
  line_total_cents: number
  // Why this line can't be bought as-is ("" = fine).
  problem: '' | 'not_found' | 'out_of_stock' | 'not_enough_stock'
}

export type Quote = {
  items: QuoteLine[]
  total_cents: number
  can_checkout: boolean // false if any line has a problem or the cart is empty
}

export type OrderItem = {
  id: number
  offer_id: number
  product_title: string
  product_slug: string
  image_url: string
  platform: string
  quantity: number
  unit_price_cents: number
  keys: string[] // only filled once the order is paid
}

export type Order = {
  id: number
  status: 'pending' | 'paid' | 'failed'
  total_cents: number
  created_at: string
  paid_at: string | null
  card_last4: string | null
  items: OrderItem[]
}

export type PaymentDetails = {
  cardholder: string
  card_number: string
  expiry: string // "MM/YY"
  cvc: string
}

// ---------- Making requests ----------

// ApiError carries the HTTP status, so pages can react differently to
// 401 (log in again), 402 (card declined), 404 (not found), 409 (sold out)...
// `extends Error` = an Error with one extra field (like subclassing Exception in Python).
export class ApiError extends Error {
  status: number // 0 = network problem (server unreachable)
  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

type RequestOptions = {
  method?: 'GET' | 'POST'
  body?: unknown // any value; sent as JSON
  token?: string | null // JWT for protected endpoints
  signal?: AbortSignal // lets the caller cancel the request
}

// apiRequest calls our API and returns the parsed JSON, or throws an ApiError.
// <T> is a "generic": the caller says what type of JSON it expects.
export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = {}
  if (options.body !== undefined) headers['Content-Type'] = 'application/json'
  if (options.token) headers['Authorization'] = `Bearer ${options.token}`

  let response: Response
  try {
    // `await` pauses this function until the network answers, without freezing the page.
    response = await fetch(API_URL + path, {
      method: options.method ?? 'GET',
      headers,
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
      signal: options.signal,
    })
  } catch (err) {
    if (options.signal?.aborted) throw err // cancelled on purpose: let the caller ignore it
    throw new ApiError('Cannot reach the server. If the site was idle, it may need up to a minute to wake up.', 0)
  }

  if (!response.ok) {
    // Our backend always sends errors as {"error": "..."}; use that message if we can read it.
    let message = `Request failed (${response.status})`
    try {
      const body = await response.json()
      if (body.error) message = body.error
    } catch {
      // The body wasn't JSON. Keep the generic message.
    }
    throw new ApiError(message, response.status)
  }
  return (await response.json()) as T
}

// ---------- One small function per endpoint ----------

// The store page passes its URL query straight through (e.g. "platform=Steam&sort=price_asc"),
// because the store URL and the API use the SAME parameter names. One source of truth: the URL.
export function fetchProducts(query: string, signal?: AbortSignal): Promise<ProductList> {
  return apiRequest<ProductList>('/api/products' + (query ? '?' + query : ''), { signal })
}

export function fetchProductFilters(signal?: AbortSignal): Promise<ProductFilters> {
  return apiRequest<ProductFilters>('/api/products/filters', { signal })
}

export function fetchSuggestions(query: string, signal?: AbortSignal): Promise<SearchSuggestions> {
  // URLSearchParams encodes the text safely: "gta 5 & more" -> "q=gta+5+%26+more".
  // Gluing raw text into a URL would break on characters like & ? #.
  return apiRequest<SearchSuggestions>('/api/search/suggest?' + new URLSearchParams({ q: query }), { signal })
}

export function register(email: string, password: string, displayName: string): Promise<AuthResponse> {
  return apiRequest<AuthResponse>('/api/auth/register', {
    method: 'POST',
    body: { email, password, display_name: displayName },
  })
}

export function login(email: string, password: string): Promise<AuthResponse> {
  return apiRequest<AuthResponse>('/api/auth/login', { method: 'POST', body: { email, password } })
}

export function fetchMe(token: string, signal?: AbortSignal): Promise<User> {
  return apiRequest<User>('/api/auth/me', { token, signal })
}

export function fetchQuote(items: CartLine[], signal?: AbortSignal): Promise<Quote> {
  return apiRequest<Quote>('/api/cart/quote', { method: 'POST', body: { items }, signal })
}

export function createOrder(items: CartLine[], token: string): Promise<Order> {
  return apiRequest<Order>('/api/orders', { method: 'POST', body: { items }, token })
}

export function payOrder(orderId: number, payment: PaymentDetails, token: string): Promise<Order> {
  return apiRequest<Order>(`/api/orders/${orderId}/pay`, { method: 'POST', body: payment, token })
}

// ---------- Small helpers ----------

// The API stores image paths like "/api/covers/x.svg"; images must load from the API's domain.
export function imageUrl(path: string): string {
  return API_URL + path
}

// Homepage blocks tell us which endpoint to call. We only follow paths on OUR API
// ("/api/..."), never a full URL, so a bad database row can't make the browser
// send requests to some other website.
export function isSafeApiPath(path: string): boolean {
  return path.startsWith('/api/') && !path.includes('//') && !path.includes('..')
}

// 1479 -> "€14.79". Intl.NumberFormat formats money correctly for a locale.
const euroFormat = new Intl.NumberFormat('en-IE', { style: 'currency', currency: 'EUR' })
export function formatPrice(cents: number): string {
  return euroFormat.format(cents / 100)
}

// SQLite stores dates like "2026-11-14 18:05:12" in UTC. Turning it into "2026-11-14T18:05:12Z"
// (Z = UTC) lets the browser convert it to the user's local time: "14 Nov 2026, 20:05".
export function formatDate(sqliteDate: string): string {
  return new Date(sqliteDate.replace(' ', 'T') + 'Z').toLocaleString('en-GB', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}
