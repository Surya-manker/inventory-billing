import { NavLink, Link } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'

const NAV = [
  {
    to: '/dashboard',
    label: 'Dashboard',
    icon: (
      <svg className="h-5 w-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8}
          d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
      </svg>
    ),
  },
  {
    to: '/products',
    label: 'Products',
    icon: (
      <svg className="h-5 w-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8}
          d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
      </svg>
    ),
  },
  {
    to: '/customers',
    label: 'Customers',
    icon: (
      <svg className="h-5 w-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8}
          d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
    ),
  },
  {
    to: '/invoices',
    label: 'Invoices',
    icon: (
      <svg className="h-5 w-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8}
          d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
    ),
  },
]

export default function Sidebar({ open, onClose }) {
  const { user, logout } = useAuth()

  return (
    <aside
      className={[
        // Base styles
        'flex flex-col w-64 bg-brand-900 text-white flex-shrink-0',
        // Mobile: fixed full-height drawer, above backdrop (z-30)
        'fixed inset-y-0 left-0 z-30 transition-transform duration-300 ease-in-out',
        // Desktop: static, always visible, no transform needed
        'lg:static lg:z-auto lg:translate-x-0',
        // Slide in/out on mobile
        open ? 'translate-x-0' : '-translate-x-full',
      ].join(' ')}
    >
      {/* ── Logo + mobile close button ── */}
      <div className="px-4 py-4 border-b border-brand-800 flex items-center justify-between gap-3">
        <div className="flex items-center gap-3 min-w-0">
          <div className="h-8 w-8 rounded-lg bg-brand-500 flex items-center justify-center flex-shrink-0">
            <svg className="h-4 w-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
            </svg>
          </div>
          <div className="min-w-0">
            <p className="text-sm font-bold leading-tight">InvoBill</p>
            <p className="text-xs text-brand-400 truncate">Inventory & Billing</p>
          </div>
        </div>

        {/* Close button — mobile only */}
        <button
          onClick={onClose}
          className="lg:hidden p-1.5 rounded-lg text-brand-400 hover:text-white hover:bg-brand-800 transition-colors flex-shrink-0"
          aria-label="Close navigation"
        >
          <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* ── Nav links ── */}
      <nav className="flex-1 px-3 py-4 space-y-0.5 overflow-y-auto">
        {NAV.map(({ to, label, icon }) => (
          <NavLink
            key={to}
            to={to}
            onClick={onClose}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                isActive
                  ? 'bg-brand-700 text-white'
                  : 'text-brand-300 hover:bg-brand-800 hover:text-white active:bg-brand-700'
              }`
            }
          >
            {icon}
            {label}
          </NavLink>
        ))}
      </nav>

      {/* ── User section ── */}
      <div className="px-3 py-3 border-t border-brand-800 space-y-0.5">
        {/* Profile link */}
        <Link
          to="/profile"
          onClick={onClose}
          className="flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-brand-800 active:bg-brand-700 transition-colors group"
        >
          <div className="h-8 w-8 rounded-full bg-brand-600 flex items-center justify-center text-xs font-bold flex-shrink-0 group-hover:ring-2 group-hover:ring-white/30 transition-all">
            {user?.name?.[0]?.toUpperCase() ?? '?'}
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium truncate leading-tight">{user?.name}</p>
            <p className="text-xs text-brand-400 capitalize">{user?.role}</p>
          </div>
          <svg className="h-3.5 w-3.5 text-brand-500 group-hover:text-brand-300 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        </Link>

        {/* Logout */}
        <button
          onClick={logout}
          className="w-full flex items-center gap-3 px-3 py-2.5 text-sm text-brand-300 hover:text-white hover:bg-brand-800 active:bg-brand-700 rounded-lg transition-colors"
        >
          <svg className="h-5 w-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8}
              d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
          </svg>
          Logout
        </button>
      </div>
    </aside>
  )
}
