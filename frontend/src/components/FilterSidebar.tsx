import { useState, type FormEvent } from 'react'
import type { FilterOption, ProductFilters } from '../api'
import './FilterSidebar.css'

// The three checkbox groups, in display order. `key` = URL/API parameter name.
const GROUPS = [
  { key: 'platform', label: 'Platform', optionsKey: 'platforms' },
  { key: 'region', label: 'Region', optionsKey: 'regions' },
  { key: 'genre', label: 'Genre', optionsKey: 'genres' },
] as const // "as const" = these values never change, so TypeScript knows the exact strings

type Props = {
  filters: ProductFilters | null // null while the options are loading
  selected: Record<string, string[]> // e.g. { platform: ["Steam"], region: [], genre: [] }
  minPrice: string // current URL values, "" if not set
  maxPrice: string
  onToggle: (key: string, value: string) => void
  onPriceChange: (min: string, max: string) => void
}

// The left column of the store: price range + checkbox lists.
// It doesn't change the URL itself; it tells StorePage what the user clicked (onToggle, ...).
// "Lifting state up": StorePage owns the state, this component only displays it.
export default function FilterSidebar({ filters, selected, minPrice, maxPrice, onToggle, onPriceChange }: Props) {
  if (!filters) {
    return <div className="filters-loading">Loading filters…</div>
  }

  return (
    <div className="filters">
      {/* key={...}: when the URL prices change (e.g. "Clear filters"), React creates a NEW
          PriceFilter, so its inputs reset to the URL values instead of keeping old typing. */}
      <PriceFilter
        key={`${minPrice}-${maxPrice}`}
        initialMin={minPrice}
        initialMax={maxPrice}
        range={filters.price}
        onApply={onPriceChange}
      />

      {GROUPS.map((group) => (
        <CheckboxGroup
          key={group.key}
          label={group.label}
          options={filters[group.optionsKey]}
          selected={selected[group.key] ?? []}
          onToggle={(value) => onToggle(group.key, value)}
        />
      ))}
    </div>
  )
}

type PriceProps = {
  initialMin: string
  initialMax: string
  range: { min: number; max: number }
  onApply: (min: string, max: string) => void
}

// Price inputs keep their OWN state while the user types, and only update the URL on "Apply".
// Otherwise every keystroke ("1", "15", "150") would start a new API request.
function PriceFilter({ initialMin, initialMax, range, onApply }: PriceProps) {
  const [min, setMin] = useState(initialMin)
  const [max, setMax] = useState(initialMax)

  function handleSubmit(event: FormEvent) {
    event.preventDefault() // stop the browser's default "reload the page" on form submit
    onApply(min.trim(), max.trim())
  }

  return (
    <form className="filter-group" onSubmit={handleSubmit}>
      <h3>Price, €</h3>
      <div className="price-inputs">
        <input
          type="number"
          inputMode="numeric"
          min={0}
          step={1}
          placeholder={String(range.min)}
          value={min}
          onChange={(e) => setMin(e.target.value)}
          aria-label="Minimum price in euros"
        />
        <span>–</span>
        <input
          type="number"
          inputMode="numeric"
          min={0}
          step={1}
          placeholder={String(range.max)}
          value={max}
          onChange={(e) => setMax(e.target.value)}
          aria-label="Maximum price in euros"
        />
      </div>
      <button type="submit" className="btn btn-secondary btn-small">
        Apply
      </button>
    </form>
  )
}

type GroupProps = {
  label: string
  options: FilterOption[]
  selected: string[]
  onToggle: (value: string) => void
}

const COLLAPSED_COUNT = 6 // long lists show 6 options until "Show all" is clicked

function CheckboxGroup({ label, options, selected, onToggle }: GroupProps) {
  const [expanded, setExpanded] = useState(false)
  const visible = expanded ? options : options.slice(0, COLLAPSED_COUNT)

  return (
    <fieldset className="filter-group">
      <legend>{label}</legend>
      {visible.map((option) => (
        <label key={option.value} className="checkbox-row">
          <input
            type="checkbox"
            checked={selected.includes(option.value)}
            onChange={() => onToggle(option.value)}
          />
          <span className="checkbox-label">{option.value}</span>
          <span className="checkbox-count">{option.count}</span>
        </label>
      ))}
      {options.length > COLLAPSED_COUNT && (
        <button type="button" className="link-button" onClick={() => setExpanded(!expanded)}>
          {expanded ? 'Show less' : `Show all (${options.length})`}
        </button>
      )}
    </fieldset>
  )
}
