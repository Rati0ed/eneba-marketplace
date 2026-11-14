import { Link, NavLink, useLocation } from 'react-router'
import SearchBar from './SearchBar'
import { useCart } from '../hooks/useCart'
import { useAuth } from '../hooks/useAuth'
import './Header.css'

// The purple bar at the top of every page.
export default function Header() {
  // On the store page, the search box shows the current ?q= so the user sees what they searched.
  const location = useLocation()
  const urlQuery = location.pathname === '/store' ? (new URLSearchParams(location.search).get('q') ?? '') : ''
  const { count } = useCart()
  const auth = useAuth()

  return (
    <header className="header">
      <div className="header-inner">
        {/* <Link> changes the URL without reloading the page (unlike a normal <a href>). */}
        <Link to="/" className="logo" aria-label="GameKey home">
          <img src="/favicon.svg" alt="" width={32} height={32} />
          <span>
            game<strong>key</strong>
          </span>
        </Link>

        <div className="header-search-slot">
          {/* key={urlQuery}: when the URL's search text changes (new search, or back button),
              React creates a fresh SearchBar starting from that text and with the dropdown closed. */}
          <SearchBar key={urlQuery} initialQuery={urlQuery} />
        </div>

        <nav className="header-nav">
          {/* NavLink adds class "active" when its page is open, so we can highlight it. */}
          <NavLink to="/store" className="hide-on-phone">
            Store
          </NavLink>
          {auth.user ? (
            <>
              {/* title = tooltip with the full name/email */}
              <NavLink to="/orders" title={`Signed in as ${auth.user.email}`}>
                Orders
              </NavLink>
              <button className="header-logout" onClick={auth.logout}>
                Log out
              </button>
            </>
          ) : (
            // Remember the current page, so after signing in the user comes back here.
            <NavLink to={'/login?' + new URLSearchParams({ next: location.pathname + location.search })}>Sign in</NavLink>
          )}
          <NavLink to="/cart" className="cart-link" aria-label={`Cart, ${count} ${count === 1 ? 'item' : 'items'}`}>
            {/* Shopping cart icon drawn with SVG */}
            <svg width="22" height="22" viewBox="0 0 24 24" aria-hidden="true">
              <path d="M3 4h2l2.4 10.2a1 1 0 0 0 1 .8h8.9a1 1 0 0 0 1-.8L20 8H6.2" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
              <circle cx="9" cy="19" r="1.6" fill="currentColor" />
              <circle cx="17" cy="19" r="1.6" fill="currentColor" />
            </svg>
            {count > 0 && <span className="cart-badge">{count > 99 ? '99+' : count}</span>}
          </NavLink>
        </nav>
      </div>
    </header>
  )
}
