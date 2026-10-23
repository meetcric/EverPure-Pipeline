// Route "/pipelines/:id": the pipeline's runs with a status filter, "load more" and a create-run form.
import { useState } from 'react'
import { Link, useSearchParams } from 'react-router'
import { useCreateRun, usePipeline, useRunList } from '../../state/hooks'
import { RUN_STATUSES, parseStatus, type RunStatus } from '../../types'
import { EmptyState, ErrorState, FormError, Loading, StatusBadge } from '../components'
import { STATUS_LABELS, formatDateTime, formatDuration, shortSha } from '../format'

export function RunsPage({ pipelineId }: { pipelineId: string }) {
  const pipeline = usePipeline(pipelineId)
  // The filter lives in the URL (?status=failed), so refresh and the back button keep it.
  const [params, setParams] = useSearchParams()
  const status = parseStatus(params.get('status'))

  return (
    <section>
      <Link to="/" className="back">
        ← All pipelines
      </Link>
      <h1>
        {pipeline.data?.name ?? 'Runs'}
        {pipeline.data && <span className="tag">{pipeline.data.product}</span>}
      </h1>

      {pipeline.isError ? (
        <ErrorState error={pipeline.error} onRetry={() => void pipeline.refetch()} />
      ) : (
        <>
          <div className="toolbar">
            <label className="field">
              <span>Status</span>
              <select
                value={status ?? ''}
                onChange={(e) => {
                  const next = parseStatus(e.target.value)
                  setParams(next ? { status: next } : {}, { replace: true })
                }}
              >
                <option value="">All</option>
                {RUN_STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {STATUS_LABELS[s]}
                  </option>
                ))}
              </select>
            </label>
            <CreateRunForm pipelineId={pipelineId} />
          </div>
          <RunList pipelineId={pipelineId} status={status} />
        </>
      )}
    </section>
  )
}

function RunList({ pipelineId, status }: { pipelineId: string; status: RunStatus | undefined }) {
  const runs = useRunList(pipelineId, status)

  if (runs.data === undefined) {
    return runs.isError ? (
      <ErrorState error={runs.error} onRetry={() => void runs.refetch()} />
    ) : (
      <Loading label="Loading runs…" />
    )
  }

  const items = runs.data.pages.flatMap((page) => page.items)
  if (items.length === 0) {
    return (
      <EmptyState>{status ? `No ${STATUS_LABELS[status].toLowerCase()} runs.` : 'No runs yet. Create one above.'}</EmptyState>
    )
  }

  return (
    <>
      <div className="run-table">
        <div className="run-head" aria-hidden="true">
          <span>Status</span>
          <span>Commit</span>
          <span>Triggered by</span>
          <span>Started</span>
          <span>Duration</span>
        </div>
        <ul className="list">
          {items.map((run) => (
            <li key={run.id}>
              <Link to={`/runs/${run.id}`} className="run-row" data-testid="run-row">
                <span className="cell-status">
                  <StatusBadge status={run.status} />
                </span>
                <span className="cell-commit mono">{shortSha(run.commitSha)}</span>
                <span className="cell-by">{run.triggeredBy}</span>
                <span className="cell-started">{formatDateTime(run.startedAt)}</span>
                <span className="cell-duration">
                  {run.durationMs !== undefined ? formatDuration(run.durationMs) : '—'}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      </div>

      <button
        type="button"
        className="btn load-more"
        data-testid="run-load-more"
        disabled={!runs.hasNextPage || runs.isFetchingNextPage}
        onClick={() => void runs.fetchNextPage()}
      >
        {runs.isFetchingNextPage ? 'Loading…' : runs.hasNextPage ? 'Load more' : 'All runs loaded'}
      </button>
      {runs.isFetchNextPageError && <FormError error={runs.error} />}
    </>
  )
}

const SHA_PATTERN = '[0-9a-fA-F]{4,64}' // same rule as the API
const SHA_RE = new RegExp(`^${SHA_PATTERN}$`)

function CreateRunForm({ pipelineId }: { pipelineId: string }) {
  const create = useCreateRun(pipelineId)
  const [triggeredBy, setTriggeredBy] = useState('')
  const [commitSha, setCommitSha] = useState('')
  const ready = triggeredBy.trim() !== '' && SHA_RE.test(commitSha.trim())

  return (
    <form
      className="create-run"
      onSubmit={(e) => {
        e.preventDefault()
        if (!ready) return
        create.mutate(
          { triggeredBy: triggeredBy.trim(), commitSha: commitSha.trim() },
          { onSuccess: () => setCommitSha('') },
        )
      }}
    >
      <label className="field">
        <span>Triggered by</span>
        <input
          value={triggeredBy}
          maxLength={100}
          placeholder="e.g. priya.n"
          onChange={(e) => setTriggeredBy(e.target.value)}
        />
      </label>
      <label className="field grow">
        <span>Commit SHA</span>
        <input
          className="mono"
          value={commitSha}
          maxLength={64}
          pattern={SHA_PATTERN}
          placeholder="e.g. 9f1c2a7"
          onChange={(e) => setCommitSha(e.target.value)}
        />
      </label>
      <button type="submit" className="btn btn-primary" disabled={!ready || create.isPending}>
        {create.isPending ? 'Creating…' : 'Create run'}
      </button>
      {create.isError && <FormError error={create.error} />}
    </form>
  )
}
