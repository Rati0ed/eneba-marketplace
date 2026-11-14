import { Link, NavLink } from 'react-router'
import './Header.css'

// The purple bar at the top of every page.
export default function Header() {
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

        {/* The search bar goes here on Day 3. */}
        <div className="header-search-slot" />

        <nav className="header-nav">
          {/* NavLink adds class "active" when its page is open, so we can highlight it. */}
          <NavLink to="/store">Store</NavLink>
        </nav>
      </div>
    </header>
  )
}
