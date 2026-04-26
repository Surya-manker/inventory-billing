import { useEffect, useState, useCallback } from 'react'
import { getInvoices, createInvoice, updateInvoiceStatus, deleteInvoice } from '../api/invoices'
import { getCustomers } from '../api/customers'
import { getProducts } from '../api/products'
import { useAuth } from '../context/AuthContext'
import Modal from '../components/common/Modal'
import ConfirmDialog from '../components/common/ConfirmDialog'
import LoadingSpinner from '../components/common/LoadingSpinner'
import Pagination from '../components/common/Pagination'
import Badge from '../components/common/Badge'
import { formatCurrency, formatDate, statusColors } from '../utils/formatters'

const LIMIT = 10
const STATUSES = ['', 'pending', 'paid', 'canceled']

function InvoiceRow({ inv, isAdmin, onStatusChange, onDelete, onView }) {
  return (
    <tr className="hover:bg-gray-50/70 transition-colors">
      <td className="table-td">
        <button onClick={() => onView(inv)}
          className="font-mono text-xs text-brand-600 hover:underline leading-tight block">
          {inv.invoice_number}
        </button>
        {/* Show customer inline on mobile */}
        <p className="text-xs text-gray-500 mt-0.5 sm:hidden truncate max-w-[140px]">
          {inv.customer?.name ?? '—'}
        </p>
      </td>
      <td className="table-td hidden sm:table-cell">
        <p className="font-medium text-gray-800 leading-tight">{inv.customer?.name ?? '—'}</p>
        {inv.customer?.company && (
          <p className="text-xs text-gray-400 mt-0.5">{inv.customer.company}</p>
        )}
      </td>
      <td className="table-td font-semibold text-gray-900 whitespace-nowrap">
        {formatCurrency(inv.total_price)}
      </td>
      <td className="table-td">
        <Badge className={statusColors[inv.status]}>{inv.status}</Badge>
      </td>
      <td className="table-td text-gray-400 text-xs hidden md:table-cell whitespace-nowrap">
        {formatDate(inv.issued_at)}
      </td>
      {isAdmin && (
        <td className="table-td text-right">
          <div className="flex items-center justify-end gap-1">
            {inv.status === 'pending' && (
              <>
                <button
                  className="hidden sm:inline-flex btn-secondary btn-sm text-green-700 border-green-300 hover:bg-green-50"
                  onClick={() => onStatusChange(inv, 'paid')}
                >
                  Paid
                </button>
                <button
                  className="hidden sm:inline-flex btn-secondary btn-sm text-gray-500"
                  onClick={() => onStatusChange(inv, 'canceled')}
                >
                  Cancel
                </button>
                {/* Mobile: compact icon buttons */}
                <button
                  className="sm:hidden p-1.5 rounded-lg text-green-600 hover:bg-green-50 border border-green-200"
                  onClick={() => onStatusChange(inv, 'paid')}
                  title="Mark Paid"
                >
                  <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
                  </svg>
                </button>
              </>
            )}
            <button
              className="btn-danger btn-sm"
              onClick={() => onDelete(inv)}
            >
              <span className="hidden sm:inline">Delete</span>
              <svg className="h-3.5 w-3.5 sm:hidden" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </td>
      )}
    </tr>
  )
}

export default function Invoices() {
  const { isAdmin } = useAuth()
  const [items, setItems]           = useState([])
  const [total, setTotal]           = useState(0)
  const [offset, setOffset]         = useState(0)
  const [statusFilter, setStatusFilter] = useState('')
  const [loading, setLoading]       = useState(true)
  const [error, setError]           = useState('')

  const [createOpen, setCreateOpen] = useState(false)
  const [customers, setCustomers]   = useState([])
  const [products, setProducts]     = useState([])
  const [invoiceForm, setInvoiceForm] = useState({ customer_id: '', notes: '', discount: '', due_at: '' })
  const [lineItems, setLineItems]   = useState([{ product_id: '', quantity: 1 }])
  const [formError, setFormError]   = useState('')
  const [saving, setSaving]         = useState(false)

  const [viewInvoice, setViewInvoice]   = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)
  const [deleting, setDeleting]         = useState(false)

  const load = useCallback(async () => {
    setLoading(true); setError('')
    try {
      const res = await getInvoices({ limit: LIMIT, offset, status: statusFilter || undefined })
      setItems(res.data.data.items ?? [])
      setTotal(res.data.data.total)
    } catch { setError('Failed to load invoices.') }
    finally { setLoading(false) }
  }, [offset, statusFilter])

  useEffect(() => { load() }, [load])

  const openCreate = async () => {
    setFormError('')
    setInvoiceForm({ customer_id: '', notes: '', discount: '', due_at: '' })
    setLineItems([{ product_id: '', quantity: 1 }])
    try {
      const [c, p] = await Promise.all([getCustomers({ limit: 100 }), getProducts({ limit: 100 })])
      setCustomers(c.data.data.items ?? [])
      setProducts(p.data.data.items ?? [])
    } catch { setFormError('Failed to load customers/products.') }
    setCreateOpen(true)
  }

  const addLine   = () => setLineItems([...lineItems, { product_id: '', quantity: 1 }])
  const removeLine = (i) => setLineItems(lineItems.filter((_, idx) => idx !== i))
  const updateLine = (i, key, val) =>
    setLineItems(lineItems.map((l, idx) => (idx === i ? { ...l, [key]: val } : l)))

  const subtotal = lineItems.reduce((sum, l) => {
    const p = products.find((p) => p.id === l.product_id)
    return sum + (p ? p.price * (parseInt(l.quantity) || 0) : 0)
  }, 0)

  const handleCreate = async (e) => {
    e.preventDefault(); setFormError('')
    if (!invoiceForm.customer_id) return setFormError('Please select a customer.')
    const validLines = lineItems.filter((l) => l.product_id && l.quantity > 0)
    if (!validLines.length) return setFormError('Add at least one product.')
    if (!invoiceForm.due_at) return setFormError('Please select a due date.')
    setSaving(true)
    try {
      await createInvoice({
        customer_id: parseInt(invoiceForm.customer_id),
        items: validLines.map((l) => ({ product_id: l.product_id, quantity: parseInt(l.quantity) })),
        discount: invoiceForm.discount ? parseFloat(invoiceForm.discount) : 0,
        notes: invoiceForm.notes || undefined,
        due_at: new Date(invoiceForm.due_at).toISOString(),
      })
      setCreateOpen(false); setOffset(0); load()
    } catch (err) {
      setFormError(err.response?.data?.message || 'Failed to create invoice.')
    } finally { setSaving(false) }
  }

  const handleStatusChange = async (inv, status) => {
    try { await updateInvoiceStatus(inv.id, status); load() }
    catch (err) { setError(err.response?.data?.message || 'Failed to update status.') }
  }

  const handleDelete = async () => {
    setDeleting(true)
    try { await deleteInvoice(deleteTarget.id); setDeleteTarget(null); load() }
    catch (err) { setError(err.response?.data?.message || 'Failed to delete invoice.'); setDeleteTarget(null) }
    finally { setDeleting(false) }
  }

  return (
    <div className="p-4 sm:p-6 lg:p-8 space-y-5">

      {/* Header */}
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-xl sm:text-2xl font-bold text-gray-900">Invoices</h1>
          <p className="text-gray-500 text-sm mt-0.5">{total} total</p>
        </div>
        <button className="btn-primary flex-shrink-0" onClick={openCreate}>
          <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          <span className="hidden sm:inline">New Invoice</span>
          <span className="sm:hidden">New</span>
        </button>
      </div>

      {/* Status filter — scrollable on small screens */}
      <div className="flex gap-2 overflow-x-auto pb-1 -mx-4 px-4 sm:mx-0 sm:px-0 sm:flex-wrap scrollbar-none">
        {STATUSES.map((s) => (
          <button
            key={s}
            onClick={() => { setStatusFilter(s); setOffset(0) }}
            className={`px-3 py-1.5 text-sm rounded-lg border whitespace-nowrap transition-colors flex-shrink-0 ${
              statusFilter === s
                ? 'bg-brand-600 text-white border-brand-600'
                : 'border-gray-200 text-gray-600 hover:bg-gray-50 bg-white'
            }`}
          >
            {s === '' ? 'All' : s.charAt(0).toUpperCase() + s.slice(1)}
          </button>
        ))}
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-xl text-sm text-red-700">{error}</div>
      )}

      {/* Table */}
      <div className="card overflow-hidden">
        {loading ? (
          <LoadingSpinner className="py-16" />
        ) : items.length === 0 ? (
          <div className="py-16 text-center text-gray-400 text-sm px-4">No invoices found.</div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[380px]">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="table-th">Invoice #</th>
                    <th className="table-th hidden sm:table-cell">Customer</th>
                    <th className="table-th">Total</th>
                    <th className="table-th">Status</th>
                    <th className="table-th hidden md:table-cell">Date</th>
                    {isAdmin && <th className="table-th text-right">Actions</th>}
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {items.map((inv) => (
                    <InvoiceRow
                      key={inv.id}
                      inv={inv}
                      isAdmin={isAdmin}
                      onStatusChange={handleStatusChange}
                      onDelete={setDeleteTarget}
                      onView={setViewInvoice}
                    />
                  ))}
                </tbody>
              </table>
            </div>
            <Pagination total={total} limit={LIMIT} offset={offset} onPageChange={setOffset} />
          </>
        )}
      </div>

      {/* ── Create Invoice Modal ── */}
      <Modal open={createOpen} onClose={() => setCreateOpen(false)} title="Create Invoice" size="xl">
        <form onSubmit={handleCreate} className="space-y-5">
          {formError && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">{formError}</div>
          )}

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="sm:col-span-2">
              <label className="label">Customer *</label>
              <select className="input" required value={invoiceForm.customer_id}
                onChange={(e) => setInvoiceForm({ ...invoiceForm, customer_id: e.target.value })}>
                <option value="">Select a customer…</option>
                {customers.map((c) => (
                  <option key={c.id} value={c.id}>{c.name}{c.company ? ` — ${c.company}` : ''}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="label">Due Date *</label>
              <input type="date" className="input" required value={invoiceForm.due_at}
                min={new Date().toISOString().split('T')[0]}
                onChange={(e) => setInvoiceForm({ ...invoiceForm, due_at: e.target.value })} />
            </div>
            <div>
              <label className="label">Discount (₹)</label>
              <input type="number" className="input" min="0" step="0.01" placeholder="0.00"
                value={invoiceForm.discount}
                onChange={(e) => setInvoiceForm({ ...invoiceForm, discount: e.target.value })} />
            </div>
            <div className="sm:col-span-2">
              <label className="label">Notes</label>
              <textarea className="input" rows={2} value={invoiceForm.notes}
                onChange={(e) => setInvoiceForm({ ...invoiceForm, notes: e.target.value })}
                placeholder="Optional notes" />
            </div>
          </div>

          {/* Line items */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="label mb-0">Products *</label>
              <button type="button" className="btn-secondary btn-sm" onClick={addLine}>+ Add Row</button>
            </div>

            {/* Mobile: card list */}
            <div className="sm:hidden space-y-3">
              {lineItems.map((line, i) => {
                const prod = products.find((p) => p.id === line.product_id)
                const sub  = prod ? prod.price * (parseInt(line.quantity) || 0) : 0
                return (
                  <div key={i} className="border border-gray-200 rounded-xl p-3 space-y-3 bg-gray-50">
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-medium text-gray-500">Item {i + 1}</span>
                      {lineItems.length > 1 && (
                        <button type="button" onClick={() => removeLine(i)}
                          className="text-gray-400 hover:text-red-500 transition-colors p-1">
                          <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                          </svg>
                        </button>
                      )}
                    </div>
                    <div>
                      <label className="label text-xs">Product</label>
                      <select className="input text-sm" value={line.product_id}
                        onChange={(e) => updateLine(i, 'product_id', e.target.value)}>
                        <option value="">Select product…</option>
                        {products.map((p) => (
                          <option key={p.id} value={p.id}>{p.name} (Stock: {p.stock})</option>
                        ))}
                      </select>
                    </div>
                    <div className="flex gap-3 items-end">
                      <div className="w-24 flex-shrink-0">
                        <label className="label text-xs">Qty</label>
                        <input type="number" className="input text-sm" min="1" value={line.quantity}
                          onChange={(e) => updateLine(i, 'quantity', e.target.value)} />
                      </div>
                      <div className="flex-1 text-right">
                        <p className="text-xs text-gray-400 mb-1">Subtotal</p>
                        <p className="font-semibold text-gray-900 text-sm">
                          {sub > 0 ? formatCurrency(sub) : '—'}
                        </p>
                      </div>
                    </div>
                  </div>
                )
              })}
              {/* Totals */}
              <div className="border border-gray-200 rounded-xl p-3 bg-white space-y-1.5">
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Subtotal</span>
                  <span className="font-medium">{formatCurrency(subtotal)}</span>
                </div>
                {invoiceForm.discount > 0 && (
                  <div className="flex justify-between text-sm text-red-500">
                    <span>Discount</span>
                    <span>− {formatCurrency(parseFloat(invoiceForm.discount))}</span>
                  </div>
                )}
                <div className="flex justify-between text-sm font-bold border-t border-gray-100 pt-1.5">
                  <span>Est. Total</span>
                  <span className="text-brand-700">
                    {formatCurrency(Math.max(0, subtotal - parseFloat(invoiceForm.discount || 0)))}
                  </span>
                </div>
              </div>
            </div>

            {/* Desktop: table */}
            <div className="hidden sm:block border border-gray-200 rounded-xl overflow-hidden">
              <table className="w-full text-sm">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="table-th">Product</th>
                    <th className="table-th w-24">Qty</th>
                    <th className="table-th w-28 text-right">Unit Price</th>
                    <th className="table-th w-28 text-right">Subtotal</th>
                    <th className="w-8" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {lineItems.map((line, i) => {
                    const prod = products.find((p) => p.id === line.product_id)
                    const sub  = prod ? prod.price * (parseInt(line.quantity) || 0) : 0
                    return (
                      <tr key={i}>
                        <td className="px-3 py-2">
                          <select className="input text-sm" value={line.product_id}
                            onChange={(e) => updateLine(i, 'product_id', e.target.value)}>
                            <option value="">Select product…</option>
                            {products.map((p) => (
                              <option key={p.id} value={p.id}>{p.name} (Stock: {p.stock})</option>
                            ))}
                          </select>
                        </td>
                        <td className="px-3 py-2">
                          <input type="number" className="input text-sm" min="1" value={line.quantity}
                            onChange={(e) => updateLine(i, 'quantity', e.target.value)} />
                        </td>
                        <td className="px-3 py-2 text-right text-gray-500">
                          {prod ? formatCurrency(prod.price) : '—'}
                        </td>
                        <td className="px-3 py-2 text-right font-medium">
                          {sub > 0 ? formatCurrency(sub) : '—'}
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
                <tfoot className="bg-gray-50 text-sm">
                  <tr>
                    <td colSpan={3} className="px-3 py-2 text-right text-gray-500">Subtotal</td>
                    <td className="px-3 py-2 text-right font-semibold">{formatCurrency(subtotal)}</td>
                    <td />
                  </tr>
                  {invoiceForm.discount > 0 && (
                    <tr>
                      <td colSpan={3} className="px-3 py-2 text-right text-red-500">Discount</td>
                      <td className="px-3 py-2 text-right text-red-500">
                        − {formatCurrency(parseFloat(invoiceForm.discount))}
                      </td>
                      <td />
                    </tr>
                  )}
                  <tr>
                    <td colSpan={3} className="px-3 py-2 text-right font-bold text-gray-900">Est. Total</td>
                    <td className="px-3 py-2 text-right font-bold text-brand-700">
                      {formatCurrency(Math.max(0, subtotal - parseFloat(invoiceForm.discount || 0)))}
                    </td>
                    <td />
                  </tr>
                </tfoot>
              </table>
            </div>
          </div>

          <div className="flex gap-3 pt-1">
            <button type="button" className="btn-secondary flex-1 sm:flex-none"
              onClick={() => setCreateOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary flex-1 sm:flex-none" disabled={saving}>
              {saving ? 'Creating…' : 'Create Invoice'}
            </button>
          </div>
        </form>
      </Modal>

      {/* ── View Invoice Modal ── */}
      <Modal open={!!viewInvoice} onClose={() => setViewInvoice(null)}
        title={viewInvoice?.invoice_number ?? 'Invoice'} size="lg">
        {viewInvoice && (
          <div className="space-y-5">
            <div className="grid grid-cols-2 gap-3 text-sm">
              <div className="col-span-2 sm:col-span-1">
                <p className="text-xs text-gray-400 mb-0.5">Customer</p>
                <p className="font-semibold text-gray-900">{viewInvoice.customer?.name}</p>
                {viewInvoice.customer?.company && (
                  <p className="text-xs text-gray-400">{viewInvoice.customer.company}</p>
                )}
              </div>
              <div>
                <p className="text-xs text-gray-400 mb-0.5">Status</p>
                <Badge className={statusColors[viewInvoice.status]}>{viewInvoice.status}</Badge>
              </div>
              <div>
                <p className="text-xs text-gray-400 mb-0.5">Issued</p>
                <p className="font-medium">{formatDate(viewInvoice.issued_at)}</p>
              </div>
              <div>
                <p className="text-xs text-gray-400 mb-0.5">Due</p>
                <p className="font-medium">{formatDate(viewInvoice.due_at)}</p>
              </div>
              {viewInvoice.notes && (
                <div className="col-span-2 p-3 bg-gray-50 rounded-lg">
                  <p className="text-xs text-gray-400 mb-0.5">Notes</p>
                  <p className="text-gray-700 text-sm">{viewInvoice.notes}</p>
                </div>
              )}
            </div>

            <div className="border border-gray-200 rounded-xl overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-sm min-w-[320px]">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="table-th">Product</th>
                      <th className="table-th text-right">Qty</th>
                      <th className="table-th text-right hidden sm:table-cell">Unit</th>
                      <th className="table-th text-right">Total</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {viewInvoice.items?.map((item) => (
                      <tr key={item.id}>
                        <td className="table-td">
                          <p className="font-medium">{item.product?.name ?? '—'}</p>
                          <p className="text-xs text-gray-400 sm:hidden">{formatCurrency(item.unit_price)} each</p>
                        </td>
                        <td className="table-td text-right">{item.quantity}</td>
                        <td className="table-td text-right hidden sm:table-cell text-gray-500">
                          {formatCurrency(item.unit_price)}
                        </td>
                        <td className="table-td text-right font-semibold">{formatCurrency(item.total)}</td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot className="bg-gray-50 text-sm">
                    <tr>
                      <td colSpan={3} className="px-4 py-2 text-right text-gray-500">Subtotal</td>
                      <td className="px-4 py-2 text-right">{formatCurrency(viewInvoice.sub_total)}</td>
                    </tr>
                    {viewInvoice.discount > 0 && (
                      <tr>
                        <td colSpan={3} className="px-4 py-2 text-right text-red-500">Discount</td>
                        <td className="px-4 py-2 text-right text-red-500">
                          − {formatCurrency(viewInvoice.discount)}
                        </td>
                      </tr>
                    )}
                    {viewInvoice.tax_amount > 0 && (
                      <tr>
                        <td colSpan={3} className="px-4 py-2 text-right text-gray-500">
                          Tax ({viewInvoice.cgst_amount > 0 ? 'CGST+SGST' : 'IGST'})
                        </td>
                        <td className="px-4 py-2 text-right">{formatCurrency(viewInvoice.tax_amount)}</td>
                      </tr>
                    )}
                    <tr>
                      <td colSpan={3} className="px-4 py-2 text-right font-bold text-gray-900">Total</td>
                      <td className="px-4 py-2 text-right font-bold text-brand-700">
                        {formatCurrency(viewInvoice.total_price)}
                      </td>
                    </tr>
                  </tfoot>
                </table>
              </div>
            </div>
          </div>
        )}
      </Modal>

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        loading={deleting}
        title="Delete Invoice"
        message={`Delete invoice ${deleteTarget?.invoice_number}? This cannot be undone.`}
      />
    </div>
  )
}
