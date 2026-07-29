import './App.css'
import { AuthProvider, useAuth } from './context/AuthContext'
import { RouterProvider, matchPath, useRouter } from './router'
import { Layout } from './components/Layout'
import { AuthPage } from './pages/AuthPage'
import { TasksListPage } from './pages/TasksListPage'
import { TaskCreatePage } from './pages/TaskCreatePage'
import { TaskDetailPage } from './pages/TaskDetailPage'
import { ProfilePage } from './pages/ProfilePage'

function FullscreenLoader() {
  return (
    <div className="fullscreen-loader">
      <div className="spinner" />
    </div>
  )
}

function AppRoutes() {
  const { user, loading } = useAuth()
  const { path } = useRouter()

  if (loading) {
    return <FullscreenLoader />
  }

  if (!user) {
    const mode = path === '/register' ? 'register' : 'login'
    return <AuthPage key={mode} initialMode={mode} />
  }

  switch (path) {
    case '/tasks/new':
      return (
        <Layout>
          <TaskCreatePage />
        </Layout>
      )
    case '/profile':
      return (
        <Layout>
          <ProfilePage />
        </Layout>
      )
  }

  const taskDetailParams = matchPath('/tasks/:slug', path)
  if (taskDetailParams) {
    return (
      <Layout>
        <TaskDetailPage key={taskDetailParams.slug} slug={taskDetailParams.slug} />
      </Layout>
    )
  }

  return (
    <Layout>
      <TasksListPage />
    </Layout>
  )
}

function App() {
  return (
    <AuthProvider>
      <RouterProvider>
        <AppRoutes />
      </RouterProvider>
    </AuthProvider>
  )
}

export default App
