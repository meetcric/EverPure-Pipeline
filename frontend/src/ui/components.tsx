// Shared presentation pieces used by more than one page.
import type { ReactNode } from 'react'
import { Link, Outlet } from 'react-router'
import type { RunStatus } from '../types'
import { STATUS_LABELS, errorMessage } from './format'

export function Layout() {
  return (
    <>
      <header className="topbar">
        <Link to="/" className="brand" data-testid="nav-home">
          Pipeline Pulse
        </Link>
      </header>
      <main className="container">
        <Outlet />
      </main>
    </>
  )
}

export function NotFoundPage() {
  return (
    <section>
      <h1>Page not found</h1>
      <Link to="/">Back to pipelines</Link>
    </section>
  )
}

export function StatusBadge({ status, testId }: { status: RunStatus; testId?: string }) {
  return (
    <span className={`badge badge-${status}`} data-testid={testId}>
      {STATUS_LABELS[status]}
    </span>
  )
}

// The three states every data-fetching view needs, so they look the same everywhere.

export function Loading({ label = 'Loading…' }: { label?: string }) {
  return (
    <p className="state" role="status">
      {label}
    </p>
  )
}

export function ErrorState({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  return (
    <div className="state state-error" role="alert">
      <p>{errorMessage(error)}</p>
      {onRetry && (
        <button type="button" className="btn" onClick={onRetry}>
          Try again
        </button>
      )}
    </div>
  )
}

export function EmptyState({ children }: { children: ReactNode }) {
  return <p className="state">{children}</p>
}

export function FormError({ error }: { error: unknown }) {
  return (
    <p className="form-error" role="alert">
      {errorMessage(error)}
    </p>
  )
}
