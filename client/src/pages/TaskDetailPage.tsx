import { useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { ApiError, submitApi, tasksApi } from '../api'
import type { Submission, Task } from '../types'
import { CodeEditor } from '../components/CodeEditor'
import { DifficultyBadge } from '../components/DifficultyBadge'
import { SubmissionHistoryModal } from '../components/SubmissionHistoryModal'
import { TestsUploadCard } from '../components/TestsUploadCard'
import { useAuth } from '../context/AuthContext'
import { useRouter } from '../router'

type SubmitState = 'idle' | 'checking' | 'done'

const LANGUAGES = [
  { value: 'python', label: 'Python' },
  { value: 'go', label: 'Go' },
  { value: 'cpp', label: 'C++' },
  { value: 'javascript', label: 'JavaScript' },
]

const POLL_INTERVAL_MS = 1000

const STATUS_LABELS: Record<string, string> = {
  OK: 'Решение принято',
  WA: 'Неверный ответ',
  RE: 'Ошибка выполнения',
  CE: 'Ошибка компиляции',
  TL: 'Превышено время выполнения',
  ML: 'Превышен лимит памяти',
  ERROR: 'Не удалось проверить решение, попробуйте ещё раз',
}

export function TaskDetailPage({ slug }: { slug: string }) {
  const [task, setTask] = useState<Task | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [code, setCode] = useState('')
  const [language, setLanguage] = useState('python')
  const [submitState, setSubmitState] = useState<SubmitState>('idle')
  const [submitError, setSubmitError] = useState('')
  const [verdict, setVerdict] = useState<Submission | null>(null)
  const [showHistory, setShowHistory] = useState(false)
  const pollHandle = useRef<number | null>(null)
  const { navigate } = useRouter()
  const { user } = useAuth()

  useEffect(() => {
    let cancelled = false

    tasksApi
      .getBySlug(slug)
      .then((res) => {
        if (!cancelled) setTask(res)
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof ApiError ? err.message : 'Не удалось загрузить задачу')
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [slug])

  useEffect(() => {
    return () => {
      if (pollHandle.current !== null) {
        window.clearInterval(pollHandle.current)
      }
    }
  }, [])

  const stopPolling = () => {
    if (pollHandle.current !== null) {
      window.clearInterval(pollHandle.current)
      pollHandle.current = null
    }
  }

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault()
    if (!code.trim() || submitState === 'checking' || !task) return

    stopPolling()
    setSubmitError('')
    setVerdict(null)
    setSubmitState('checking')

    try {
      const created = await submitApi.create({ task_id: task.id, code, language })

      pollHandle.current = window.setInterval(async () => {
        try {
          const res = await submitApi.get(created.id)
          if (res.submission.status === 'pending' || res.submission.status === 'running') {
            return
          }
          stopPolling()
          setVerdict(res.submission)
          setSubmitState('done')
        } catch (err) {
          stopPolling()
          setSubmitError(err instanceof ApiError ? err.message : 'Не удалось получить результат проверки')
          setSubmitState('idle')
        }
      }, POLL_INTERVAL_MS)
    } catch (err) {
      setSubmitError(err instanceof ApiError ? err.message : 'Не удалось отправить решение')
      setSubmitState('idle')
    }
  }

  if (loading) {
    return (
      <div className="page">
        <div className="task-detail-grid">
          <div className="card skeleton-card" aria-hidden="true">
            <div className="skeleton-line skeleton-badge" />
            <div className="skeleton-line skeleton-title" />
            <div className="skeleton-line" />
            <div className="skeleton-line" />
            <div className="skeleton-line skeleton-short" />
          </div>
          <div className="card skeleton-card" aria-hidden="true">
            <div className="skeleton-line skeleton-title" />
            <div className="skeleton-line skeleton-code" />
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="page">
        <div className="alert alert-error">{error}</div>
        <button type="button" className="back-link" onClick={() => navigate('/tasks')}>
          ← Все задачи
        </button>
      </div>
    )
  }

  if (!task) {
    return null
  }

  return (
    <div className="page">
      <button type="button" className="back-link" onClick={() => navigate('/tasks')}>
        ← Все задачи
      </button>

      <div className="task-detail-grid">
        <section className="card task-statement">
          <div className="task-detail-header">
            <DifficultyBadge difficulty={task.difficulty} />
            <span className="task-slug">{task.slug}</span>
          </div>
          <h1>{task.title}</h1>
          {task.tags.length > 0 ? (
            <div className="task-tags">
              {task.tags.map((tag) => (
                <span className="task-tag" key={tag}>
                  {tag}
                </span>
              ))}
            </div>
          ) : null}
          <p className="task-statement-text">{task.statement}</p>
          <div className="task-limits">
            <div>
              <span>Время</span>
              <strong>{task.time_limit_ms} мс</strong>
            </div>
            <div>
              <span>Память</span>
              <strong>{task.memory_limit_mb} МБ</strong>
            </div>
          </div>
        </section>

        <form className="card task-submit" onSubmit={handleSubmit}>
          <div className="task-submit-header">
            <h2>Решение</h2>
            <div className="task-submit-header-actions">
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => setShowHistory(true)}>
                История посылок
              </button>
              <select value={language} onChange={(event) => setLanguage(event.target.value)}>
                {LANGUAGES.map((lang) => (
                  <option key={lang.value} value={lang.value}>
                    {lang.label}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <CodeEditor value={code} onChange={setCode} language={language} />

          <div className="task-submit-footer">
            <button
              type="submit"
              className="btn btn-primary"
              disabled={submitState === 'checking' || !code.trim()}
            >
              {submitState === 'checking' ? 'Проверка...' : 'Отправить решение'}
            </button>
          </div>

          {submitError ? <div className="alert alert-error">{submitError}</div> : null}

          {submitState === 'done' && verdict ? (
            <div className={`alert ${verdict.status === 'OK' ? 'alert-success' : 'alert-error'}`}>
              {STATUS_LABELS[verdict.status] ?? verdict.status}
              {verdict.error ? <div className="verdict-detail">{verdict.error}</div> : null}
            </div>
          ) : null}
        </form>
      </div>

      {user?.id === task.created_by ? <TestsUploadCard taskId={task.id} /> : null}

      {showHistory ? (
        <SubmissionHistoryModal taskId={task.id} onClose={() => setShowHistory(false)} />
      ) : null}
    </div>
  )
}
