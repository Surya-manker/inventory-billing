import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getDashboardStats } from '../api/dashboard'
import { getInvoices } from '../api/invoices'
import { useAuth } from '../context/AuthContext'
import LoadingSpinner from '../components/common/LoadingSpinner'
import Badge from '../components/common/Badge'
import { formatCurrency, formatDate, statusColors } from '../utils/formatters'

// ─── Stat Card ────────────────────────────────────────────────────────────────

function StatCard({ title, value, icon, bg, fg, to, sub }) {
  const inner = (
    <div className={`card p-5 group ${to ? 'hover:shadow-md transition-all duration-200 cursor-pointer' : ''}`}>
      <div className="flex items-start justify-between">
        <div className={`h-11 w-11 rounded-xl flex items-center justify-center ${bg}`}>
          <span className={fg}>{icon}</span>
        </div>
        {to && (
          <svg className="h-4 w-4 text-gray-300 group-hover:text-brand-500 transition-colors mt-1"
            fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        )}
      </div>
      <div className="mt-4">
        <p className="text-2xl font-bold text-gray-900 tabular-nums">
          {value ?? <span className="text-gray-300 text-lg">—</span>}
        </p>
        <p className="text-sm text-gray-500 mt-0.5">{title}</p>
        {sub && <p className="text-xs text-gray-400 mt-1">{sub}</p>}
      </div>
    </div>
  )
  return to ? <Link to={to}>{inner}</Link> : inner
}

// ─── Invoice Status Ring ──────────────────────────────────────────────────────

function InvoiceStatusBar({ pending, paid, canceled }) {
  const total = pending + paid + canceled
  if (total === 0) return <p className="text-sm text-gray-400 py-4 text-center">No invoices yet</p>

  const paidPct = Math.round((paid / total) * 100)
  const pendingPct = Math.round((pending / total) * 100)
  const canceledPct = Math.round((canceled / total) * 100)

  return (
    <div className="space-y-3">
      {[
        { label: 'Paid', count: paid, pct: paidPct, bar: 'bg-green-500', text: 'text-green-700', bg: 'bg-green-50' },
        { label: 'Pending', count: pending, pct: pendingPct, bar: 'bg-yellow-400', text: 'text-yellow-700', bg: 'bg-yellow-50' },
        { label: 'Canceled', count: canceled, pct: canceledPct, bar: 'bg-red-400', text: 'text-red-700', bg: 'bg-red-50' },
      ].map(({ label, count, pct, bar, text, bg }) => (
        <div key={label}>
          <div className="flex items-center justify-between mb-1">
            <div className="flex items-center gap-2">
              <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${bg} ${text}`}>
                {label}
              </span>
            </div>
            <span className="text-sm font-semibold text-gray-700 tabular-nums">{count}</span>
          </div>
          <div className="h-2 w-full bg-gray-100 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-700 ${bar}`}
              style={{ width: `${pct}%` }}
            />
          </div>
        </div>
      ))}
    </div>
  )
}

// ─── Low Stock Row ────────────────────────────────────────────────────────────

function LowStockRow({ item }) {
  const pct = Math.min(100, Math.round((item.stock / (item.low_stock_threshold || 10)) * 100))
  return (
    <div className="flex items-center gap-3 py-2.5">
      <div className="h-8 w-8 rounded-lg bg-red-50 flex items-center justify-center flex-shrink-0">
        <svg className="h-4 w-4 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between mb-1">
          <p className="text-sm font-medium text-gray-800 truncate">{item.name}</p>
          <span className="text-xs font-semibold text-red-600 ml-2 flex-shrink-0">{item.stock} left</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="flex-1 h-1.5 bg-gray-100 rounded-full overflow-hidden">
            <div
              className="h-full bg-red-400 rounded-full"
              style={{ width: `${pct}%` }}
            />
          </div>
          <span className="text-xs text-gray-400 font-mono flex-shrink-0">{item.sku}</span>
        </div>
      </div>
    </div>
  )
}

// ─── Main Component ───────────────────────────────────────────────────────────

export default function Dashboard() {
  const { user } = useAuth()
  const [stats, setStats] = useState(null)
  const [recentInvoices, setRecentInvoices] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    const load = async () => {
      try {
        const [s, inv] = await Promise.all([
          getDashboardStats(),
          getInvoices({ limit: 6 }),
        ])
        setStats(s)
        setRecentInvoices(inv.data.data.items ?? [])
      } catch {
        setError('Failed to load dashboard data.')
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <LoadingSpinner size="lg" />
      </div>
    )
  }

  return (
    <div className="p-4 sm:p-6 lg:p-8 space-y-6 max-w-7xl mx-auto">

      {/* ── Header ── */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">
            Good {getGreeting()}, {user?.name?.split(' ')[0]} 👋
          </h1>
          <p className="text-gray-500 text-sm mt-1">
            Here's what's happening with your business today.
          </p>
        </div>
        <div className="text-right text-sm text-gray-400 hidden sm:block">
          <p className="font-medium text-gray-600">{new Date().toLocaleDateString('en-IN', { weekday: 'long' })}</p>
          <p>{new Date().toLocaleDateString('en-IN', { day: 'numeric', month: 'long', year: 'numeric' })}</p>
        </div>
      </div>

      {error && (
        <div className="p-4 bg-red-50 border border-red-200 rounded-xl text-sm text-red-700 flex items-center gap-2">
          <svg className="h-4 w-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
              d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          {error}
        </div>
      )}

      {/* ── Stat Cards ── */}
      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-5">
        <StatCard
          title="Total Products"
          value={stats?.totalProducts}
          to="/products"
          bg="bg-blue-100"
          fg="text-blue-600"
          sub={stats?.lowStockCount > 0 ? `${stats.lowStockCount} low stock` : 'All stocked'}
          icon={
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8}
                d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
          }
        />
        <StatCard
          title="Total Customers"
          value={stats?.totalCustomers}
          to="/customers"
          bg="bg-emerald-100"
          fg="text-emerald-600"
          sub={stats?.totalCustomers === 0 ? 'No customers yet' : `${stats.totalCustomers} total`}
          icon={
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8}
                d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          }
        />
        <StatCard
          title="Total Invoices"
          value={stats?.totalInvoices}
          to="/invoices"
          bg="bg-violet-100"
          fg="text-violet-600"
          sub={stats?.pendingInvoices > 0 ? `${stats.pendingInvoices} pending` : 'None pending'}
          icon={
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8}
                d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
          }
        />
        <StatCard
          title="Low Stock Alerts"
          value={stats?.lowStockCount}
          bg="bg-red-100"
          fg="text-red-500"
          sub={stats?.lowStockCount === 0 ? 'All items healthy' : 'Needs restocking'}
          icon={
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8}
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          }
        />
      </div>

      {/* ── Middle Row ── */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">

        {/* Invoice Status Breakdown */}
        <div className="card p-6">
          <h2 className="text-sm font-semibold text-gray-900 mb-5">Invoice Status</h2>
          <InvoiceStatusBar
            paid={stats?.paidInvoices ?? 0}
            pending={stats?.pendingInvoices ?? 0}
            canceled={stats?.canceledInvoices ?? 0}
          />
          <div className="mt-5 pt-4 border-t border-gray-100 grid grid-cols-3 gap-2 text-center">
            {[
              { label: 'Paid', val: stats?.paidInvoices, cls: 'text-green-600' },
              { label: 'Pending', val: stats?.pendingInvoices, cls: 'text-yellow-600' },
              { label: 'Canceled', val: stats?.canceledInvoices, cls: 'text-red-500' },
            ].map(({ label, val, cls }) => (
              <div key={label}>
                <p className={`text-xl font-bold tabular-nums ${cls}`}>{val ?? 0}</p>
                <p className="text-xs text-gray-400 mt-0.5">{label}</p>
              </div>
            ))}
          </div>
        </div>

        {/* Low Stock Alerts */}
        <div className="card p-6 lg:col-span-2">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-sm font-semibold text-gray-900">Low Stock Alerts</h2>
            <Link to="/products" className="text-xs text-brand-600 hover:text-brand-700 font-medium">
              Manage stock →
            </Link>
          </div>
          {stats?.lowStockItems?.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-8 text-center">
              <div className="h-10 w-10 rounded-full bg-green-100 flex items-center justify-center mb-3">
                <svg className="h-5 w-5 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <p className="text-sm font-medium text-gray-700">All products are well stocked</p>
              <p className="text-xs text-gray-400 mt-1">No alerts at this time</p>
            </div>
          ) : (
            <div className="divide-y divide-gray-50">
              {stats.lowStockItems.map((item) => (
                <LowStockRow key={item.id} item={item} />
              ))}
            </div>
          )}
        </div>
      </div>

      {/* ── Recent Invoices ── */}
      <div className="card overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-100 flex items-center justify-between">
          <div>
            <h2 className="text-sm font-semibold text-gray-900">Recent Invoices</h2>
            <p className="text-xs text-gray-400 mt-0.5">Latest 6 transactions</p>
          </div>
          <Link
            to="/invoices"
            className="text-sm text-brand-600 hover:text-brand-700 font-medium flex items-center gap-1"
          >
            View all
            <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </Link>
        </div>

        {recentInvoices.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-14 text-center">
            <div className="h-12 w-12 rounded-full bg-gray-100 flex items-center justify-center mb-3">
              <svg className="h-6 w-6 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
                  d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <p className="text-sm font-medium text-gray-600">No invoices yet</p>
            <p className="text-xs text-gray-400 mt-1">Create your first invoice to get started</p>
            <Link to="/invoices" className="btn-primary btn-sm mt-4">
              Create Invoice
            </Link>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-gray-50 border-b border-gray-100">
                  <th className="table-th">Invoice #</th>
                  <th className="table-th">Customer</th>
                  <th className="table-th hidden sm:table-cell">Items</th>
                  <th className="table-th">Amount</th>
                  <th className="table-th">Status</th>
                  <th className="table-th hidden md:table-cell">Date</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-50">
                {recentInvoices.map((inv) => (
                  <tr key={inv.id} className="hover:bg-gray-50/70 transition-colors">
                    <td className="table-td">
                      <span className="font-mono text-xs text-brand-600 bg-brand-50 px-2 py-0.5 rounded">
                        {inv.invoice_number}
                      </span>
                    </td>
                    <td className="table-td">
                      <div>
                        <p className="font-medium text-gray-800 text-sm">{inv.customer?.name ?? '—'}</p>
                        {inv.customer?.company && (
                          <p className="text-xs text-gray-400">{inv.customer.company}</p>
                        )}
                      </div>
                    </td>
                    <td className="table-td hidden sm:table-cell text-gray-500 text-xs">
                      {inv.items?.length ?? 0} item{inv.items?.length !== 1 ? 's' : ''}
                    </td>
                    <td className="table-td font-semibold text-gray-900">
                      {formatCurrency(inv.total_price)}
                    </td>
                    <td className="table-td">
                      <Badge className={statusColors[inv.status]}>
                        {inv.status}
                      </Badge>
                    </td>
                    <td className="table-td hidden md:table-cell text-gray-400 text-xs">
                      {formatDate(inv.issued_at)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

    </div>
  )
}

function getGreeting() {
  const h = new Date().getHours()
  if (h < 12) return 'morning'
  if (h < 17) return 'afternoon'
  return 'evening'
}
