import { useState } from 'react'
import { Link, useNavigate, Navigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import api from '../api/axios'

export default function Register() {
  const { user } = useAuth()
  const navigate = useNavigate()

  const [form, setForm] = useState({ name: '', email: '', password: '', confirm: '', role: 'admin' })
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [loading, setLoading] = useState(false)

  if (user) return <Navigate to="/dashboard" replace />

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setSuccess('')

    if (form.password.length < 8) {
      return setError('Password must be at least 8 characters.')
    }
    if (form.password !== form.confirm) {
      return setError('Passwords do not match.')
    }

    setLoading(true)
    try {
      await api.post('/auth/register', {
        name: form.name,
        email: form.email,
        password: form.password,
        role: form.role,
      })
      setSuccess('Account created! Redirecting to login…')
      setTimeout(() => navigate('/login'), 1500)
    } catch (err) {
      setError(err.response?.data?.message || 'Registration failed. Try again.')
    } finally {
      setLoading(false)
    }
  }

  const f = (k) => (e) => setForm({ ...form, [k]: e.target.value })

  return (
    <div className="min-h-screen bg-gradient-to-br from-brand-900 via-brand-800 to-brand-700 flex items-center justify-center p-4">
      <div className="w-full max-w-md">

        {/* Logo */}
        <div className="text-center mb-8">
          <div className="inline-flex h-14 w-14 rounded-2xl bg-white/10 backdrop-blur items-center justify-center mb-4">
            <svg className="h-7 w-7 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
            </svg>
          </div>
          <h1 className="text-2xl font-bold text-white">InvoBill</h1>
          <p className="text-brand-300 text-sm mt-1">Inventory & Billing System</p>
        </div>

        <div className="bg-white rounded-2xl shadow-2xl p-8">
          <h2 className="text-xl font-semibold text-gray-900 mb-1">Create an account</h2>
          <p className="text-sm text-gray-400 mb-6">Fill in the details below to get started</p>

          {/* Error */}
          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700 flex items-center gap-2">
              <svg className="h-4 w-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                  d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              {error}
            </div>
          )}

          {/* Success */}
          {success && (
            <div className="mb-4 p-3 bg-green-50 border border-green-200 rounded-lg text-sm text-green-700 flex items-center gap-2">
              <svg className="h-4 w-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              {success}
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            {/* Full Name */}
            <div>
              <label className="label">Full Name *</label>
              <input
                className="input"
                placeholder="John Doe"
                required
                value={form.name}
                onChange={f('name')}
                autoFocus
              />
            </div>

            {/* Email */}
            <div>
              <label className="label">Email Address *</label>
              <input
                type="email"
                className="input"
                placeholder="you@example.com"
                required
                value={form.email}
                onChange={f('email')}
              />
            </div>

            {/* Role */}
            <div>
              <label className="label">Role *</label>
              <div className="grid grid-cols-2 gap-3 mt-1">
                {[
                  { val: 'admin', label: 'Admin', desc: 'Full access' },
                  { val: 'staff', label: 'Staff', desc: 'View & create invoices' },
                ].map(({ val, label, desc }) => (
                  <button
                    key={val}
                    type="button"
                    onClick={() => setForm({ ...form, role: val })}
                    className={`flex flex-col items-start p-3 rounded-lg border-2 text-left transition-colors ${form.role === val
                        ? 'border-brand-500 bg-brand-50'
                        : 'border-gray-200 hover:border-gray-300'
                      }`}
                  >
                    <span className={`text-sm font-semibold ${form.role === val ? 'text-brand-700' : 'text-gray-700'}`}>
                      {label}
                    </span>
                    <span className="text-xs text-gray-400 mt-0.5">{desc}</span>
                  </button>
                ))}
              </div>
            </div>

            {/* Password */}
            <div>
              <label className="label">Password *</label>
              <input
                type="password"
                className="input"
                placeholder="Min. 8 characters"
                required
                minLength={8}
                value={form.password}
                onChange={f('password')}
              />
              {/* Strength indicator */}
              {form.password.length > 0 && (
                <div className="mt-2 flex gap-1">
                  {[1, 2, 3, 4].map((i) => (
                    <div
                      key={i}
                      className={`h-1 flex-1 rounded-full transition-colors ${passwordStrength(form.password) >= i
                          ? strengthColor(passwordStrength(form.password))
                          : 'bg-gray-200'
                        }`}
                    />
                  ))}
                </div>
              )}
            </div>

            {/* Confirm Password */}
            <div>
              <label className="label">Confirm Password *</label>
              <input
                type="password"
                className="input"
                placeholder="Re-enter password"
                required
                value={form.confirm}
                onChange={f('confirm')}
              />
              {form.confirm && form.password !== form.confirm && (
                <p className="text-xs text-red-500 mt-1">Passwords do not match</p>
              )}
              {form.confirm && form.password === form.confirm && form.confirm.length > 0 && (
                <p className="text-xs text-green-600 mt-1 flex items-center gap-1">
                  <svg className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                  </svg>
                  Passwords match
                </p>
              )}
            </div>

            <button
              type="submit"
              className="btn-primary w-full py-2.5 mt-2"
              disabled={loading || !!success}
            >
              {loading ? (
                <span className="flex items-center justify-center gap-2">
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                  Creating account…
                </span>
              ) : (
                'Create Account'
              )}
            </button>
          </form>

          <p className="mt-6 text-center text-sm text-gray-500">
            Already have an account?{' '}
            <Link to="/login" className="text-brand-600 hover:text-brand-700 font-medium">
              Sign in
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}

function passwordStrength(pw) {
  let score = 0
  if (pw.length >= 8) score++
  if (pw.length >= 12) score++
  if (/[A-Z]/.test(pw) && /[a-z]/.test(pw)) score++
  if (/[0-9]/.test(pw) || /[^A-Za-z0-9]/.test(pw)) score++
  return score
}

function strengthColor(score) {
  if (score <= 1) return 'bg-red-400'
  if (score === 2) return 'bg-yellow-400'
  if (score === 3) return 'bg-blue-400'
  return 'bg-green-500'
}
