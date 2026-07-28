import { useState } from 'react'
import type { FormEvent } from 'react'
import { useAuth } from '../context/AuthContext'
import { ApiError, authApi } from '../api'

export function ProfilePage() {
  const { user, setUser } = useAuth()
  const [username, setUsername] = useState(user?.username ?? '')
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault()
    setError('')
    setSuccess('')
    setLoading(true)

    try {
      const res = await authApi.updateProfile({
        username: username !== user?.username ? username : undefined,
        current_password: currentPassword || undefined,
        new_password: newPassword || undefined,
      })
      setUser(res.user)
      setUsername(res.user.username)
      setCurrentPassword('')
      setNewPassword('')
      setSuccess('Профиль обновлён')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Не удалось обновить профиль')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="page page-narrow">
      <div className="page-header">
        <div>
          <p className="eyebrow">Аккаунт</p>
          <h1>Профиль</h1>
        </div>
      </div>

      <div className="profile-grid">
        <div className="card">
          <h2>Данные аккаунта</h2>
          <div className="profile-info">
            <div>
              <span>Email</span>
              <strong>{user?.email ?? '—'}</strong>
            </div>
            <div>
              <span>Логин</span>
              <strong>{user?.username ?? '—'}</strong>
            </div>
            <div>
              <span>Регистрация</span>
              <strong>
                {user?.created_at ? new Date(user.created_at).toLocaleDateString('ru-RU') : '—'}
              </strong>
            </div>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="card form-card">
          <h2>Изменить профиль</h2>
          <label className="field">
            <span>Имя пользователя</span>
            <input value={username} onChange={(event) => setUsername(event.target.value)} />
          </label>
          <label className="field">
            <span>Текущий пароль</span>
            <input
              type="password"
              value={currentPassword}
              onChange={(event) => setCurrentPassword(event.target.value)}
              placeholder="Введите текущий пароль"
            />
          </label>
          <label className="field">
            <span>Новый пароль</span>
            <input
              type="password"
              value={newPassword}
              onChange={(event) => setNewPassword(event.target.value)}
              placeholder="Минимум 8 символов"
            />
          </label>

          {error ? <div className="alert alert-error">{error}</div> : null}
          {success ? <div className="alert alert-success">{success}</div> : null}

          <button type="submit" className="btn btn-primary" disabled={loading}>
            {loading ? 'Сохранение...' : 'Сохранить'}
          </button>
        </form>
      </div>
    </div>
  )
}
