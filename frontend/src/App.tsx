import { Route, Routes } from 'react-router'
import Header from './components/Header'
import Footer from './components/Footer'
import HomePage from './pages/HomePage'
import StorePage from './pages/StorePage'
import ProductPage from './pages/ProductPage'
import NotFoundPage from './pages/NotFoundPage'
import AuthPage from './pages/AuthPage'
import CartPage from './pages/CartPage'
import CheckoutPage from './pages/CheckoutPage'
import OrdersPage from './pages/OrdersPage'
import OrderPage from './pages/OrderPage'
import RequireAuth from './components/RequireAuth'

// App = the frame shared by every page (header) + which page to show for the current URL.
export default function App() {
  return (
    <>
      <Header />
      {/* <Routes> picks the FIRST <Route> whose path matches the URL. */}
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/store" element={<StorePage />} />
        {/* ":slug" is a URL parameter: /product/elden-ring-pc-steam-global -> slug = "elden-ring-..." */}
        <Route path="/product/:slug" element={<ProductPage />} />
        <Route path="/login" element={<AuthPage mode="login" />} />
        <Route path="/register" element={<AuthPage mode="register" />} />
        <Route path="/cart" element={<CartPage />} />
        {/* Members only: <RequireAuth> sends guests to /login?next=... and brings them back. */}
        <Route path="/checkout" element={<RequireAuth><CheckoutPage /></RequireAuth>} />
        <Route path="/orders" element={<RequireAuth><OrdersPage /></RequireAuth>} />
        <Route path="/orders/:id" element={<RequireAuth><OrderPage /></RequireAuth>} />
        {/* "*" matches anything else */}
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
      <Footer />
    </>
  )
}
