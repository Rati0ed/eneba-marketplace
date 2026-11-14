import { Link, NavLink, useLocation } from 'react-router'
import SearchBar from './SearchBar'
import './Header.css'

// The purple bar at the top of every page.
export default function Header() {
  // On the store page, the search box shows the current ?q= so the user sees what they searched.
  const location = useLocation()
  const urlQuery = location.pathname === '/store' ? (new URLSearchParams(location.search).get('q') ?? '') : ''

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
          <NavLink to="/store">Store</NavLink>
        </nav>
      </div>
    </header>
  )
}
