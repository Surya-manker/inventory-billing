import api from './axios'

export const login = (email, password) =>
  api.post('/auth/login', { email, password })

export const getMe = () => api.get('/auth/me')

export const logout = (refresh_token) =>
  api.post('/auth/logout', { refresh_token })
