import './Pagination.css'

type Props = {
  page: number
  totalPages: number
  onChange: (page: number) => void // a function the parent gives us, called with the new page
}

// Page buttons: « 1 2 [3] 4 5 »  (with "…" when there are many pages).
export default function Pagination({ page, totalPages, onChange }: Props) {
  if (totalPages <= 1) return null // nothing to paginate: render nothing

  return (
    <nav className="pagination" aria-label="Pages">
      <button onClick={() => onChange(page - 1)} disabled={page <= 1} aria-label="Previous page">
        ‹
      </button>

      {pageNumbers(page, totalPages).map((n, i) =>
        n === null ? (
          <span key={`gap-${i}`} className="pagination-gap">
            …
          </span>
        ) : (
          <button
            key={n}
            onClick={() => onChange(n)}
            className={n === page ? 'current' : ''}
            aria-current={n === page ? 'page' : undefined}
          >
            {n}
          </button>
        ),
      )}

      <button onClick={() => onChange(page + 1)} disabled={page >= totalPages} aria-label="Next page">
        ›
      </button>
    </nav>
  )
}

// pageNumbers(6, 12) -> [1, null, 5, 6, 7, null, 12]   (null = "…")
// Always show the first, last, current and its neighbours.
function pageNumbers(page: number, totalPages: number): (number | null)[] {
  const result: (number | null)[] = []
  for (let n = 1; n <= totalPages; n++) {
    const nearCurrent = Math.abs(n - page) <= 1
    if (n === 1 || n === totalPages || nearCurrent) {
      result.push(n)
    } else if (result[result.length - 1] !== null) {
      result.push(null) // only one "…" per gap
    }
  }
  return result
}
