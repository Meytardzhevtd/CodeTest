import { useEffect, useState } from 'react'
import { ApiError, tasksApi } from '../api'
import type { Task } from '../types'
import { DifficultyBadge } from '../components/DifficultyBadge'
import { Link } from '../router'
import { useAuth } from '../context/AuthContext'

function excerpt(text: string, length: number): string {
  const trimmed = text.trim()
  if (trimmed.length <= length) {
    return trimmed
  }
  return `${trimmed.slice(0, length).trimEnd()}…`
}

export function TasksListPage() {
  const { user } = useAuth()
  const [tasks, setTasks] = useState<Task[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [onlyMine, setOnlyMine] = useState(false)

  useEffect(() => {
    let cancelled = false

    tasksApi
      .list(50, 0)
      .then((res) => {
        if (cancelled) return
        setTasks(res.tasks)
        setTotal(res.total)
      })
      .catch((err) => {
        if (cancelled) return
        setError(err instanceof ApiError ? err.message : 'Не удалось загрузить задачи')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [])

  const visibleTasks = onlyMine ? tasks.filter((task) => task.created_by === user?.id) : tasks

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <p className="eyebrow">Каталог</p>
          <h1>Задачи</h1>
          <p className="muted">Всего задач: {total}</p>
        </div>
        <Link to="/tasks/new" className="btn btn-primary">
          + Новая задача
        </Link>
      </div>

      {!loading && !error && tasks.length > 0 ? (
        <label className="filter-checkbox">
          <input
            type="checkbox"
            checked={onlyMine}
            onChange={(event) => setOnlyMine(event.target.checked)}
          />
          Мои задачи
        </label>
      ) : null}

      {loading ? (
        <div className="task-grid">
          {[0, 1, 2].map((i) => (
            <div key={i} className="task-card skeleton-card" aria-hidden="true">
              <div className="skeleton-line skeleton-badge" />
              <div className="skeleton-line skeleton-title" />
              <div className="skeleton-line" />
              <div className="skeleton-line skeleton-short" />
            </div>
          ))}
        </div>
      ) : null}

      {!loading && error ? <div className="alert alert-error">{error}</div> : null}

      {!loading && !error && tasks.length === 0 ? (
        <div className="empty-state">
          <p>Пока нет ни одной задачи.</p>
          <Link to="/tasks/new" className="btn btn-primary">
            Создать первую
          </Link>
        </div>
      ) : null}

      {!loading && !error && tasks.length > 0 && visibleTasks.length === 0 ? (
        <div className="empty-state">
          <p>Среди своих задач ничего не нашлось.</p>
        </div>
      ) : null}

      {!loading && !error && visibleTasks.length > 0 ? (
        <div className="task-grid">
          {visibleTasks.map((task) => (
            <Link to={`/tasks/${task.id}`} key={task.id} className="task-card">
              <div className="task-card-top">
                <DifficultyBadge difficulty={task.difficulty} />
                <span className="task-slug">{task.slug}</span>
              </div>
              <h2>{task.title}</h2>
              <p className="task-excerpt">{excerpt(task.statement, 120)}</p>
              <div className="task-meta">
                <span>⏱ {task.time_limit_ms} мс</span>
                <span>▢ {task.memory_limit_mb} МБ</span>
              </div>
              <span className="task-creator">от {task.created_by_username}</span>
            </Link>
          ))}
        </div>
      ) : null}
    </div>
  )
}
