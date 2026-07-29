import { useState } from 'react'
import type { FormEvent, KeyboardEvent } from 'react'
import { ApiError, tasksApi } from '../api'
import type { Difficulty } from '../types'
import { useRouter } from '../router'

export function TaskCreatePage() {
  const [slug, setSlug] = useState('')
  const [title, setTitle] = useState('')
  const [statement, setStatement] = useState('')
  const [difficulty, setDifficulty] = useState<Difficulty>('easy')
  const [timeLimitMs, setTimeLimitMs] = useState(1000)
  const [memoryLimitMb, setMemoryLimitMb] = useState(256)
  const [tags, setTags] = useState<string[]>([])
  const [tagInput, setTagInput] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const { navigate } = useRouter()

  const addTag = () => {
    const tag = tagInput.trim().toLowerCase().replace(/\s+/g, '-')
    if (tag && !tags.includes(tag)) {
      setTags([...tags, tag])
    }
    setTagInput('')
  }

  const removeTag = (tag: string) => {
    setTags(tags.filter((t) => t !== tag))
  }

  const handleTagInputKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter' || event.key === ',') {
      event.preventDefault()
      addTag()
    }
  }

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault()
    setError('')
    setLoading(true)

    try {
      const task = await tasksApi.create({
        slug,
        title,
        statement,
        difficulty,
        time_limit_ms: timeLimitMs,
        memory_limit_mb: memoryLimitMb,
      })

      if (tags.length > 0) {
        await tasksApi.addTags(task.id, tags)
      }

      navigate(`/tasks/${task.slug}`)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Не удалось создать задачу')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="page page-narrow">
      <div className="page-header">
        <div>
          <p className="eyebrow">Новая задача</p>
          <h1>Создать задачу</h1>
          <p className="muted">Только условие — тесты можно будет добавить позже.</p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="card form-card">
        <div className="form-row">
          <label className="field">
            <span>Название</span>
            <input
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              placeholder="Two Sum"
              required
            />
          </label>
          <label className="field">
            <span>Slug</span>
            <input
              value={slug}
              onChange={(event) => setSlug(event.target.value)}
              placeholder="two-sum"
              required
            />
          </label>
        </div>

        <label className="field">
          <span>Условие</span>
          <textarea
            className="statement-input"
            value={statement}
            onChange={(event) => setStatement(event.target.value)}
            rows={10}
            placeholder="Опишите условие задачи..."
            required
          />
        </label>

        <label className="field">
          <span>Теги</span>
          <div className="tag-input-row">
            <input
              value={tagInput}
              onChange={(event) => setTagInput(event.target.value)}
              onKeyDown={handleTagInputKeyDown}
              placeholder="dp, greedy, graphs..."
            />
            <button type="button" className="btn btn-ghost btn-sm" onClick={addTag}>
              Добавить
            </button>
          </div>
          {tags.length > 0 ? (
            <div className="tag-chips">
              {tags.map((tag) => (
                <span className="tag-chip" key={tag}>
                  {tag}
                  <button
                    type="button"
                    className="tag-chip-remove"
                    aria-label={`Убрать тег ${tag}`}
                    onClick={() => removeTag(tag)}
                  >
                    ✕
                  </button>
                </span>
              ))}
            </div>
          ) : null}
        </label>

        <div className="form-row form-row-three">
          <label className="field">
            <span>Сложность</span>
            <select
              value={difficulty}
              onChange={(event) => setDifficulty(event.target.value as Difficulty)}
            >
              <option value="easy">Лёгкая</option>
              <option value="medium">Средняя</option>
              <option value="hard">Сложная</option>
            </select>
          </label>
          <label className="field">
            <span>Лимит времени (мс)</span>
            <input
              type="number"
              min={1}
              value={timeLimitMs}
              onChange={(event) => setTimeLimitMs(Number(event.target.value))}
              required
            />
          </label>
          <label className="field">
            <span>Лимит памяти (МБ)</span>
            <input
              type="number"
              min={1}
              value={memoryLimitMb}
              onChange={(event) => setMemoryLimitMb(Number(event.target.value))}
              required
            />
          </label>
        </div>

        {error ? <div className="alert alert-error">{error}</div> : null}

        <div className="form-actions">
          <button type="button" className="btn btn-ghost" onClick={() => navigate('/tasks')}>
            Отмена
          </button>
          <button type="submit" className="btn btn-primary" disabled={loading}>
            {loading ? 'Создание...' : 'Создать задачу'}
          </button>
        </div>
      </form>
    </div>
  )
}
