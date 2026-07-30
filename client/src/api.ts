import type {
  AddTagsResponse,
  AuthResponse,
  CreateSubmissionRequest,
  CreateSubmissionResponse,
  CreateTaskRequest,
  GetSubmissionResponse,
  ProfileResponse,
  SubmissionHistoryResponse,
  Task,
  TaskListResponse,
  UploadTestsResponse,
} from './types'

const TOKEN_KEY = 'auth:access_token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken()
  // FormData bodies (e.g. file uploads) must not get an explicit Content-Type:
  // the browser needs to set its own multipart boundary.
  const isFormData = options.body instanceof FormData
  const headers: Record<string, string> = {
    ...(options.body && !isFormData ? { 'Content-Type': 'application/json' } : {}),
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  }

  const response = await fetch(path, { ...options, headers })
  const payload = await response.json().catch(() => null)

  if (!response.ok) {
    const message =
      payload && typeof payload === 'object' && 'error' in payload
        ? String((payload as { error: unknown }).error)
        : 'Request failed'
    throw new ApiError(response.status, message)
  }

  return payload as T
}

export const authApi = {
  register: (body: { email: string; username: string; password: string }) =>
    request<AuthResponse>('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  login: (body: { email: string; password: string }) =>
    request<AuthResponse>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  logout: () => request<{ message: string }>('/api/auth/logout', { method: 'POST' }),

  me: () => request<ProfileResponse>('/api/auth/me'),

  updateProfile: (body: {
    username?: string
    current_password?: string
    new_password?: string
  }) =>
    request<ProfileResponse>('/api/auth/profile', {
      method: 'PUT',
      body: JSON.stringify(body),
    }),
}

export const tasksApi = {
  list: (limit = 50, offset = 0) =>
    request<TaskListResponse>(`/api/tasks?limit=${limit}&offset=${offset}`),

  getBySlug: (slug: string) => request<Task>(`/api/tasks/slug/${encodeURIComponent(slug)}`),

  create: (body: CreateTaskRequest) =>
    request<Task>('/api/tasks', {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  uploadTests: (taskId: string, archive: File) => {
    const formData = new FormData()
    formData.append('archive', archive)
    return request<UploadTestsResponse>(`/api/tasks/${encodeURIComponent(taskId)}/tests`, {
      method: 'POST',
      body: formData,
    })
  },

  addTags: (taskId: string, tags: string[]) =>
    request<AddTagsResponse>(`/api/tasks/${encodeURIComponent(taskId)}/tags`, {
      method: 'POST',
      body: JSON.stringify({ tags }),
    }),
}

export const submitApi = {
  create: (body: CreateSubmissionRequest) =>
    request<CreateSubmissionResponse>('/api/submit', {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  get: (id: string) => request<GetSubmissionResponse>(`/api/submit?id=${encodeURIComponent(id)}`),

  history: (taskId: string) =>
    request<SubmissionHistoryResponse>(`/api/submit/history?task_id=${encodeURIComponent(taskId)}`),
}
