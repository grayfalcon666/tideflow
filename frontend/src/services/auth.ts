import api from './api'
import type { ApiResponse, LoginReq, LoginResp } from '../types'

export const login = (data: LoginReq) =>
  api.post<ApiResponse<LoginResp>>('/auth/login', data)

export const register = (data: { username: string; password: string }) =>
  api.post<ApiResponse<{ access_token: string; refresh_token: string }>>('/auth/register', data)

export const refreshToken = (refresh_token: string) =>
  api.post<ApiResponse<{ access_token: string }>>('/auth/refresh', { refresh_token })
