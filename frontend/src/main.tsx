import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router'
import './index.css'
import App from './App.tsx'
import { AuthProvider } from './context/AuthContext'
import { CartProvider } from './context/CartContext'

// Find <div id="root"> in index.html and let React control everything inside it.
// The "!" tells TypeScript "trust me, this element exists".
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    {/* BrowserRouter makes the URL part of the app's state: it decides which page to show
        and lets pages read/change the URL without reloading the whole site. */}
    <BrowserRouter>
      {/* Providers make shared state (login, cart) available to every component inside them. */}
      <AuthProvider>
        <CartProvider>
          <App />
        </CartProvider>
      </AuthProvider>
    </BrowserRouter>
  </StrictMode>,
)
