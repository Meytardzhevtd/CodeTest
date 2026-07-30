import { useEffect, useState } from 'react'
import { ApiError, submitApi } from '../api'
import type { Submission, SubmissionHistoryItem } from '../types'

const STATUS_LABELS: Record<string, string> = {
  pending: 'В очереди',
  running: 'Проверяется',
  OK: 'OK',
  WA: 'WA',
  RE: 'RE',
  CE: 'CE',
  TL: 'TL',
  ML: 'ML',
  ERROR: 'ERROR',
}

function statusTone(status: string): 'success' | 'fail' | 'pending' {
  if (status === 'OK') return 'success'
  if (status === 'pending' || status === 'running') return 'pending'
  return 'fail'
}

export function SubmissionHistoryModal({ taskId, onClose }: { taskId: string; onClose: () => void }) {
  const [items, setItems] = useState<SubmissionHistoryItem[] | null>(null)
  const [error, setError] = useState('')
  const [selected, setSelected] = useState<Submission | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  useEffect(() => {
    let cancelled = false

    submitApi
      .history(taskId)
      .then((res) => {
        if (!cancelled) setItems(res.submissions)
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof ApiError ? err.message : 'Не удалось загрузить историю посылок')
        }
      })

    return () => {
      cancelled = true
    }
  }, [taskId])

  const openSubmission = async (id: string) => {
    setError('')
    setDetailLoading(true)
    try {
      const res = await submitApi.get(id)
      setSelected(res.submission)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Не удалось загрузить посылку')
    } finally {
      setDetailLoading(false)
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-card" onClick={(event) => event.stopPropagation()}>
        <div className="modal-header">
          {selected ? (
            <button type="button" className="back-link" onClick={() => setSelected(null)}>
              ← К списку
            </button>
          ) : (
            <h2>История посылок</h2>
          )}
          <button type="button" className="modal-close" onClick={onClose} aria-label="Закрыть">
            ×
          </button>
        </div>

        {error ? <div className="alert alert-error">{error}</div> : null}

        {selected ? (
          <div className="submission-detail">
            <div className="submission-detail-meta">
              <span className={`badge badge-status-${statusTone(selected.status)}`}>
                {STATUS_LABELS[selected.status] ?? selected.status}
              </span>
              <span className="submission-detail-lang">{selected.language}</span>
            </div>
            <pre className="code-view">{selected.code}</pre>
            {selected.error ? <div className="verdict-detail">{selected.error}</div> : null}
          </div>
        ) : detailLoading || items === null ? (
          <p className="submission-history-empty">Загрузка...</p>
        ) : items.length === 0 ? (
          <p className="submission-history-empty">Посылок по этой задаче пока нет.</p>
        ) : (
          <ul className="submission-history-list">
            {items.map((item) => (
              <li key={item.id}>
                <button
                  type="button"
                  className="submission-history-row"
                  onClick={() => openSubmission(item.id)}
                >
                  <span className="submission-history-number">#{item.number}</span>
                  <span className={`badge badge-status-${statusTone(item.status)}`}>
                    {STATUS_LABELS[item.status] ?? item.status}
                  </span>
                  <span className="submission-history-lang">{item.language}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
