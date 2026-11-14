import { Route, Routes } from 'react-router'
import Header from './components/Header'
import HomePage from './pages/HomePage'
import StorePage from './pages/StorePage'
import ProductPage from './pages/ProductPage'
import NotFoundPage from './pages/NotFoundPage'

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
        {/* "*" matches anything else */}
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </>
  )
}
