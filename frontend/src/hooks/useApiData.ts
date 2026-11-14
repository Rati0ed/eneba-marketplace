import { useEffect, useState } from 'react'
import { ApiError, apiRequest } from '../api'

// useApiData is a CUSTOM HOOK: a function that uses React's hooks (useState, useEffect)
// so several components can share the same logic. Its name must start with "use".
//
// It GETs `path` from our API and gives back { data, error, loading, retry }:
//
//   const { data, error, loading } = useApiData<ProductDetail>(`/api/products/${slug}`)
//
// - When `path` changes, it fetches again (and cancels the old request).
// - path = null means "don't fetch yet" (e.g. we have no token).
// - `loading` isn't stored: it's true when the stored answer is for a different request
//   than the current one. (Same idea as on the store page.)
export function useApiData<T>(path: string | null, token?: string | null) {
  const [retryCount, setRetryCount] = useState(0)
  const requestKey = path === null ? null : `${path}|${token ?? ''}|${retryCount}`

  const [loaded, setLoaded] = useState<{ key: string | null; data: T | null; error: ApiError | null }>({
    key: null,
    data: null,
    error: null,
  })

  useEffect(() => {
    if (path === null) return
    const controller = new AbortController()

    apiRequest<T>(path, { token, signal: controller.signal })
      .then((data) => setLoaded({ key: requestKey, data, error: null }))
      .catch((err) => {
        if (controller.signal.aborted) return // cancelled because path changed: ignore
        const apiError = err instanceof ApiError ? err : new ApiError(String(err), 0)
        setLoaded({ key: requestKey, data: null, error: apiError })
      })

    return () => controller.abort()
  }, [path, token, requestKey])

  const loading = requestKey !== null && loaded.key !== requestKey
  return {
    // While loading a NEW path, hide the old data so product A's info never shows on product B's page.
    data: loading ? null : loaded.data,
    error: loading ? null : loaded.error,
    loading,
    retry: () => setRetryCount((n) => n + 1),
  }
}
