import { isSafeApiPath, type HomeBlock } from '../api'
import { useApiData } from '../hooks/useApiData'
import BannerCarousel from '../components/BannerCarousel'
import ProductRow from '../components/ProductRow'
import './HomePage.css'

// The homepage has NO content of its own. It asks the API which blocks to show
// (/api/home/blocks), and draws one component per block, in the order given.
// Each block then loads its own content from its own endpoint.
// Change the homepage_blocks table -> the homepage changes, without a new frontend deploy.
export default function HomePage() {
  const blocks = useApiData<{ items: HomeBlock[] }>('/api/home/blocks')

  if (blocks.error) {
    return (
      <main className="page">
        <div className="empty-state">
          <h1>Something went wrong</h1>
          <p className="muted">{blocks.error.message}</p>
          <button className="btn btn-primary" onClick={blocks.retry}>
            Try again
          </button>
        </div>
      </main>
    )
  }

  return (
    <main className="page home">
      {!blocks.data && <div className="banner-skeleton skeleton" />}

      {blocks.data?.items.map((block) => {
        if (!isSafeApiPath(block.endpoint)) return null // never follow a link outside our API

        // A "switch" on the block type picks the component. Unknown types are skipped,
        // so the backend can add a new block type before the frontend knows how to draw it.
        switch (block.type) {
          case 'banner':
            return <BannerCarousel key={block.id} endpoint={block.endpoint} />
          case 'product_row':
            return <ProductRow key={block.id} title={block.title} endpoint={block.endpoint} linkUrl={block.link_url} />
          default:
            return null
        }
      })}
    </main>
  )
}
