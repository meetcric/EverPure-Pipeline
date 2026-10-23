// Route "/runs/:id": run facts, a live status badge (SSE), status buttons and notes.
import { useState } from 'react'
import { Link } from 'react-router'
import {
  useAddNote,
  useDeleteNote,
  useNow,
  useRun,
  useRunEvents,
  useUpdateRunStatus,
  type LiveState,
} from '../../state/hooks'
import type { Note, Run } from '../../types'
import { EmptyState, ErrorState, FormError, Loading, StatusBadge } from '../components'
import { STATUS_ACTIONS, formatDateTime, formatDuration, isFinished } from '../format'

const LIVE_LABELS: Record<LiveState, string> = {
  connecting: 'Connecting…',
  live: 'Live',
  reconnecting: 'Reconnecting…',
  closed: 'Not live',
}

export function RunDetailPage({ runId }: { runId: string }) {
  const run = useRun(runId)
  // Subscribes to the run's status stream for as long as this page is open.
  const live = useRunEvents(runId)

  if (run.data === undefined) {
    return run.isError ? (
      <ErrorState error={run.error} onRetry={() => void run.refetch()} />
    ) : (
      <Loading label="Loading run…" />
    )
  }
  const r = run.data

  return (
    <article>
      <Link to={`/pipelines/${r.pipelineId}`} className="back">
        ← Pipeline runs
      </Link>
      <header className="run-header">
        <h1>
          Run <span className="mono">{r.id.length > 12 ? r.id.slice(0, 8) : r.id}</span>
        </h1>
        <StatusBadge status={r.status} testId="run-status-badge" />
        <span className={`live live-${live}`}>{LIVE_LABELS[live]}</span>
      </header>

      <RunFacts run={r} />
      <StatusControls run={r} />
      <Notes runId={r.id} notes={r.notes} />
    </article>
  )
}

function RunFacts({ run }: { run: Run }) {
  const running = run.status === 'running'
  // Ticks once a second only while the run is running; the timer stops on unmount.
  const now = useNow(running)

  let duration = '—'
  if (run.durationMs !== undefined) duration = formatDuration(run.durationMs)
  else if (running) duration = `${formatDuration(now - Date.parse(run.startedAt))} so far`

  return (
    <dl className="facts">
      <div>
        <dt>Started</dt>
        <dd>{formatDateTime(run.startedAt)}</dd>
      </div>
      <div>
        <dt>Finished</dt>
        <dd>{run.finishedAt ? formatDateTime(run.finishedAt) : '—'}</dd>
      </div>
      <div>
        <dt>Duration</dt>
        <dd>{duration}</dd>
      </div>
      <div>
        <dt>Triggered by</dt>
        <dd>{run.triggeredBy}</dd>
      </div>
      <div className="wide">
        <dt>Commit</dt>
        <dd className="mono break">{run.commitSha}</dd>
      </div>
    </dl>
  )
}

function StatusControls({ run }: { run: Run }) {
  const update = useUpdateRunStatus(run.id)

  return (
    <section className="panel">
      <h2>Status</h2>
      <div className="actions">
        {STATUS_ACTIONS.map((action) => (
          <button
            key={action.to}
            type="button"
            className="btn"
            disabled={update.isPending || !action.from.includes(run.status)}
            onClick={() => update.mutate(action.to)}
          >
            {action.label}
          </button>
        ))}
      </div>
      {isFinished(run.status) && <p className="muted">This run has finished; its status can't change.</p>}
      {update.isError && <FormError error={update.error} />}
    </section>
  )
}

function Notes({ runId, notes }: { runId: string; notes: Note[] }) {
  const add = useAddNote(runId)
  const remove = useDeleteNote(runId)
  const [author, setAuthor] = useState('')
  const [body, setBody] = useState('')
  const ready = author.trim() !== '' && body.trim() !== ''

  return (
    <section className="panel">
      <h2>Notes</h2>
      {notes.length === 0 ? (
        <EmptyState>No notes yet.</EmptyState>
      ) : (
        <ul className="notes">
          {notes.map((note) => (
            <li key={note.id} className="note" data-testid="note-row">
              <div className="note-meta">
                <strong>{note.author}</strong>
                <span className="muted">{formatDateTime(note.createdAt)}</span>
                <button
                  type="button"
                  className="btn btn-small"
                  disabled={remove.isPending && remove.variables === note.id}
                  onClick={() => remove.mutate(note.id)}
                >
                  Delete
                </button>
              </div>
              <p className="note-body">{note.body}</p>
            </li>
          ))}
        </ul>
      )}
      {remove.isError && <FormError error={remove.error} />}

      <form
        className="note-form"
        data-testid="note-add"
        onSubmit={(e) => {
          e.preventDefault()
          if (!ready) return
          // Keep the author for the next note; clear only the text.
          add.mutate({ author: author.trim(), body: body.trim() }, { onSuccess: () => setBody('') })
        }}
      >
        <label className="field">
          <span>Author</span>
          <input value={author} maxLength={100} placeholder="e.g. priya.n" onChange={(e) => setAuthor(e.target.value)} />
        </label>
        <label className="field">
          <span>Note</span>
          <textarea
            value={body}
            maxLength={5000}
            rows={3}
            placeholder="What happened in this run?"
            onChange={(e) => setBody(e.target.value)}
          />
        </label>
        <button type="submit" className="btn btn-primary" disabled={!ready || add.isPending}>
          {add.isPending ? 'Adding…' : 'Add note'}
        </button>
        {add.isError && <FormError error={add.error} />}
      </form>
    </section>
  )
}
