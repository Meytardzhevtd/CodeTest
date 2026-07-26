import { useEffect, useMemo, useState } from 'react'
import './App.css'
import type { AuthResponse, ProfileResponse, User } from './types'

type Mode = 'login' | 'register'

function App() {
  const [mode, setMode] = useState<Mode>('register')
  const [email, setEmail] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [isLoggedIn, setIsLoggedIn] = useState(false)
  const [profile, setProfile] = useState<User | null>(null)
  const [profileUsername, setProfileUsername] = useState('')
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [profileError, setProfileError] = useState('')
  const [profileSuccess, setProfileSuccess] = useState('')

  useEffect(() => {
    const saved = localStorage.getItem('auth:access_token')
    if (saved) {
      setIsLoggedIn(true)
      void loadProfile(saved)
    }
  }, [])

  const title = useMemo(() => (mode === 'register' ? 'Create account' : 'Welcome back'), [mode])

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    setError('')
    setLoading(true)

    try {
      const payloadBody = mode === 'register'
        ? { email, username, password }
        : { email, password }

      const response = await fetch(`/api/auth/${mode}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payloadBody),
      })

      const payload = await response.json().catch(() => ({}))

      if (!response.ok) {
        throw new Error(payload.error || 'Request failed')
      }

      const auth = payload as AuthResponse
      localStorage.setItem('auth:access_token', auth.access_token)
      setProfile(auth.user)
      setProfileUsername(auth.user.username)
      setIsLoggedIn(true)
      setEmail('')
      setUsername('')
      setPassword('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unexpected error')
    } finally {
      setLoading(false)
    }
  }

  const loadProfile = async (token: string) => {
    try {
      const response = await fetch('/api/auth/me', {
        headers: { Authorization: `Bearer ${token}` },
      })
      const payload = await response.json().catch(() => ({}))
      if (!response.ok) {
        throw new Error(payload.error || 'Failed to load profile')
      }
      const profilePayload = payload as ProfileResponse
      setProfile(profilePayload.user)
      setProfileUsername(profilePayload.user.username)
    } catch (err) {
      setProfileError(err instanceof Error ? err.message : 'Failed to load profile')
    }
  }

  const handleProfileUpdate = async (event: React.FormEvent) => {
    event.preventDefault()
    setProfileError('')
    setProfileSuccess('')

    const token = localStorage.getItem('auth:access_token')
    if (!token) {
      setProfileError('Please log in again')
      return
    }

    try {
      const response = await fetch('/api/auth/profile', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          username: profileUsername,
          current_password: currentPassword || undefined,
          new_password: newPassword || undefined,
        }),
      })

      const payload = await response.json().catch(() => ({}))
      if (!response.ok) {
        throw new Error(payload.error || 'Update failed')
      }

      const updated = payload as ProfileResponse
      setProfile(updated.user)
      setProfileUsername(updated.user.username)
      setCurrentPassword('')
      setNewPassword('')
      setProfileSuccess('Profile updated successfully')
    } catch (err) {
      setProfileError(err instanceof Error ? err.message : 'Unexpected error')
    }
  }

  const handleLogout = () => {
    localStorage.removeItem('auth:access_token')
    setIsLoggedIn(false)
    setProfile(null)
    setProfileUsername('')
    setCurrentPassword('')
    setNewPassword('')
    setProfileError('')
    setProfileSuccess('')
  }

  if (isLoggedIn) {
    return (
      <main className="dashboard-shell">
        <section className="dashboard-card">
          <div className="dashboard-header">
            <div>
              <p className="eyebrow">Personal cabinet</p>
              <h1>Welcome, {profile?.username || 'friend'}</h1>
              <p className="muted">Manage your profile and account security.</p>
            </div>
            <button type="button" className="primary-btn" onClick={handleLogout}>
              Log out
            </button>
          </div>

          <div className="profile-grid">
            <div className="profile-panel">
              <h2>Account details</h2>
              <div className="profile-info">
                <div><span>Email</span><strong>{profile?.email || '—'}</strong></div>
                <div><span>Username</span><strong>{profile?.username || '—'}</strong></div>
                <div><span>Joined</span><strong>{profile?.created_at ? new Date(profile.created_at).toLocaleDateString() : '—'}</strong></div>
              </div>
            </div>

            <form onSubmit={handleProfileUpdate} className="profile-panel form-panel">
              <h2>Update profile</h2>
              <label className="field">
                <span>Username</span>
                <input
                  type="text"
                  value={profileUsername}
                  onChange={(event) => setProfileUsername(event.target.value)}
                  placeholder="new username"
                />
              </label>

              <label className="field">
                <span>Current password</span>
                <input
                  type="password"
                  value={currentPassword}
                  onChange={(event) => setCurrentPassword(event.target.value)}
                  placeholder="Enter current password"
                />
              </label>

              <label className="field">
                <span>New password</span>
                <input
                  type="password"
                  value={newPassword}
                  onChange={(event) => setNewPassword(event.target.value)}
                  placeholder="At least 8 characters"
                />
              </label>

              {profileError ? <div className="error">{profileError}</div> : null}
              {profileSuccess ? <div className="success-message">{profileSuccess}</div> : null}

              <button type="submit" className="primary-btn" disabled={loading}>
                {loading ? 'Saving...' : 'Save changes'}
              </button>
            </form>
          </div>
        </section>
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

          {mode === 'register' ? (
            <label className="field">
              <span>Username</span>
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
