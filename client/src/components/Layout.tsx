import type { ReactNode } from 'react'
import { useAuth } from '../context/AuthContext'
import { Link, useRouter } from '../router'

function isTasksActive(path: string): boolean {
  return path === '/' || path === '/tasks' || (path.startsWith('/tasks/') && path !== '/tasks/new')
}

export function Layout({ children }: { children: ReactNode }) {
  const { user, logout } = useAuth()
  const { path } = useRouter()

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="topbar-inner">
          <div className="topbar-left">
            <Link to="/tasks" className="brand">
              <span className="brand-mark">{'</>'}</span> CodeTest
            </Link>
            <nav className="main-nav">
              <Link to="/tasks" className={isTasksActive(path) ? 'nav-link active' : 'nav-link'}>
                Задачи
              </Link>
            </nav>
          </div>

          <div className="topbar-right">
            <Link to="/tasks/new" className="btn btn-primary btn-sm">
              + Новая задача
            </Link>
            <Link to="/profile" className="user-chip">
              {user?.username}
            </Link>
            <button type="button" className="btn btn-ghost btn-sm" onClick={logout}>
              Выйти
            </button>
          </div>
        </div>
      </header>

      <main className="app-content">{children}</main>
    </div>
  )
}
