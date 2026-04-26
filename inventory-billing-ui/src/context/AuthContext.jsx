import { createContext, useContext, useState, useEffect, useCallback } from 'react'
import { login as loginApi, logout as logoutApi } from '../api/auth'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => {
    try {
      const saved = localStorage.getItem('user')
      return saved ? JSON.parse(saved) : null
    } catch {
      return null
    }
  })
  const [loading, setLoading] = useState(false)

  const login = useCallback(async (email, password) => {
    setLoading(true)
    try {
      const res = await loginApi(email, password)
      const { user, tokens } = res.data.data
      localStorage.setItem('access_token', tokens.access_token)
      localStorage.setItem('refresh_token', tokens.refresh_token)
      localStorage.setItem('user', JSON.stringify(user))
      setUser(user)
      return { ok: true }
    } catch (err) {
      return { ok: false, message: err.response?.data?.message || 'Login failed' }
    } finally {
      setLoading(false)
    }
  }, [])

  const logout = useCallback(async () => {
    const refreshToken = localStorage.getItem('refresh_token')
    try {
      if (refreshToken) await logoutApi(refreshToken)
    } catch {
      // ignore errors on logout
    } finally {
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
      localStorage.removeItem('user')
      setUser(null)
    }
  }, [])

  const isAdmin = user?.role === 'admin'

  // Call this after a successful profile PUT so the sidebar name updates
  // without requiring a page refresh or re-login.
  const updateUser = useCallback((patch) => {
    setUser((prev) => {
      const next = { ...prev, ...patch }
      localStorage.setItem('user', JSON.stringify(next))
      return next
    })
  }, [])

  return (
    <AuthContext.Provider value={{ user, loading, login, logout, isAdmin, updateUser }}>
      {children}
    </AuthContext.Provider>
  )
}

export const useAuth = () => {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}
