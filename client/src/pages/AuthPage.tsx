import { useState } from 'react'
import type { FormEvent } from 'react'
import { useAuth } from '../context/AuthContext'
import { ApiError } from '../api'
import { useRouter } from '../router'

type Mode = 'login' | 'register'

export function AuthPage({ initialMode }: { initialMode: Mode }) {
  const [mode, setMode] = useState<Mode>(initialMode)
  const [email, setEmail] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const { login, register } = useAuth()
  const { navigate } = useRouter()

  const switchMode = (next: Mode) => {
    setMode(next)
    setError('')
    navigate(next === 'register' ? '/register' : '/login')
  }

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault()
    setError('')
    setLoading(true)

    try {
      if (mode === 'register') {
        await register(email, username, password)
      } else {
        await login(email, password)
      }
      navigate('/tasks')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Что-то пошло не так')
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="auth-shell">
      <section className="auth-card">
        <div className="brand">
          <span className="brand-mark">{'</>'}</span> CodeTest
        </div>

        <div className="auth-header">
          <h1>{mode === 'register' ? 'Создать аккаунт' : 'С возвращением'}</h1>
          <p className="muted">
            {mode === 'register'
              ? 'Зарегистрируйтесь, чтобы решать задачи.'
              : 'Войдите, чтобы продолжить.'}
          </p>
        </div>

        <form onSubmit={handleSubmit} className="auth-form">
          <label className="field">
            <span>Email</span>
            <input
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="you@example.com"
              required
            />
          </label>

          {mode === 'register' ? (
            <label className="field">
              <span>Имя пользователя</span>
              <input
                type="text"
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                placeholder="coolname"
                required
              />
            </label>
          ) : null}

          <label className="field">
            <span>Пароль</span>
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="Минимум 8 символов"
              required
            />
          </label>

          {error ? <div className="alert alert-error">{error}</div> : null}

          <button type="submit" className="btn btn-primary" disabled={loading}>
            {loading ? 'Подождите...' : mode === 'register' ? 'Создать аккаунт' : 'Войти'}
          </button>
        </form>

        <div className="switch-row">
          {mode === 'register' ? (
            <>
              <span>Уже есть аккаунт?</span>
              <button type="button" className="link-btn" onClick={() => switchMode('login')}>
                Войти
              </button>
            </>
          ) : (
            <>
              <span>Нет аккаунта?</span>
              <button type="button" className="link-btn" onClick={() => switchMode('register')}>
                Зарегистрироваться
              </button>
            </>
          )}
        </div>
      </section>
    </main>
  )
}
