import { Link } from 'react-router'
import './Footer.css'

// Shown on every page: makes clear this is a demo shop, not a real store (and not Eneba).
export default function Footer() {
  return (
    <footer className="footer">
      <div className="footer-inner">
        <div>
          <strong>
            game<span className="footer-accent">key</span>
          </strong>{' '}
          <span className="muted">is a demo marketplace built for the Eneba internship assignment.</span>
        </div>
        <p className="muted small">
          Payments are simulated (no real money) and game keys are generated, so they can't be redeemed. Not affiliated with
          Eneba or any publisher; product names belong to their owners.
        </p>
        <nav className="footer-links">
          <Link to="/store">Store</Link>
          <Link to="/orders">Orders</Link>
          <a href="https://github.com/Rati0ed/eneba-marketplace" target="_blank" rel="noreferrer">
            Source code
          </a>
        </nav>
      </div>
    </footer>
  )
}
