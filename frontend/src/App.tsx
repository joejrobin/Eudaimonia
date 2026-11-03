import { useEffect, useState } from 'react'
import './App.css'

type NodeInfo = {
  name: string
  version: string
  livekit_url: string
  time: string
}

type Status =
  | { state: 'loading' }
  | { state: 'ok'; node: NodeInfo }
  | { state: 'error'; message: string }

function App() {
  const [status, setStatus] = useState<Status>({ state: 'loading' })

  useEffect(() => {
    const controller = new AbortController()
    fetch('/api/v1/node', { signal: controller.signal })
      .then(async (res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        setStatus({ state: 'ok', node: (await res.json()) as NodeInfo })
      })
      .catch((err: unknown) => {
        if (controller.signal.aborted) return
        setStatus({
          state: 'error',
          message: err instanceof Error ? err.message : String(err),
        })
      })
    return () => controller.abort()
  }, [])

  return (
    <main className="app">
      <h1>Eudaimonia</h1>
      <p className="tagline">
        Decentralized communities. Your keys, your node.
      </p>
      <section className="node">
        <h2>Node</h2>
        {status.state === 'loading' && <p>Connecting…</p>}
        {status.state === 'error' && (
          <p className="error">Backend unreachable: {status.message}</p>
        )}
        {status.state === 'ok' && (
          <dl>
            <dt>Name</dt>
            <dd>{status.node.name}</dd>
            <dt>Version</dt>
            <dd>{status.node.version}</dd>
            <dt>Voice (LiveKit)</dt>
            <dd>{status.node.livekit_url}</dd>
            <dt>Node time</dt>
            <dd>{status.node.time}</dd>
          </dl>
        )}
      </section>
    </main>
  )
}

export default App
