// Route "/": list pipelines, and create / rename / delete them inline.
import { useState } from 'react'
import { Link } from 'react-router'
import { useCreatePipeline, useDeletePipeline, usePipelines, useRenamePipeline } from '../../state/hooks'
import { PRODUCTS, type Pipeline, type Product } from '../../types'
import { EmptyState, ErrorState, FormError, Loading } from '../components'

export function PipelinesPage() {
  return (
    <section>
      <h1>Pipelines</h1>
      <CreatePipelineForm />
      <PipelineList />
    </section>
  )
}

function PipelineList() {
  const pipelines = usePipelines()

  // No data yet: either the first load or its error. A failed background refresh keeps showing the list.
  if (pipelines.data === undefined) {
    return pipelines.isError ? (
      <ErrorState error={pipelines.error} onRetry={() => void pipelines.refetch()} />
    ) : (
      <Loading label="Loading pipelines…" />
    )
  }
  if (pipelines.data.length === 0) {
    return <EmptyState>No pipelines yet. Create the first one above.</EmptyState>
  }
  return (
    <ul className="list">
      {pipelines.data.map((p) => (
        <PipelineRow key={p.id} pipeline={p} />
      ))}
    </ul>
  )
}

function CreatePipelineForm() {
  const create = useCreatePipeline()
  const [name, setName] = useState('')
  const [product, setProduct] = useState<Product>('flasharray')
  const trimmed = name.trim()

  return (
    <form
      className="toolbar"
      data-testid="pipeline-create"
      onSubmit={(e) => {
        e.preventDefault()
        if (!trimmed) return
        create.mutate({ name: trimmed, product }, { onSuccess: () => setName('') })
      }}
    >
      <label className="field grow">
        <span>Name</span>
        <input
          value={name}
          maxLength={100}
          placeholder="e.g. FlashArray Nightly"
          onChange={(e) => setName(e.target.value)}
        />
      </label>
      <label className="field">
        <span>Product</span>
        <select value={product} onChange={(e) => setProduct(e.target.value as Product)}>
          {PRODUCTS.map((p) => (
            <option key={p} value={p}>
              {p}
            </option>
          ))}
        </select>
      </label>
      <button type="submit" className="btn btn-primary" disabled={!trimmed || create.isPending}>
        {create.isPending ? 'Creating…' : 'Create pipeline'}
      </button>
      {create.isError && <FormError error={create.error} />}
    </form>
  )
}

function PipelineRow({ pipeline }: { pipeline: Pipeline }) {
  const [mode, setMode] = useState<'view' | 'rename' | 'confirm-delete'>('view')
  const rename = useRenamePipeline()
  const remove = useDeletePipeline()
  const [name, setName] = useState(pipeline.name)

  if (mode === 'rename') {
    const trimmed = name.trim()
    const close = () => {
      setName(pipeline.name)
      setMode('view')
    }
    return (
      <li className="row" data-testid="pipeline-row">
        <form
          className="row-edit"
          onSubmit={(e) => {
            e.preventDefault()
            if (trimmed === pipeline.name) return close()
            rename.mutate({ id: pipeline.id, name: trimmed }, { onSuccess: () => setMode('view') })
          }}
        >
          <input
            aria-label="Pipeline name"
            value={name}
            maxLength={100}
            autoFocus
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => e.key === 'Escape' && close()}
          />
          <button type="submit" className="btn btn-primary" disabled={!trimmed || rename.isPending}>
            {rename.isPending ? 'Saving…' : 'Save'}
          </button>
          <button type="button" className="btn" onClick={close}>
            Cancel
          </button>
          {rename.isError && <FormError error={rename.error} />}
        </form>
      </li>
    )
  }

  return (
    <li className="row" data-testid="pipeline-row">
      <div className="row-main">
        <Link to={`/pipelines/${pipeline.id}`} className="row-title">
          {pipeline.name}
        </Link>
        <span className="tag">{pipeline.product}</span>
      </div>
      {/* An inline confirm instead of window.confirm(): it doesn't block the page, and automated tests can click it. */}
      {mode === 'confirm-delete' ? (
        <div className="row-actions">
          <span>Delete with all its runs?</span>
          <button
            type="button"
            className="btn btn-danger"
            disabled={remove.isPending}
            onClick={() => remove.mutate(pipeline.id)}
          >
            {remove.isPending ? 'Deleting…' : 'Yes, delete'}
          </button>
          <button type="button" className="btn" onClick={() => setMode('view')}>
            Cancel
          </button>
          {remove.isError && <FormError error={remove.error} />}
        </div>
      ) : (
        <div className="row-actions">
          <button type="button" className="btn" onClick={() => setMode('rename')}>
            Rename
          </button>
          <button type="button" className="btn btn-danger" onClick={() => setMode('confirm-delete')}>
            Delete
          </button>
        </div>
      )}
    </li>
  )
}
