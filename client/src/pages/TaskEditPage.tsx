import { useEffect, useState } from 'react'
import type { FormEvent, KeyboardEvent } from 'react'
import { ApiError, tasksApi } from '../api'
import type { Difficulty, Task } from '../types'
import { useAuth } from '../context/AuthContext'
import { useRouter } from '../router'

export function TaskEditPage({ slug }: { slug: string }) {
  const [task, setTask] = useState<Task | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  const [title, setTitle] = useState('')
  const [statement, setStatement] = useState('')
  const [difficulty, setDifficulty] = useState<Difficulty>('easy')
  const [timeLimitMs, setTimeLimitMs] = useState(1000)
  const [memoryLimitMb, setMemoryLimitMb] = useState(256)
  const [tags, setTags] = useState<string[]>([])
  const [tagInput, setTagInput] = useState('')
  const [examples, setExamples] = useState<{ input: string; output: string }[]>([
    { input: '', output: '' },
    { input: '', output: '' },
    { input: '', output: '' },
  ])

  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const { navigate } = useRouter()
  const { user } = useAuth()

  useEffect(() => {
    let cancelled = false

    tasksApi
      .getBySlug(slug)
      .then((res) => {
        if (cancelled) return
        setTask(res)
        setTitle(res.title)
        setStatement(res.statement)
        setDifficulty(res.difficulty)
        setTimeLimitMs(res.time_limit_ms)
        setMemoryLimitMb(res.memory_limit_mb)
        setTags(res.tags)
        setExamples([0, 1, 2].map((i) => res.examples[i] ?? { input: '', output: '' }))
      })
      .catch((err) => {
        if (!cancelled) {
          setLoadError(err instanceof ApiError ? err.message : 'Не удалось загрузить задачу')
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [slug])

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

  const updateExample = (index: number, field: 'input' | 'output', value: string) => {
    setExamples((prev) => prev.map((example, i) => (i === index ? { ...example, [field]: value } : example)))
  }

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault()
    if (!task) return
    setError('')
    setSaving(true)

    try {
      const updated = await tasksApi.update(task.id, {
        title,
        statement,
        difficulty,
        time_limit_ms: timeLimitMs,
        memory_limit_mb: memoryLimitMb,
      })

      await tasksApi.setTags(task.id, tags)

      const nonEmptyExamples = examples.filter((example) => example.input.trim() || example.output.trim())
      await tasksApi.setExamples(task.id, nonEmptyExamples)

      navigate(`/tasks/${updated.slug}`)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Не удалось сохранить изменения')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="page page-narrow">
        <div className="card skeleton-card" aria-hidden="true">
          <div className="skeleton-line skeleton-title" />
          <div className="skeleton-line" />
          <div className="skeleton-line" />
        </div>
      </div>
    )
  }

  if (loadError || !task) {
    return (
      <div className="page">
        <div className="alert alert-error">{loadError || 'Задача не найдена'}</div>
        <button type="button" className="back-link" onClick={() => navigate('/tasks')}>
          ← Все задачи
        </button>
      </div>
    )
  }

  if (user?.id !== task.created_by) {
    return (
      <div className="page">
        <div className="alert alert-error">Редактировать задачу может только её автор.</div>
        <button type="button" className="back-link" onClick={() => navigate(`/tasks/${task.slug}`)}>
          ← К задаче
        </button>
      </div>
    )
  }

  return (
    <div className="page page-narrow">
      <div className="page-header">
        <div>
          <p className="eyebrow">Редактирование</p>
          <h1>{task.title}</h1>
          <p className="muted">Slug задачи менять нельзя — ссылка на неё останется прежней.</p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="card form-card">
        <label className="field">
          <span>Название</span>
          <input value={title} onChange={(event) => setTitle(event.target.value)} required />
        </label>

        <label className="field">
          <span>Условие</span>
          <textarea
            className="statement-input"
            value={statement}
            onChange={(event) => setStatement(event.target.value)}
            rows={10}
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

        <div className="field">
          <span>Примеры (необязательно, до 3 штук)</span>
          <p className="muted example-hint">
            Показываются всем на странице задачи. Достаточно заполнить хотя бы одно поле в паре —
            пустые пары не сохранятся.
          </p>
          <div className="example-list">
            {examples.map((example, index) => (
              <div className="example-pair" key={index}>
                <span className="example-pair-label">Пример {index + 1}</span>
                <div className="form-row">
                  <label className="field">
                    <span>Ввод</span>
                    <textarea
                      rows={3}
                      value={example.input}
                      onChange={(event) => updateExample(index, 'input', event.target.value)}
                    />
                  </label>
                  <label className="field">
                    <span>Вывод</span>
                    <textarea
                      rows={3}
                      value={example.output}
                      onChange={(event) => updateExample(index, 'output', event.target.value)}
                    />
                  </label>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="form-row form-row-three">
          <label className="field">
            <span>Сложность</span>
            <select value={difficulty} onChange={(event) => setDifficulty(event.target.value as Difficulty)}>
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
          <button type="button" className="btn btn-ghost" onClick={() => navigate(`/tasks/${task.slug}`)}>
            Отмена
          </button>
          <button type="submit" className="btn btn-primary" disabled={saving}>
            {saving ? 'Сохранение...' : 'Сохранить изменения'}
          </button>
        </div>
      </form>
    </div>
  )
}
