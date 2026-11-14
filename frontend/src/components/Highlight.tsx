// Highlight shows `text` with the words of `query` marked in yellow:
// text "Elden Ring (PC)", query "elden" -> <mark>Elden</mark> Ring (PC)
// Words with typos aren't highlighted (they don't appear in the title as typed); that's fine.
export default function Highlight({ text, query }: { text: string; query: string }) {
  const words = query
    .trim()
    .split(/\s+/) // split on any whitespace
    .filter((word) => word.length >= 2)
    // The user's text goes into a regular expression, where . * ( ) etc. have special meaning.
    // Escaping them makes "c++" or "(pc" search for those characters instead of breaking the regex.
    .map((word) => word.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))

  if (words.length === 0) return <>{text}</>

  // "gi" = global (all matches) + case-insensitive. The ( ) "capture group" makes split()
  // KEEP the matched parts: "Elden Ring".split(/(elden)/i) -> ["", "Elden", " Ring"].
  // So every odd position (1, 3, 5...) is a match.
  const parts = text.split(new RegExp(`(${words.join('|')})`, 'gi'))

  return (
    <>
      {parts.map((part, i) => (i % 2 === 1 ? <mark key={i}>{part}</mark> : <span key={i}>{part}</span>))}
    </>
  )
}
