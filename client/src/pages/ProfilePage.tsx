import { useRef, useState } from 'react'
import type { ChangeEvent, FormEvent } from 'react'
import { useAuth } from '../context/AuthContext'
import { ApiError, authApi } from '../api'

const MAX_AVATAR_BYTES = 5 * 1024 * 1024
const ALLOWED_AVATAR_TYPES = ['image/jpeg', 'image/png', 'image/webp']

export function ProfilePage() {
  const { user, setUser } = useAuth()
  const [username, setUsername] = useState(user?.username ?? '')
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [loading, setLoading] = useState(false)
  const [avatarError, setAvatarError] = useState('')
  const [avatarUploading, setAvatarUploading] = useState(false)
  const avatarInputRef = useRef<HTMLInputElement>(null)

  const handleAvatarChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return

    setAvatarError('')
    if (!ALLOWED_AVATAR_TYPES.includes(file.type)) {
      setAvatarError('Поддерживаются только JPEG, PNG и WebP')
      return
    }
    if (file.size > MAX_AVATAR_BYTES) {
      setAvatarError('Файл слишком большой (максимум 5 МБ)')
      return
    }

    setAvatarUploading(true)
    try {
      const res = await authApi.uploadAvatar(file)
      setUser(res.user)
    } catch (err) {
      setAvatarError(err instanceof ApiError ? err.message : 'Не удалось загрузить аватарку')
    } finally {
      setAvatarUploading(false)
    }
  }

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
          <div className="profile-avatar">
            {user?.avatar_url ? (
              <img src={user.avatar_url} alt="Аватарка" className="profile-avatar-img" />
            ) : (
              <div className="profile-avatar-placeholder">
                {(user?.username ?? '?').charAt(0).toUpperCase()}
              </div>
            )}
            <div>
              <button
                type="button"
                className="btn btn-ghost"
                disabled={avatarUploading}
                onClick={() => avatarInputRef.current?.click()}
              >
                {avatarUploading ? 'Загрузка...' : 'Загрузить аватарку'}
              </button>
              <input
                ref={avatarInputRef}
                type="file"
                accept="image/jpeg,image/png,image/webp"
                hidden
                onChange={handleAvatarChange}
              />
              {avatarError ? <div className="alert alert-error">{avatarError}</div> : null}
            </div>
          </div>
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
