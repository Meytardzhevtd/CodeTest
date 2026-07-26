import { useEffect, useMemo, useState } from 'react'
import './App.css'
import type { AuthResponse } from './types'

type Mode = 'login' | 'register'

function App() {
  const [mode, setMode] = useState<Mode>('register')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [isLoggedIn, setIsLoggedIn] = useState(false)
  const [userEmail, setUserEmail] = useState('')

  useEffect(() => {
    const saved = localStorage.getItem('auth:access_token')
    const savedEmail = localStorage.getItem('auth:user_email')
    if (saved) {
      setIsLoggedIn(true)
      setUserEmail(savedEmail ?? '')
    }
  }, [])

  const title = useMemo(() => (mode === 'register' ? 'Create account' : 'Welcome back'), [mode])

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    setError('')
    setLoading(true)

    try {
      const response = await fetch(`/api/auth/${mode}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      })

      const payload = await response.json().catch(() => ({}))

      if (!response.ok) {
        throw new Error(payload.error || 'Request failed')
      }

      const auth = payload as AuthResponse
      localStorage.setItem('auth:access_token', auth.access_token)
      localStorage.setItem('auth:user_email', auth.user.email)
      setUserEmail(auth.user.email)
      setIsLoggedIn(true)
      setEmail('')
      setPassword('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unexpected error')
    } finally {
      setLoading(false)
    }
  }

  const handleLogout = () => {
    localStorage.removeItem('auth:access_token')
    localStorage.removeItem('auth:user_email')
    setIsLoggedIn(false)
    setUserEmail('')
  }

  if (isLoggedIn) {
    return (
      <main className="success-shell">
        <div className="success-card">
          <div className="success-badge">✓</div>
          <h1>Login successful</h1>
          <p>Welcome back, {userEmail || 'friend'}.</p>
          <button type="button" className="primary-btn" onClick={handleLogout}>
            Log out
          </button>
        </div>
      </main>
    )
  }

  return (
    <main className="auth-shell">
      <section className="auth-card">
        <div className="auth-header">
          <p className="eyebrow">CodeTest</p>
          <h1>{title}</h1>
          <p className="muted">
            {mode === 'register'
              ? 'Create a fresh account to get started.'
              : 'Sign in to continue to your workspace.'}
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

          <label className="field">
            <span>Password</span>
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="At least 8 characters"
              required
            />
          </label>

          {error ? <div className="error">{error}</div> : null}

          <button type="submit" className="primary-btn" disabled={loading}>
            {loading ? 'Working...' : mode === 'register' ? 'Create account' : 'Log in'}
          </button>
        </form>

        <div className="switch-row">
          {mode === 'register' ? (
            <>
              <span>Already have an account?</span>
              <button type="button" className="link-btn" onClick={() => setMode('login')}>
                Sign in
              </button>
            </>
          ) : (
            <>
              <span>Need an account?</span>
              <button type="button" className="link-btn" onClick={() => setMode('register')}>
                Register
              </button>
            </>
          )}
        </div>
      </section>
    </main>
  )
}

export default App
