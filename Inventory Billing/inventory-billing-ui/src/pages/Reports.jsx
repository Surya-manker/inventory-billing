import { useEffect, useState } from 'react'
import { getRevenueReport, getTopProducts, getInventoryValue, getAgingReport } from '../api/reports'
import LoadingSpinner from '../components/common/LoadingSpinner'
import { formatCurrency, formatDate } from '../utils/formatters'

const today = () => new Date().toISOString().split('T')[0]
const firstOfMonth = () => {
  const d = new Date()
  return new Date(d.getFullYear(), d.getMonth(), 1).toISOString().split('T')[0]
}

export default function Reports() {
  const [from, setFrom] = useState(firstOfMonth())
  const [to,   setTo]   = useState(today())

  const [revenue,   setRevenue]   = useState(null)
  const [topProds,  setTopProds]  = useState([])
  const [inventory, setInventory] = useState(null)
  const [aging,     setAging]     = useState(null)
  const [loading,   setLoading]   = useState(true)
  const [error,     setError]     = useState('')

  const load = async () => {
    setLoading(true); setError('')
    try {
      const [r, t, i, a] = await Promise.all([
        getRevenueReport({ from_date: from, to_date: to }),
        getTopProducts({ from_date: from, to_date: to, limit: 10 }),
        getInventoryValue(),
        getAgingReport(),
      ])
      setRevenue(r.data.data)
      setTopProds(t.data.data?.items ?? t.data.data ?? [])
      setInventory(i.data.data)
      setAging(a.data.data)
    } catch { setError('Failed to load report data.') }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="p-4 sm:p-6 lg:p-8 space-y-6">
      <div>
        <h1 className="text-xl sm:text-2xl font-bold text-gray-900">Reports</h1>
        <p className="text-gray-500 text-sm mt-0.5">Business analytics and financial overview</p>
      </div>

      {/* Date range + apply */}
      <div className="card p-4 flex flex-wrap items-end gap-3">
        <div>
          <label className="label">From</label>
          <input type="date" className="input w-auto" value={from} max={to}
            onChange={(e) => setFrom(e.target.value)} />
        </div>
        <div>
          <label className="label">To</label>
          <input type="date" className="input w-auto" value={to} min={from} max={today()}
            onChange={(e) => setTo(e.target.value)} />
        </div>
        <button className="btn-primary" onClick={load} disabled={loading}>
          {loading ? 'Loading…' : 'Apply'}
        </button>
      </div>

      {error && <div className="p-3 bg-red-50 border border-red-200 rounded-xl text-sm text-red-700">{error}</div>}

      {loading ? <LoadingSpinner className="py-20" /> : (
        <div className="space-y-6">

          {/* Revenue Summary */}
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            {[
              { label: 'Revenue',      val: formatCurrency(revenue?.total_revenue   ?? revenue?.revenue ?? 0),   bg: 'bg-green-50',  fg: 'text-green-700' },
              { label: 'Tax Collected',val: formatCurrency(revenue?.total_tax       ?? revenue?.tax_amount ?? 0), bg: 'bg-blue-50',   fg: 'text-blue-700'  },
              { label: 'Invoices',     val: revenue?.invoice_count  ?? '—',                                       bg: 'bg-violet-50', fg: 'text-violet-700'},
              { label: 'Inv. Value',   val: formatCurrency(inventory?.total_value   ?? 0),                        bg: 'bg-amber-50',  fg: 'text-amber-700' },
            ].map(({ label, val, bg, fg }) => (
              <div key={label} className={`card p-4 ${bg}`}>
                <p className={`text-2xl font-bold tabular-nums ${fg}`}>{val}</p>
                <p className="text-sm text-gray-500 mt-1">{label}</p>
              </div>
            ))}
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">

            {/* Top Products */}
            <div className="card overflow-hidden">
              <div className="px-5 py-4 border-b border-gray-100">
                <h2 className="text-sm font-semibold text-gray-900">Top Products by Revenue</h2>
                <p className="text-xs text-gray-400 mt-0.5">{formatDate(from)} – {formatDate(to)}</p>
              </div>
              {topProds.length === 0 ? (
                <div className="py-10 text-center text-gray-400 text-sm">No sales data for this period.</div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="table-th">#</th>
                        <th className="table-th">Product</th>
                        <th className="table-th text-right">Qty</th>
                        <th className="table-th text-right">Revenue</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                      {topProds.map((p, i) => (
                        <tr key={p.product_id ?? i} className="hover:bg-gray-50/70">
                          <td className="table-td text-gray-400 w-8">{i + 1}</td>
                          <td className="table-td font-medium text-gray-800">{p.product_name ?? p.name}</td>
                          <td className="table-td text-right text-gray-600 tabular-nums">{p.quantity_sold ?? p.quantity}</td>
                          <td className="table-td text-right font-semibold text-gray-900">
                            {formatCurrency(p.revenue ?? p.total)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>

            {/* AR Aging */}
            <div className="card overflow-hidden">
              <div className="px-5 py-4 border-b border-gray-100">
                <h2 className="text-sm font-semibold text-gray-900">Accounts Receivable Aging</h2>
                <p className="text-xs text-gray-400 mt-0.5">Outstanding invoices by age</p>
              </div>
              {!aging || Object.keys(aging).length === 0 ? (
                <div className="py-10 text-center text-gray-400 text-sm">No outstanding receivables.</div>
              ) : (
                <div className="divide-y divide-gray-100">
                  {[
                    { label: 'Current (0–30 days)',   key: 'current'   },
                    { label: '31–60 days',            key: 'days_31_60'},
                    { label: '61–90 days',            key: 'days_61_90'},
                    { label: 'Over 90 days',          key: 'over_90'   },
                  ].map(({ label, key }) => {
                    const bucket = aging[key] ?? aging[label] ?? { count: 0, amount: 0 }
                    const amount = bucket.amount ?? bucket.total_amount ?? 0
                    const count  = bucket.count  ?? bucket.invoice_count ?? 0
                    return (
                      <div key={key} className="px-5 py-3 flex items-center justify-between">
                        <div>
                          <p className="text-sm font-medium text-gray-800">{label}</p>
                          <p className="text-xs text-gray-400">{count} invoice{count !== 1 ? 's' : ''}</p>
                        </div>
                        <p className={`font-semibold text-sm tabular-nums ${amount > 0 ? 'text-red-600' : 'text-gray-400'}`}>
                          {formatCurrency(amount)}
                        </p>
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          </div>

          {/* Inventory Value Table */}
          {inventory?.items?.length > 0 && (
            <div className="card overflow-hidden">
              <div className="px-5 py-4 border-b border-gray-100">
                <h2 className="text-sm font-semibold text-gray-900">Inventory Value Breakdown</h2>
                <p className="text-xs text-gray-400 mt-0.5">Total: {formatCurrency(inventory.total_value)}</p>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="table-th">Product</th>
                      <th className="table-th text-right">Stock</th>
                      <th className="table-th text-right hidden sm:table-cell">Cost Price</th>
                      <th className="table-th text-right">Value</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {inventory.items.map((item) => (
                      <tr key={item.product_id ?? item.id} className="hover:bg-gray-50/70">
                        <td className="table-td">
                          <p className="font-medium text-gray-800">{item.product_name ?? item.name}</p>
                          <p className="text-xs font-mono text-gray-400">{item.sku}</p>
                        </td>
                        <td className="table-td text-right text-gray-600 tabular-nums">{item.stock}</td>
                        <td className="table-td text-right text-gray-500 hidden sm:table-cell">
                          {formatCurrency(item.cost_price ?? 0)}
                        </td>
                        <td className="table-td text-right font-semibold text-gray-900">
                          {formatCurrency(item.value ?? item.total_value ?? 0)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
