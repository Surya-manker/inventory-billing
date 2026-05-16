import { useState, useEffect } from 'react'
import { useAuth } from '../context/AuthContext'
import { getMe, updateProfile, changePassword } from '../api/auth'

// ── helpers ────────────────────────────────────────────────────────────────────

function avatarColor(name = '') {
  const palettes = [
    ['bg-violet-500', 'text-violet-50'],
    ['bg-blue-500', 'text-blue-50'],
    ['bg-emerald-500', 'text-emerald-50'],
    ['bg-rose-500', 'text-rose-50'],
    ['bg-amber-500', 'text-amber-50'],
    ['bg-cyan-500', 'text-cyan-50'],
    ['bg-pink-500', 'text-pink-50'],
    ['bg-indigo-500', 'text-indigo-50'],
  ]
  let hash = 0
  for (let i = 0; i < name.length; i++) hash = name.charCodeAt(i) + ((hash << 5) - hash)
  return palettes[Math.abs(hash) % palettes.length]
}

function passwordStrength(pw) {
  let score = 0
  if (pw.length >= 8) score++
  if (pw.length >= 12) score++
  if (/[A-Z]/.test(pw) && /[a-z]/.test(pw)) score++
  if (/[0-9]/.test(pw) || /[^A-Za-z0-9]/.test(pw)) score++
  return score
}

const strengthLabels = ['', 'Weak', 'Fair', 'Good', 'Strong']
const strengthColors = [
  '',
  'bg-red-400',
  'bg-yellow-400',
  'bg-blue-400',
  'bg-green-500',
]

function PasswordStrengthBar({ password }) {
  if (!password) return null
  const s = passwordStrength(password)
  return (
    <div className="mt-2 space-y-1.5">
      <div className="flex gap-1">
        {[1, 2, 3, 4].map((i) => (
          <div
            key={i}
            className={`h-1.5 flex-1 rounded-full transition-all duration-300 ${s >= i ? strengthColors[s] : 'bg-gray-200'
              }`}
          />
        ))}
      </div>
      <p className={`text-xs font-medium ${s <= 1 ? 'text-red-500' : s === 2 ? 'text-yellow-600' : s === 3 ? 'text-blue-600' : 'text-green-600'
        }`}>
        {strengthLabels[s]}
      </p>
    </div>
  )
}

// ── reusable field components ──────────────────────────────────────────────────

function Field({ label, children, hint }) {
  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1.5">{label}</label>
      {children}
      {hint && <p className="mt-1.5 text-xs text-gray-400">{hint}</p>}
    </div>
  )
}

function Toast({ type, message, onClose }) {
  useEffect(() => {
    const t = setTimeout(onClose, 4000)
    return () => clearTimeout(t)
  }, [onClose])

  const styles = {
    success: 'bg-green-50 border-green-200 text-green-800',
    error: 'bg-red-50   border-red-200   text-red-800',
  }
  const icons = {
    success: (
      <svg className="h-4 w-4 text-green-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
      </svg>
    ),
    error: (
      <svg className="h-4 w-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
  }

  return (
    <div className={`flex items-center gap-3 px-4 py-3 rounded-xl border text-sm font-medium animate-in slide-in-from-top-2 ${styles[type]}`}>
      {icons[type]}
      <span className="flex-1">{message}</span>
      <button onClick={onClose} className="ml-2 opacity-60 hover:opacity-100 transition-opacity">
        <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  )
}

// ── main component ─────────────────────────────────────────────────────────────

export default function Profile() {
  const { user, isAdmin, updateUser } = useAuth()

  const [freshUser, setFreshUser] = useState(user)
  const [fetchError, setFetchError] = useState('')

  // Profile edit state
  const [profileForm, setProfileForm] = useState({ name: user?.name || '', email: user?.email || '' })
  const [profileSaving, setProfileSaving] = useState(false)
  const [profileToast, setProfileToast] = useState(null) // { type, message }

  // Password state
  const [pwForm, setPwForm] = useState({ old_password: '', new_password: '', confirm: '' })
  const [pwSaving, setPwSaving] = useState(false)
  const [pwToast, setPwToast] = useState(null)
  const [showOld, setShowOld] = useState(false)
  const [showNew, setShowNew] = useState(false)
  const [showConfirm, setShowConfirm] = useState(false)

  // Fetch fresh user data on mount
  useEffect(() => {
    getMe()
      .then((res) => {
        const u = res.data.data
        setFreshUser(u)
        setProfileForm({ name: u.name, email: u.email })
      })
      .catch(() => setFetchError('Could not load profile. Showing cached data.'))
  }, [])

  const displayUser = freshUser || user
  const initials = displayUser?.name
    ? displayUser.name.split(' ').map((w) => w[0]).slice(0, 2).join('').toUpperCase()
    : '?'
  const [avatarBg, avatarFg] = avatarColor(displayUser?.name)

  const joinedDate = displayUser?.created_at
    ? new Date(displayUser.created_at).toLocaleDateString('en-IN', {
      day: 'numeric', month: 'long', year: 'numeric',
    })
    : '—'

  // ── handlers ────────────────────────────────────────────────────────────────

  const handleProfileSave = async (e) => {
    e.preventDefault()
    if (!profileForm.name.trim()) return
    setProfileSaving(true)
    setProfileToast(null)
    try {
      await updateProfile(displayUser.id, {
        name: profileForm.name.trim(),
        email: profileForm.email.trim(),
      })
      updateUser({ name: profileForm.name.trim(), email: profileForm.email.trim() })
      setFreshUser((prev) => ({ ...prev, name: profileForm.name.trim(), email: profileForm.email.trim() }))
      setProfileToast({ type: 'success', message: 'Profile updated successfully.' })
    } catch (err) {
      setProfileToast({ type: 'error', message: err.response?.data?.message || 'Failed to update profile.' })
    } finally {
      setProfileSaving(false)
    }
  }

  const handlePasswordSave = async (e) => {
    e.preventDefault()
    setPwToast(null)
    if (pwForm.new_password.length < 8) {
      setPwToast({ type: 'error', message: 'New password must be at least 8 characters.' })
      return
    }
    if (pwForm.new_password !== pwForm.confirm) {
      setPwToast({ type: 'error', message: 'New passwords do not match.' })
      return
    }
    setPwSaving(true)
    try {
      await changePassword(pwForm.old_password, pwForm.new_password)
      setPwForm({ old_password: '', new_password: '', confirm: '' })
      setPwToast({ type: 'success', message: 'Password changed. Use it next time you log in.' })
    } catch (err) {
      setPwToast({ type: 'error', message: err.response?.data?.message || 'Failed to change password.' })
    } finally {
      setPwSaving(false)
    }
  }

  const pf = (k) => (e) => setProfileForm({ ...profileForm, [k]: e.target.value })
  const pw = (k) => (e) => setPwForm({ ...pwForm, [k]: e.target.value })

  const profileDirty =
    profileForm.name.trim() !== (displayUser?.name || '') ||
    profileForm.email.trim() !== (displayUser?.email || '')

  // ── render ──────────────────────────────────────────────────────────────────

  return (
    <div className="min-h-full bg-gray-50">
      {/* ── Page header ── */}
      <div className="bg-white border-b border-gray-200 px-4 sm:px-8 py-5">
        <h1 className="text-xl font-bold text-gray-900">My Profile</h1>
        <p className="text-sm text-gray-500 mt-0.5">Manage your account information and security settings</p>
      </div>

      <div className="px-4 sm:px-6 lg:px-8 py-4 sm:py-6 lg:py-8 max-w-5xl mx-auto space-y-5 sm:space-y-6">

        {fetchError && (
          <div className="p-3 bg-amber-50 border border-amber-200 rounded-xl text-sm text-amber-700 flex items-center gap-2">
            <svg className="h-4 w-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            {fetchError}
          </div>
        )}

        {/* ── Identity card ── */}
        <div className="bg-white rounded-2xl border border-gray-200 shadow-sm overflow-visible">
          {/* Gradient banner */}
          <div className="h-24 sm:h-32 bg-gradient-to-r from-brand-700 via-brand-600 to-violet-600 relative">
            <div className="absolute inset-0 opacity-20"
              style={{ backgroundImage: 'radial-gradient(circle at 20% 50%, white 1px, transparent 1px), radial-gradient(circle at 80% 20%, white 1px, transparent 1px)', backgroundSize: '30px 30px' }} />
          </div>

          <div className="px-4 sm:px-6 pb-6">
            <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4 -mt-8 sm:-mt-12">
              {/* Avatar */}
              <div className={`relative z-10 h-20 w-20 sm:h-28 sm:w-28 rounded-2xl ${avatarBg} ${avatarFg} flex items-center justify-center text-2xl sm:text-3xl font-bold ring-4 ring-white shadow-lg flex-shrink-0`}>
                {initials}
              </div>

              {/* Role badge */}
              <div className="pb-1">
                <span className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold ${isAdmin
                  ? 'bg-violet-100 text-violet-700'
                  : 'bg-blue-100 text-blue-700'
                  }`}>
                  <span className="h-1.5 w-1.5 rounded-full bg-current" />
                  {displayUser?.role?.charAt(0).toUpperCase() + displayUser?.role?.slice(1)}
                </span>
              </div>
            </div>

            <div className="mt-4">
              <h2 className="text-xl sm:text-2xl font-bold text-gray-900">{displayUser?.name}</h2>
              <p className="text-gray-500 text-sm mt-0.5">{displayUser?.email}</p>
            </div>

            {/* Stats row */}
            <div className="mt-5 grid grid-cols-2 sm:grid-cols-3 gap-3">
              {[
                {
                  label: 'Account Type',
                  value: displayUser?.role?.charAt(0).toUpperCase() + displayUser?.role?.slice(1),
                  icon: (
                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                    </svg>
                  ),
                },
                {
                  label: 'Member Since',
                  value: joinedDate,
                  icon: (
                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                    </svg>
                  ),
                },
                {
                  label: 'User ID',
                  value: displayUser?.id ? `#${displayUser.id.slice(0, 8)}…` : '—',
                  icon: (
                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V8a2 2 0 00-2-2h-5m-4 0V5a2 2 0 114 0v1m-4 0a2 2 0 104 0m-5 8a2 2 0 100-4 2 2 0 000 4zm0 0c1.306 0 2.417.835 2.83 2M9 14a3.001 3.001 0 00-2.83 2M15 11h3m-3 4h2" />
                    </svg>
                  ),
                },
              ].map(({ label, value, icon }) => (
                <div key={label} className="flex items-center gap-3 p-3 bg-gray-50 rounded-xl">
                  <div className="h-8 w-8 rounded-lg bg-white shadow-sm border border-gray-100 flex items-center justify-center text-gray-500 flex-shrink-0">
                    {icon}
                  </div>
                  <div className="min-w-0">
                    <p className="text-xs text-gray-400">{label}</p>
                    <p className="text-sm font-semibold text-gray-800 truncate">{value}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* ── Two-column grid: Edit Profile + Change Password ── */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">

          {/* ── Edit Profile card ── */}
          <div className="bg-white rounded-2xl border border-gray-200 shadow-sm">
            <div className="px-6 py-5 border-b border-gray-100 flex items-center gap-3">
              <div className="h-9 w-9 rounded-xl bg-brand-50 flex items-center justify-center">
                <svg className="h-5 w-5 text-brand-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                </svg>
              </div>
              <div>
                <h3 className="text-sm font-semibold text-gray-900">Personal Information</h3>
                <p className="text-xs text-gray-400 mt-0.5">
                  {isAdmin ? 'Update your display name and email' : 'Contact an admin to update your details'}
                </p>
              </div>
            </div>

            <form onSubmit={handleProfileSave} className="p-6 space-y-5">
              {profileToast && (
                <Toast
                  type={profileToast.type}
                  message={profileToast.message}
                  onClose={() => setProfileToast(null)}
                />
              )}

              <Field label="Full Name">
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <svg className="h-4 w-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                    </svg>
                  </div>
                  <input
                    type="text"
                    className={`input pl-9 ${!isAdmin ? 'bg-gray-50 text-gray-500 cursor-not-allowed' : ''}`}
                    value={profileForm.name}
                    onChange={pf('name')}
                    disabled={!isAdmin}
                    required
                    placeholder="Your full name"
                  />
                </div>
              </Field>

              <Field
                label="Email Address"
                hint={!isAdmin ? 'Only admins can change email addresses.' : ''}
              >
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <svg className="h-4 w-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                    </svg>
                  </div>
                  <input
                    type="email"
                    className={`input pl-9 ${!isAdmin ? 'bg-gray-50 text-gray-500 cursor-not-allowed' : ''}`}
                    value={profileForm.email}
                    onChange={pf('email')}
                    disabled={!isAdmin}
                    required
                    placeholder="your@email.com"
                  />
                </div>
              </Field>

              <Field label="Role">
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <svg className="h-4 w-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                    </svg>
                  </div>
                  <input
                    type="text"
                    className="input pl-9 bg-gray-50 text-gray-500 cursor-not-allowed capitalize"
                    value={displayUser?.role || ''}
                    disabled
                    readOnly
                  />
                </div>
              </Field>

              {isAdmin ? (
                <button
                  type="submit"
                  disabled={profileSaving || !profileDirty}
                  className={`w-full flex items-center justify-center gap-2 py-2.5 px-4 rounded-xl text-sm font-medium transition-all ${profileDirty && !profileSaving
                    ? 'bg-brand-600 text-white hover:bg-brand-700 shadow-sm hover:shadow'
                    : 'bg-gray-100 text-gray-400 cursor-not-allowed'
                    }`}
                >
                  {profileSaving ? (
                    <>
                      <span className="h-4 w-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
                      Saving…
                    </>
                  ) : (
                    <>
                      <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                      Save Changes
                    </>
                  )}
                </button>
              ) : (
                <div className="flex items-center gap-2 p-3 bg-amber-50 border border-amber-200 rounded-xl text-xs text-amber-700">
                  <svg className="h-4 w-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  Contact your administrator to update your profile information.
                </div>
              )}
            </form>
          </div>

          {/* ── Change Password card ── */}
          <div className="bg-white rounded-2xl border border-gray-200 shadow-sm">
            <div className="px-6 py-5 border-b border-gray-100 flex items-center gap-3">
              <div className="h-9 w-9 rounded-xl bg-red-50 flex items-center justify-center">
                <svg className="h-5 w-5 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                </svg>
              </div>
              <div>
                <h3 className="text-sm font-semibold text-gray-900">Change Password</h3>
                <p className="text-xs text-gray-400 mt-0.5">Must be at least 8 characters</p>
              </div>
            </div>

            <form onSubmit={handlePasswordSave} className="p-6 space-y-5">
              {pwToast && (
                <Toast
                  type={pwToast.type}
                  message={pwToast.message}
                  onClose={() => setPwToast(null)}
                />
              )}

              {/* Current password */}
              <Field label="Current Password">
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <svg className="h-4 w-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
                    </svg>
                  </div>
                  <input
                    type={showOld ? 'text' : 'password'}
                    className="input pl-9 pr-10"
                    placeholder="••••••••"
                    value={pwForm.old_password}
                    onChange={pw('old_password')}
                    required
                  />
                  <button
                    type="button"
                    onClick={() => setShowOld((v) => !v)}
                    className="absolute inset-y-0 right-0 pr-3 flex items-center text-gray-400 hover:text-gray-600"
                  >
                    {showOld ? (
                      <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                      </svg>
                    ) : (
                      <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                      </svg>
                    )}
                  </button>
                </div>
              </Field>

              {/* New password */}
              <Field label="New Password">
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <svg className="h-4 w-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                    </svg>
                  </div>
                  <input
                    type={showNew ? 'text' : 'password'}
                    className="input pl-9 pr-10"
                    placeholder="Min. 8 characters"
                    value={pwForm.new_password}
                    onChange={pw('new_password')}
                    minLength={8}
                    required
                  />
                  <button
                    type="button"
                    onClick={() => setShowNew((v) => !v)}
                    className="absolute inset-y-0 right-0 pr-3 flex items-center text-gray-400 hover:text-gray-600"
                  >
                    {showNew ? (
                      <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                      </svg>
                    ) : (
                      <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                      </svg>
                    )}
                  </button>
                </div>
                <PasswordStrengthBar password={pwForm.new_password} />
              </Field>

              {/* Confirm password */}
              <Field label="Confirm New Password">
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <svg className="h-4 w-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                    </svg>
                  </div>
                  <input
                    type={showConfirm ? 'text' : 'password'}
                    className={`input pl-9 pr-10 ${pwForm.confirm && pwForm.new_password !== pwForm.confirm
                      ? 'border-red-300 focus:border-red-400 focus:ring-red-300'
                      : pwForm.confirm && pwForm.new_password === pwForm.confirm
                        ? 'border-green-300 focus:border-green-400 focus:ring-green-300'
                        : ''
                      }`}
                    placeholder="Re-enter new password"
                    value={pwForm.confirm}
                    onChange={pw('confirm')}
                    required
                  />
                  <button
                    type="button"
                    onClick={() => setShowConfirm((v) => !v)}
                    className="absolute inset-y-0 right-0 pr-3 flex items-center text-gray-400 hover:text-gray-600"
                  >
                    {showConfirm ? (
                      <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                      </svg>
                    ) : (
                      <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                      </svg>
                    )}
                  </button>
                </div>
                {pwForm.confirm && pwForm.new_password !== pwForm.confirm && (
                  <p className="mt-1.5 text-xs text-red-500 flex items-center gap-1">
                    <svg className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                    Passwords do not match
                  </p>
                )}
                {pwForm.confirm && pwForm.new_password === pwForm.confirm && pwForm.confirm.length > 0 && (
                  <p className="mt-1.5 text-xs text-green-600 flex items-center gap-1">
                    <svg className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                    </svg>
                    Passwords match
                  </p>
                )}
              </Field>

              {/* Security tips */}
              <div className="p-3 bg-gray-50 rounded-xl space-y-1.5">
                <p className="text-xs font-medium text-gray-500 uppercase tracking-wide">Tips for a strong password</p>
                {[
                  'At least 8 characters',
                  'Mix of upper and lowercase letters',
                  'Include numbers or symbols',
                  'Avoid using personal information',
                ].map((tip) => (
                  <div key={tip} className="flex items-center gap-2 text-xs text-gray-500">
                    <div className="h-1 w-1 rounded-full bg-gray-400 flex-shrink-0" />
                    {tip}
                  </div>
                ))}
              </div>

              <button
                type="submit"
                disabled={pwSaving || !pwForm.old_password || !pwForm.new_password || !pwForm.confirm}
                className="w-full flex items-center justify-center gap-2 py-2.5 px-4 rounded-xl text-sm font-medium bg-gray-900 text-white hover:bg-gray-800 disabled:bg-gray-200 disabled:text-gray-400 disabled:cursor-not-allowed transition-all shadow-sm hover:shadow"
              >
                {pwSaving ? (
                  <>
                    <span className="h-4 w-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
                    Updating…
                  </>
                ) : (
                  <>
                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                    </svg>
                    Update Password
                  </>
                )}
              </button>
            </form>
          </div>
        </div>

        {/* ── Danger zone (admin only) ── */}
        {isAdmin && (
          <div className="bg-white rounded-2xl border border-red-200 shadow-sm">
            <div className="px-6 py-5 border-b border-red-100 flex items-center gap-3">
              <div className="h-9 w-9 rounded-xl bg-red-50 flex items-center justify-center">
                <svg className="h-5 w-5 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
              </div>
              <div>
                <h3 className="text-sm font-semibold text-red-700">Danger Zone</h3>
                <p className="text-xs text-red-400 mt-0.5">These actions are irreversible</p>
              </div>
            </div>
            <div className="px-6 py-5 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
              <div>
                <p className="text-sm font-medium text-gray-900">Sign out of all sessions</p>
                <p className="text-xs text-gray-400 mt-0.5">
                  Invalidates your refresh token and logs you out everywhere.
                </p>
              </div>
              <button
                type="button"
                className="flex-shrink-0 flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium text-red-600 bg-red-50 hover:bg-red-100 border border-red-200 transition-colors"
              >
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                </svg>
                Sign Out All
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
