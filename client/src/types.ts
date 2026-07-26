export interface AuthResponse {
  access_token: string
  token_type: string
  expires_in: number
  user: {
    id: string
    email: string
    username: string
    created_at: string
  }
}
