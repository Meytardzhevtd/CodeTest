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
  tags: string[]
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

export interface UploadTestsResponse {
  tests_uploaded: number
}

export interface AddTagsResponse {
  tags: string[]
}

export type SubmissionStatus = 'pending' | 'running' | 'OK' | 'WA' | 'RE' | 'CE' | 'TL' | 'ML' | 'ERROR'

export interface Submission {
  id: string
  task_id: string
  user_id: string
  code: string
  language: string
  status: SubmissionStatus
  output?: string
  error?: string
  created_at: string
  updated_at: string
}

export interface CreateSubmissionRequest {
  task_id: string
  code: string
  language: string
}

export interface CreateSubmissionResponse {
  id: string
  status: SubmissionStatus
}

export interface GetSubmissionResponse {
  submission: Submission
}
