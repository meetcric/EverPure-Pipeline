import { Route, Routes, useParams } from 'react-router'
import { Layout, NotFoundPage } from './ui/components'
import { PipelinesPage } from './ui/pages/PipelinesPage'
import { RunDetailPage } from './ui/pages/RunDetailPage'
import { RunsPage } from './ui/pages/RunsPage'

export function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<PipelinesPage />} />
        <Route path="pipelines/:id" element={<RunsRoute />} />
        <Route path="runs/:id" element={<RunDetailRoute />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  )
}

// key={id}: moving from one pipeline or run to another mounts a fresh page, so no
// state (form input, live connection) carries over from the previous one.
function RunsRoute() {
  const { id = '' } = useParams()
  return <RunsPage key={id} pipelineId={id} />
}

function RunDetailRoute() {
  const { id = '' } = useParams()
  return <RunDetailPage key={id} runId={id} />
}
