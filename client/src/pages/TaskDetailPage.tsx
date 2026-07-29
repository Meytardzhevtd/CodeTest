import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { ApiError, tasksApi } from '../api'
import type { Task } from '../types'
import { DifficultyBadge } from '../components/DifficultyBadge'
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

export function TaskDetailPage({ taskId }: { taskId: string }) {
  const [task, setTask] = useState<Task | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [code, setCode] = useState('')
  const [language, setLanguage] = useState('python')
  const [submitState, setSubmitState] = useState<SubmitState>('idle')
  const { navigate } = useRouter()
  const { user } = useAuth()

  useEffect(() => {
    let cancelled = false

    tasksApi
      .get(taskId)
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
  }, [taskId])

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    if (!code.trim()) return

    setSubmitState('checking')
    window.setTimeout(() => setSubmitState('done'), 700)
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
            <select value={language} onChange={(event) => setLanguage(event.target.value)}>
              {LANGUAGES.map((lang) => (
                <option key={lang.value} value={lang.value}>
                  {lang.label}
                </option>
              ))}
            </select>
          </div>

          <textarea
            className="code-input"
            value={code}
            onChange={(event) => setCode(event.target.value)}
            placeholder="// Ваше решение..."
            rows={16}
            spellCheck={false}
          />

          <div className="task-submit-footer">
            <button
              type="submit"
              className="btn btn-primary"
              disabled={submitState === 'checking' || !code.trim()}
            >
              {submitState === 'checking' ? 'Проверка...' : 'Отправить решение'}
            </button>
          </div>

          {submitState === 'done' ? (
            <div className="alert alert-info">
              Проверка решений скоро заработает — сервис проверки кода ещё разрабатывается.
            </div>
          ) : null}
        </form>
      </div>

      {user?.id === task.created_by ? <TestsUploadCard taskId={task.id} /> : null}
    </div>
  )
}
