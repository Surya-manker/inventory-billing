import { useEffect, useState, useCallback } from 'react'
import { getPurchaseOrders, createPurchaseOrder, updatePurchaseOrderStatus, deletePurchaseOrder } from '../api/purchaseOrders'
import { getVendors } from '../api/vendors'
import { getProducts } from '../api/products'
import Modal from '../components/common/Modal'
import ConfirmDialog from '../components/common/ConfirmDialog'
import LoadingSpinner from '../components/common/LoadingSpinner'
import Pagination from '../components/common/Pagination'
import Badge from '../components/common/Badge'
import { formatCurrency, formatDate, statusColors } from '../utils/formatters'

const LIMIT = 15
const STATUSES = ['', 'draft', 'confirmed', 'received']

const STATUS_ACTIONS = {
  draft:     [{ label: 'Confirm', next: 'confirmed', cls: 'text-blue-700 border-blue-200 hover:bg-blue-50' }],
  confirmed: [{ label: 'Received', next: 'received', cls: 'text-green-700 border-green-200 hover:bg-green-50' }],
  received:  [],
}

export default function PurchaseOrders() {
  const [items, setItems]         = useState([])
  const [total, setTotal]         = useState(0)
  const [offset, setOffset]       = useState(0)
  const [statusFilter, setStatus] = useState('')
  const [loading, setLoading]     = useState(true)
  const [error, setError]         = useState('')

  const [createOpen, setCreateOpen] = useState(false)
  const [vendors, setVendors]       = useState([])
  const [products, setProducts]     = useState([])
  const [form, setForm]             = useState({ vendor_id: '', notes: '', expected_at: '' })
  const [lineItems, setLineItems]   = useState([{ product_id: '', quantity: 1, unit_price: '' }])
  const [formError, setFormError]   = useState('')
  const [saving, setSaving]         = useState(false)

  const [viewPO, setViewPO]             = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)
  const [deleting, setDeleting]         = useState(false)

  const load = useCallback(async () => {
    setLoading(true); setError('')
    try {
      const res = await getPurchaseOrders({ limit: LIMIT, offset, status: statusFilter || undefined })
      setItems(res.data.data.items ?? [])
      setTotal(res.data.data.total)
    } catch { setError('Failed to load purchase orders.') }
    finally { setLoading(false) }
  }, [offset, statusFilter])

  useEffect(() => { load() }, [load])

  const openCreate = async () => {
    setFormError('')
    setForm({ vendor_id: '', notes: '', expected_at: '' })
    setLineItems([{ product_id: '', quantity: 1, unit_price: '' }])
    try {
      const [v, p] = await Promise.all([getVendors({ limit: 100 }), getProducts({ limit: 100 })])
      setVendors(v.data.data.items ?? [])
      setProducts(p.data.data.items ?? [])
    } catch { setFormError('Failed to load vendors/products.') }
    setCreateOpen(true)
  }

  const addLine    = () => setLineItems([...lineItems, { product_id: '', quantity: 1, unit_price: '' }])
  const removeLine = (i) => setLineItems(lineItems.filter((_, idx) => idx !== i))
  const updateLine = (i, key, val) =>
    setLineItems(lineItems.map((l, idx) => (idx === i ? { ...l, [key]: val } : l)))

  const poTotal = lineItems.reduce((s, l) => {
    return s + (parseFloat(l.unit_price) || 0) * (parseInt(l.quantity) || 0)
  }, 0)

  const handleCreate = async (e) => {
    e.preventDefault(); setFormError('')
    if (!form.vendor_id) return setFormError('Select a vendor.')
    const validLines = lineItems.filter((l) => l.product_id && l.quantity > 0 && l.unit_price)
    if (!validLines.length) return setFormError('Add at least one item with a price.')
    setSaving(true)
    try {
      await createPurchaseOrder({
        vendor_id:   parseInt(form.vendor_id),
        notes:       form.notes       || undefined,
        expected_at: form.expected_at ? new Date(form.expected_at).toISOString() : undefined,
        items: validLines.map((l) => ({
          product_id: l.product_id,
          quantity:   parseInt(l.quantity),
          unit_price: parseFloat(l.unit_price),
        })),
      })
      setCreateOpen(false); setOffset(0); load()
    } catch (err) {
      setFormError(err.response?.data?.message || 'Failed to create purchase order.')
    } finally { setSaving(false) }
  }

  const handleStatusChange = async (po, next) => {
    try { await updatePurchaseOrderStatus(po.id, next); load() }
    catch (err) { setError(err.response?.data?.message || 'Failed to update status.') }
  }

  const handleDelete = async () => {
    setDeleting(true)
    try { await deletePurchaseOrder(deleteTarget.id); setDeleteTarget(null); load() }
    catch (err) { setError(err.response?.data?.message || 'Failed to delete.'); setDeleteTarget(null) }
    finally { setDeleting(false) }
  }

  return (
    <div className="p-4 sm:p-6 lg:p-8 space-y-5">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h1 className="text-xl sm:text-2xl font-bold text-gray-900">Purchase Orders</h1>
          <p className="text-gray-500 text-sm mt-0.5">{total} total</p>
        </div>
        <button className="btn-primary flex-shrink-0" onClick={openCreate}>
          <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          <span className="hidden sm:inline">New Order</span>
          <span className="sm:hidden">New</span>
        </button>
      </div>

      {/* Status filter */}
      <div className="flex gap-2 overflow-x-auto pb-1 -mx-4 px-4 sm:mx-0 sm:px-0 scrollbar-none">
        {STATUSES.map((s) => (
          <button key={s}
            onClick={() => { setStatus(s); setOffset(0) }}
            className={`px-3 py-1.5 text-sm rounded-lg border whitespace-nowrap transition-colors flex-shrink-0 ${
              statusFilter === s ? 'bg-brand-600 text-white border-brand-600'
                                 : 'border-gray-200 text-gray-600 hover:bg-gray-50 bg-white'
            }`}>
            {s === '' ? 'All' : s.charAt(0).toUpperCase() + s.slice(1)}
          </button>
        ))}
      </div>

      {error && <div className="p-3 bg-red-50 border border-red-200 rounded-xl text-sm text-red-700">{error}</div>}

      <div className="card overflow-hidden">
        {loading ? <LoadingSpinner className="py-16" /> : items.length === 0 ? (
          <div className="py-16 text-center text-gray-400 text-sm">No purchase orders found.</div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[480px]">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="table-th">Order #</th>
                    <th className="table-th">Vendor</th>
                    <th className="table-th">Total</th>
                    <th className="table-th">Status</th>
                    <th className="table-th hidden md:table-cell">Expected</th>
                    <th className="table-th text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {items.map((po) => (
                    <tr key={po.id} className="hover:bg-gray-50/70 transition-colors">
                      <td className="table-td">
                        <button onClick={() => setViewPO(po)}
                          className="font-mono text-xs text-brand-600 hover:underline">
                          {po.order_number}
                        </button>
                      </td>
                      <td className="table-td font-medium text-gray-800">{po.vendor?.name ?? '—'}</td>
                      <td className="table-td font-semibold text-gray-900">{formatCurrency(po.total_amount ?? po.total ?? 0)}</td>
                      <td className="table-td">
                        <Badge className={statusColors[po.status]}>{po.status}</Badge>
                      </td>
                      <td className="table-td text-xs text-gray-400 hidden md:table-cell">
                        {po.expected_at ? formatDate(po.expected_at) : '—'}
                      </td>
                      <td className="table-td text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          {(STATUS_ACTIONS[po.status] ?? []).map(({ label, next, cls }) => (
                            <button key={next}
                              className={`btn-secondary btn-sm ${cls}`}
                              onClick={() => handleStatusChange(po, next)}>
                              {label}
                            </button>
                          ))}
                          {po.status === 'draft' && (
                            <button className="btn-danger btn-sm" onClick={() => setDeleteTarget(po)}>Delete</button>
                          )}
                        </div>
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

      {/* Create Modal */}
      <Modal open={createOpen} onClose={() => setCreateOpen(false)} title="New Purchase Order" size="xl">
        <form onSubmit={handleCreate} className="space-y-5">
          {formError && <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">{formError}</div>}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="sm:col-span-2">
              <label className="label">Vendor *</label>
              <select className="input" required value={form.vendor_id}
                onChange={(e) => setForm({ ...form, vendor_id: e.target.value })}>
                <option value="">Select vendor…</option>
                {vendors.filter((v) => v.is_active).map((v) => (
                  <option key={v.id} value={v.id}>{v.name}{v.company ? ` — ${v.company}` : ''}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="label">Expected Delivery</label>
              <input type="date" className="input" value={form.expected_at}
                min={new Date().toISOString().split('T')[0]}
                onChange={(e) => setForm({ ...form, expected_at: e.target.value })} />
            </div>
            <div>
              <label className="label">Notes</label>
              <input className="input" value={form.notes}
                onChange={(e) => setForm({ ...form, notes: e.target.value })} placeholder="Optional" />
            </div>
          </div>

          {/* Line items */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="label mb-0">Items *</label>
              <button type="button" className="btn-secondary btn-sm" onClick={addLine}>+ Add Row</button>
            </div>
            <div className="border border-gray-200 rounded-xl overflow-hidden">
              <table className="w-full text-sm">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="table-th">Product</th>
                    <th className="table-th w-20">Qty</th>
                    <th className="table-th w-28">Unit Price (₹)</th>
                    <th className="table-th w-28 text-right">Total</th>
                    <th className="w-8" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {lineItems.map((line, i) => {
                    const lineTotal = (parseFloat(line.unit_price) || 0) * (parseInt(line.quantity) || 0)
                    return (
                      <tr key={i}>
                        <td className="px-3 py-2">
                          <select className="input text-sm" value={line.product_id}
                            onChange={(e) => updateLine(i, 'product_id', e.target.value)}>
                            <option value="">Select…</option>
                            {products.map((p) => (
                              <option key={p.id} value={p.id}>{p.name}</option>
                            ))}
                          </select>
                        </td>
                        <td className="px-3 py-2">
                          <input type="number" className="input text-sm" min="1" value={line.quantity}
                            onChange={(e) => updateLine(i, 'quantity', e.target.value)} />
                        </td>
                        <td className="px-3 py-2">
                          <input type="number" className="input text-sm" min="0" step="0.01" placeholder="0.00"
                            value={line.unit_price}
                            onChange={(e) => updateLine(i, 'unit_price', e.target.value)} />
                        </td>
                        <td className="px-3 py-2 text-right font-medium">
                          {lineTotal > 0 ? formatCurrency(lineTotal) : '—'}
                        </td>
                        <td className="px-2 py-2">
                          {lineItems.length > 1 && (
                            <button type="button" onClick={() => removeLine(i)}
                              className="text-gray-300 hover:text-red-500 transition-colors">
                              <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                              </svg>
                            </button>
                          )}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
                <tfoot className="bg-gray-50">
                  <tr>
                    <td colSpan={3} className="px-3 py-2 text-right font-bold text-gray-900">Total</td>
                    <td className="px-3 py-2 text-right font-bold text-brand-700">{formatCurrency(poTotal)}</td>
                    <td />
                  </tr>
                </tfoot>
              </table>
            </div>
          </div>

          <div className="flex gap-3 pt-1">
            <button type="button" className="btn-secondary flex-1 sm:flex-none" onClick={() => setCreateOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary flex-1 sm:flex-none" disabled={saving}>
              {saving ? 'Creating…' : 'Create Order'}
            </button>
          </div>
        </form>
      </Modal>

      {/* View Modal */}
      <Modal open={!!viewPO} onClose={() => setViewPO(null)} title={viewPO?.order_number ?? 'Purchase Order'} size="lg">
        {viewPO && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3 text-sm">
              <div>
                <p className="text-xs text-gray-400 mb-0.5">Vendor</p>
                <p className="font-semibold text-gray-900">{viewPO.vendor?.name ?? '—'}</p>
              </div>
              <div>
                <p className="text-xs text-gray-400 mb-0.5">Status</p>
                <Badge className={statusColors[viewPO.status]}>{viewPO.status}</Badge>
              </div>
              <div>
                <p className="text-xs text-gray-400 mb-0.5">Expected</p>
                <p>{viewPO.expected_at ? formatDate(viewPO.expected_at) : '—'}</p>
              </div>
              <div>
                <p className="text-xs text-gray-400 mb-0.5">Total</p>
                <p className="font-semibold text-brand-700">{formatCurrency(viewPO.total_amount ?? viewPO.total ?? 0)}</p>
              </div>
            </div>
            <div className="border border-gray-200 rounded-xl overflow-hidden">
              <table className="w-full text-sm">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="table-th">Product</th>
                    <th className="table-th text-right">Qty</th>
                    <th className="table-th text-right">Unit Price</th>
                    <th className="table-th text-right">Total</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {viewPO.items?.map((item) => (
                    <tr key={item.id}>
                      <td className="table-td font-medium">{item.product?.name ?? '—'}</td>
                      <td className="table-td text-right">{item.quantity}</td>
                      <td className="table-td text-right text-gray-500">{formatCurrency(item.unit_price)}</td>
                      <td className="table-td text-right font-semibold">{formatCurrency(item.total ?? item.unit_price * item.quantity)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </Modal>

      <ConfirmDialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete} loading={deleting} title="Delete Purchase Order"
        message={`Delete order ${deleteTarget?.order_number}? This cannot be undone.`} />
    </div>
  )
}
