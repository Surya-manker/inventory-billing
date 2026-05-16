import { useEffect, useState, useCallback } from 'react'
import { getStockLogs } from '../api/stock'
import LoadingSpinner from '../components/common/LoadingSpinner'
import Pagination from '../components/common/Pagination'
import Badge from '../components/common/Badge'
import { formatDateTime } from '../utils/formatters'

const LIMIT = 20

const CHANGE_TYPE_COLORS = {
  sale:         'bg-red-100   text-red-700',
  purchase:     'bg-green-100 text-green-700',
  adjustment:   'bg-blue-100  text-blue-700',
  return:       'bg-yellow-100 text-yellow-700',
  transfer_in:  'bg-emerald-100 text-emerald-700',
  transfer_out: 'bg-orange-100  text-orange-700',
}

export default function StockLogs() {
  const [logs, setLogs]       = useState([])
  const [total, setTotal]     = useState(0)
  const [offset, setOffset]   = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError]     = useState('')

  const [filter, setFilter] = useState({ change_type: '', from_date: '', to_date: '' })

  const load = useCallback(async () => {
    setLoading(true); setError('')
    try {
      const params = {
        limit: LIMIT,
        offset,
        change_type: filter.change_type || undefined,
        from_date:   filter.from_date   || undefined,
        to_date:     filter.to_date     || undefined,
      }
      const res = await getStockLogs(params)
      setLogs(res.data.data.items ?? [])
      setTotal(res.data.data.total)
    } catch { setError('Failed to load stock logs.') }
    finally { setLoading(false) }
  }, [offset, filter])

  useEffect(() => { load() }, [load])

  const setF = (key, val) => { setFilter((f) => ({ ...f, [key]: val })); setOffset(0) }

  return (
    <div className="p-4 sm:p-6 lg:p-8 space-y-5">
      <div>
        <h1 className="text-xl sm:text-2xl font-bold text-gray-900">Stock Logs</h1>
        <p className="text-gray-500 text-sm mt-0.5">{total} movements</p>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3">
        <select className="input w-auto" value={filter.change_type} onChange={(e) => setF('change_type', e.target.value)}>
          <option value="">All Types</option>
          <option value="sale">Sale</option>
          <option value="purchase">Purchase</option>
          <option value="adjustment">Adjustment</option>
          <option value="return">Return</option>
          <option value="transfer_in">Transfer In</option>
          <option value="transfer_out">Transfer Out</option>
        </select>
        <input type="date" className="input w-auto" value={filter.from_date}
          onChange={(e) => setF('from_date', e.target.value)} placeholder="From date" />
        <input type="date" className="input w-auto" value={filter.to_date}
          onChange={(e) => setF('to_date', e.target.value)} placeholder="To date" />
        {(filter.change_type || filter.from_date || filter.to_date) && (
          <button className="btn-secondary btn-sm" onClick={() => { setFilter({ change_type: '', from_date: '', to_date: '' }); setOffset(0) }}>
            Clear
          </button>
        )}
      </div>

      {error && <div className="p-3 bg-red-50 border border-red-200 rounded-xl text-sm text-red-700">{error}</div>}

      <div className="card overflow-hidden">
        {loading ? <LoadingSpinner className="py-16" /> : logs.length === 0 ? (
          <div className="py-16 text-center text-gray-400 text-sm">No stock movements found.</div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[560px]">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="table-th">Product</th>
                    <th className="table-th">Type</th>
                    <th className="table-th text-center">Before</th>
                    <th className="table-th text-center">Change</th>
                    <th className="table-th text-center">After</th>
                    <th className="table-th hidden md:table-cell">Note</th>
                    <th className="table-th hidden lg:table-cell">Date</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {logs.map((log) => (
                    <tr key={log.id} className="hover:bg-gray-50/70 transition-colors">
                      <td className="table-td">
                        <p className="font-medium text-gray-900 text-sm">{log.product?.name ?? '—'}</p>
                        <p className="text-xs text-gray-400 font-mono">{log.product?.sku ?? ''}</p>
                      </td>
                      <td className="table-td">
                        <Badge className={CHANGE_TYPE_COLORS[log.change_type] ?? 'bg-gray-100 text-gray-600'}>
                          {log.change_type?.replace('_', ' ')}
                        </Badge>
                      </td>
                      <td className="table-td text-center text-gray-500 text-sm tabular-nums">{log.quantity_before}</td>
                      <td className="table-td text-center font-semibold text-sm tabular-nums">
                        <span className={log.quantity_change >= 0 ? 'text-green-600' : 'text-red-600'}>
                          {log.quantity_change >= 0 ? '+' : ''}{log.quantity_change}
                        </span>
                      </td>
                      <td className="table-td text-center font-semibold text-gray-900 text-sm tabular-nums">{log.quantity_after}</td>
                      <td className="table-td text-xs text-gray-400 hidden md:table-cell max-w-[160px] truncate">
                        {log.note || '—'}
                      </td>
                      <td className="table-td text-xs text-gray-400 whitespace-nowrap hidden lg:table-cell">
                        {formatDateTime(log.created_at)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <Pagination total={total} limit={LIMIT} offset={offset} onPageChange={setOffset} />
          </>
        )}
      </div>
    </div>
  )
}
