import { useEffect, useState } from 'react'

// usePolling calls loadData once immediately and then every intervalMs
// milliseconds. It returns the latest result and the latest error message.
export function usePolling(loadData, intervalMs) {
  const [data, setData] = useState(null)
  const [error, setError] = useState(null)

  useEffect(() => {
    let active = true // becomes false when the component leaves the page

    async function load() {
      try {
        const result = await loadData()
        if (active) {
          setData(result)
          setError(null)
        }
      } catch (err) {
        if (active) setError(err.message)
      }
    }

    load()
    const timer = setInterval(load, intervalMs)

    // Cleanup: React runs this function when the component leaves the page.
    return () => {
      active = false
      clearInterval(timer)
    }
  }, [loadData, intervalMs])

  return { data, error }
}
