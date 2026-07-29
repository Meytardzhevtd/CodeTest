export interface User {
  id: string
  email: string
  username: string
  created_at: string
}

export interface AuthResponse {
  access_token: string
  token_type: string
  expires_in: number
  user: User
}

export interface ProfileResponse {
  user: User
}

export type Difficulty = 'easy' | 'medium' | 'hard'

export interface Task {
  id: string
  slug: string
  title: string
  statement: string
  difficulty: Difficulty
  time_limit_ms: number
  memory_limit_mb: number
  created_by: string
  created_by_username: string
  created_at: string
  updated_at: string
}

export interface TaskListResponse {
  tasks: Task[]
  total: number
  limit: number
  offset: number
}

export interface CreateTaskRequest {
  slug: string
  title: string
  statement: string
  difficulty: Difficulty
  time_limit_ms: number
  memory_limit_mb: number
}
