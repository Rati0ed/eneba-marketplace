import { useEffect, useState } from 'react'
import { Link } from 'react-router'
import { imageUrl, type Banner } from '../api'
import { useApiData } from '../hooks/useApiData'

const AUTO_ADVANCE_MS = 6000

// Homepage carousel: one slide visible at a time, changes every 6 s, with arrows and dots.
// Its slides come from `endpoint` (e.g. /api/home/banners), given by the homepage block.
export default function BannerCarousel({ endpoint }: { endpoint: string }) {
  const banners = useApiData<{ items: Banner[] }>(endpoint)
  const [index, setIndex] = useState(0)
  const [paused, setPaused] = useState(false) // mouse over the banner = stop auto-advancing

  const slides = banners.data?.items ?? []
  const count = slides.length

  // Auto-advance. setInterval runs every 6 s; the cleanup stops it when the component
  // disappears or when `paused`/`count` change (the effect then starts a fresh interval).
  useEffect(() => {
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (paused || count < 2 || reducedMotion) return
    const timer = setInterval(() => setIndex((i) => (i + 1) % count), AUTO_ADVANCE_MS)
    return () => clearInterval(timer)
  }, [paused, count])

  if (banners.error) return null // a broken banner shouldn't break the whole homepage
  if (!banners.data) return <div className="banner-skeleton skeleton" />
  if (count === 0) return null

  const current = index % count // safety: if the slide list got shorter, stay in range

  return (
    <section
      className="carousel"
      aria-roledescription="carousel"
      aria-label="Featured offers"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      onFocus={() => setPaused(true)} // keyboard users too: don't move the slide away while they're on it
      onBlur={() => setPaused(false)}
    >
      {slides.map((banner, i) => (
        <div
          key={banner.id}
          className={i === current ? 'carousel-slide active' : 'carousel-slide'}
          aria-hidden={i !== current} // screen readers only read the visible slide
        >
          <img src={imageUrl(banner.image_url)} alt="" />
          <div className="carousel-text">
            <h2>{banner.title}</h2>
            <p>{banner.subtitle}</p>
            {/* tabIndex -1 on hidden slides: Tab must not land on a button you can't see */}
            {/* Only links inside our own site (same rule as ProductRow) */}
            {banner.link_url.startsWith('/') && !banner.link_url.startsWith('//') && (
              <Link to={banner.link_url} className="btn btn-primary" tabIndex={i === current ? 0 : -1}>
                View offer
              </Link>
            )}
          </div>
        </div>
      ))}

      {count > 1 && (
        <>
          <button
            className="carousel-arrow prev"
            onClick={() => setIndex((current - 1 + count) % count)}
            aria-label="Previous slide"
          >
            ‹
          </button>
          <button className="carousel-arrow next" onClick={() => setIndex((current + 1) % count)} aria-label="Next slide">
            ›
          </button>
          <div className="carousel-dots">
            {slides.map((banner, i) => (
              <button
                key={banner.id}
                className={i === current ? 'active' : ''}
                onClick={() => setIndex(i)}
                aria-label={`Show slide ${i + 1}: ${banner.title}`}
                aria-current={i === current}
              />
            ))}
          </div>
        </>
      )}
    </section>
  )
}
