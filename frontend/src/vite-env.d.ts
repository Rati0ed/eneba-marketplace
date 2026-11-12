/// <reference types="vite/client" />

// Tells TypeScript which VITE_* environment variables exist and their types,
// so a typo like import.meta.env.VITE_API_ULR becomes a compile error.
interface ImportMetaEnv {
  readonly VITE_API_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
