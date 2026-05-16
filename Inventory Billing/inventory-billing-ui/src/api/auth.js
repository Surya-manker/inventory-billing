import api from './axios'

export const login = (email, password) =>
  api.post('/auth/login', { email, password })

export const getMe = () => api.get('/auth/me')

export const logout = (refresh_token) =>
  api.post('/auth/logout', { refresh_token })

export const updateProfile = (userId, data) =>
  api.put(`/users/${userId}`, data)

export const changePassword = (old_password, new_password) =>
  api.put('/auth/change-password', { old_password, new_password })
