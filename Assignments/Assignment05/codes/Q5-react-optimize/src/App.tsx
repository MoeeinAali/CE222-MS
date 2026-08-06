import { useMemo, useState } from 'react'

type Status = 'healthy' | 'degraded' | 'down'

type Service = {
  id: string
  name: string
  status: Status
  latencyMs: number
  version: string
}

const SEED: Service[] = [
  { id: 'gw', name: 'api-gateway', status: 'healthy', latencyMs: 12, version: '2.4.1' },
  { id: 'ord', name: 'orders', status: 'healthy', latencyMs: 38, version: '1.9.0' },
  { id: 'pay', name: 'payments', status: 'degraded', latencyMs: 214, version: '3.0.2' },
  { id: 'inv', name: 'inventory', status: 'healthy', latencyMs: 47, version: '1.2.7' },
  { id: 'ntf', name: 'notifications', status: 'down', latencyMs: 0, version: '0.8.5' },
  { id: 'rpt', name: 'reporting', status: 'healthy', latencyMs: 91, version: '1.0.0' },
]

const ORDER: Record<Status, number> = { down: 0, degraded: 1, healthy: 2 }

export default function App() {
  const [services, setServices] = useState<Service[]>(SEED)
  const [query, setQuery] = useState('')
  const [onlyProblems, setOnlyProblems] = useState(false)

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase()
    return services
      .filter((s) => (q ? s.name.includes(q) : true))
      .filter((s) => (onlyProblems ? s.status !== 'healthy' : true))
      .sort((a, b) => ORDER[a.status] - ORDER[b.status] || a.name.localeCompare(b.name))
  }, [services, query, onlyProblems])

  const summary = useMemo(() => {
    const healthy = services.filter((s) => s.status === 'healthy').length
    const avg = Math.round(
      services.filter((s) => s.latencyMs > 0).reduce((t, s) => t + s.latencyMs, 0) /
        Math.max(1, services.filter((s) => s.latencyMs > 0).length),
    )
    return { healthy, total: services.length, avg }
  }, [services])

  function refresh() {
    setServices((prev) =>
      prev.map((s) => ({
        ...s,
        latencyMs: s.status === 'down' ? 0 : Math.max(5, Math.round(s.latencyMs * (0.7 + Math.random() * 0.6))),
      })),
    )
  }

  return (
    <main className="board">
      <header>
        <h1>Service Board</h1>
        <p className="sub">
          {summary.healthy}/{summary.total} healthy · avg latency {summary.avg} ms
        </p>
      </header>

      <div className="controls">
        <input
          value={query}
          placeholder="filter by name…"
          onChange={(e) => setQuery(e.target.value)}
          aria-label="filter services"
        />
        <label>
          <input type="checkbox" checked={onlyProblems} onChange={(e) => setOnlyProblems(e.target.checked)} />
          problems only
        </label>
        <button onClick={refresh}>refresh</button>
      </div>

      <ul className="list">
        {visible.map((s) => (
          <li key={s.id} className={`item ${s.status}`}>
            <span className="dot" aria-hidden />
            <span className="name">{s.name}</span>
            <span className="version">v{s.version}</span>
            <span className="latency">{s.status === 'down' ? '—' : `${s.latencyMs} ms`}</span>
            <span className="status">{s.status}</span>
          </li>
        ))}
        {visible.length === 0 && <li className="empty">no service matches</li>}
      </ul>
    </main>
  )
}
