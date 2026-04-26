import { useEffect, useState, useCallback, useRef } from 'react'
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

function InvoiceRow({ inv, isAdmin, onStatusChange, onDelete, onView }) {
  return (
    <tr className="hover:bg-gray-50 transition-colors">
      <td className="table-td">
        <button onClick={() => onView(inv)} className="font-mono text-xs text-brand-600 hover:underline">
          {inv.invoice_number}
        </button>
      </td>
      <td className="table-td font-medium">{inv.customer?.name ?? '—'}</td>
      <td className="table-td">{inv.items?.length ?? 0} items</td>
      <td className="table-td font-semibold">{formatCurrency(inv.total_price)}</td>
      <td className="table-td">
        <Badge className={statusColors[inv.status]}>{inv.status}</Badge>
      </td>
      <td className="table-td text-gray-400">{formatDate(inv.issued_at)}</td>
      {isAdmin && (
        <td className="table-td text-right">
          <div className="flex items-center justify-end gap-2">
            {inv.status === 'pending' && (
              <>
                <button className="btn-secondary btn-sm text-green-700 border-green-300 hover:bg-green-50"
                  onClick={() => onStatusChange(inv, 'paid')}>
                  Mark Paid
                </button>
                <button className="btn-secondary btn-sm text-gray-500"
                  onClick={() => onStatusChange(inv, 'canceled')}>
                  Cancel
                </button>
              </>
            )}
            <button className="btn-danger btn-sm" onClick={() => onDelete(inv)}>Delete</button>
          </div>
        </td>
      )}
    </tr>
  )
}

export default function Invoices() {
  const { isAdmin } = useAuth()
  const [items, setItems] = useState([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [statusFilter, setStatusFilter] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [createOpen, setCreateOpen] = useState(false)
  const [customers, setCustomers] = useState([])
  const [products, setProducts] = useState([])
  const [invoiceForm, setInvoiceForm] = useState({
    customer_id: '',
    notes: '',
    discount: '',
    due_at: '',
  })
  const [lineItems, setLineItems] = useState([{ product_id: '', quantity: 1 }])
  const [formError, setFormError] = useState('')
  const [saving, setSaving] = useState(false)

  const [viewInvoice, setViewInvoice] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)
  const [deleting, setDeleting] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const res = await getInvoices({
        limit: LIMIT,
        offset,
        status: statusFilter || undefined,
      })
      setItems(res.data.data.items ?? [])
      setTotal(res.data.data.total)
    } catch {
      setError('Failed to load invoices.')
    } finally {
      setLoading(false)
    }
  }, [offset, statusFilter])

  useEffect(() => { load() }, [load])

  const openCreate = async () => {
    setFormError('')
    setInvoiceForm({ customer_id: '', notes: '', discount: '', due_at: '' })
    setLineItems([{ product_id: '', quantity: 1 }])
    try {
      const [c, p] = await Promise.all([
        getCustomers({ limit: 100 }),
        getProducts({ limit: 100 }),
      ])
      setCustomers(c.data.data.items ?? [])
      setProducts(p.data.data.items ?? [])
    } catch {
      setFormError('Failed to load customers/products.')
    }
    setCreateOpen(true)
  }

  const addLine = () => setLineItems([...lineItems, { product_id: '', quantity: 1 }])
  const removeLine = (i) => setLineItems(lineItems.filter((_, idx) => idx !== i))
  const updateLine = (i, key, val) =>
    setLineItems(lineItems.map((l, idx) => (idx === i ? { ...l, [key]: val } : l)))

  const subtotal = lineItems.reduce((sum, l) => {
    const p = products.find((p) => p.id === l.product_id)
    return sum + (p ? p.price * (parseInt(l.quantity) || 0) : 0)
  }, 0)

  const handleCreate = async (e) => {
    e.preventDefault()
    setFormError('')
    if (!invoiceForm.customer_id) return setFormError('Please select a customer.')
    const validLines = lineItems.filter((l) => l.product_id && l.quantity > 0)
    if (validLines.length === 0) return setFormError('Add at least one product.')
    if (!invoiceForm.due_at) return setFormError('Please select a due date.')

    setSaving(true)
    try {
      await createInvoice({
        customer_id: parseInt(invoiceForm.customer_id),
        items: validLines.map((l) => ({
          product_id: l.product_id,
          quantity: parseInt(l.quantity),
        })),
        discount: invoiceForm.discount ? parseFloat(invoiceForm.discount) : 0,
        notes: invoiceForm.notes || undefined,
        due_at: new Date(invoiceForm.due_at).toISOString(),
      })
      setCreateOpen(false)
      setOffset(0)
      load()
    } catch (err) {
      setFormError(err.response?.data?.message || 'Failed to create invoice.')
    } finally {
      setSaving(false)
    }
  }

  const handleStatusChange = async (inv, status) => {
    try {
      await updateInvoiceStatus(inv.id, status)
      load()
    } catch (err) {
      setError(err.response?.data?.message || 'Failed to update status.')
    }
  }

  const handleDelete = async () => {
    setDeleting(true)
    try {
      await deleteInvoice(deleteTarget.id)
      setDeleteTarget(null)
      load()
    } catch (err) {
      setError(err.response?.data?.message || 'Failed to delete invoice.')
      setDeleteTarget(null)
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div className="p-8 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Invoices</h1>
          <p className="text-gray-500 text-sm mt-1">{total} invoices total</p>
        </div>
        <button className="btn-primary" onClick={openCreate}>
          <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          New Invoice
        </button>
      </div>

      {/* Filters */}
      <div className="flex gap-2">
        {['', 'pending', 'paid', 'canceled'].map((s) => (
          <button
            key={s}
            onClick={() => { setStatusFilter(s); setOffset(0) }}
            className={`px-3 py-1.5 text-sm rounded-lg border transition-colors ${
              statusFilter === s
                ? 'bg-brand-600 text-white border-brand-600'
                : 'border-gray-300 text-gray-600 hover:bg-gray-50'
            }`}
          >
            {s === '' ? 'All' : s.charAt(0).toUpperCase() + s.slice(1)}
          </button>
        ))}
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">{error}</div>
      )}

      <div className="card overflow-hidden">
        {loading ? (
          <LoadingSpinner className="py-20" />
        ) : items.length === 0 ? (
          <div className="py-20 text-center text-gray-400 text-sm">No invoices found.</div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="table-th">Invoice #</th>
                    <th className="table-th">Customer</th>
                    <th className="table-th">Items</th>
                    <th className="table-th">Total</th>
                    <th className="table-th">Status</th>
                    <th className="table-th">Date</th>
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

      {/* Create Invoice Modal */}
      <Modal open={createOpen} onClose={() => setCreateOpen(false)} title="Create Invoice" size="xl">
        <form onSubmit={handleCreate} className="space-y-5">
          {formError && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">{formError}</div>
          )}

          <div className="grid grid-cols-2 gap-4">
            <div className="col-span-2">
              <label className="label">Customer *</label>
              <select className="input" required value={invoiceForm.customer_id}
                onChange={(e) => setInvoiceForm({ ...invoiceForm, customer_id: e.target.value })}>
                <option value="">Select a customer…</option>
                {customers.map((c) => (
                  <option key={c.id} value={c.id}>{c.name} {c.company ? `— ${c.company}` : ''}</option>
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
              <input type="number" className="input" min="0" step="0.01"
                value={invoiceForm.discount}
                onChange={(e) => setInvoiceForm({ ...invoiceForm, discount: e.target.value })}
                placeholder="0.00" />
            </div>
            <div className="col-span-2">
              <label className="label">Notes</label>
              <textarea className="input" rows={2} value={invoiceForm.notes}
                onChange={(e) => setInvoiceForm({ ...invoiceForm, notes: e.target.value })} />
            </div>
          </div>

          {/* Line items */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="label mb-0">Products *</label>
              <button type="button" className="btn-secondary btn-sm" onClick={addLine}>
                + Add Row
              </button>
            </div>
            <div className="border border-gray-200 rounded-lg overflow-hidden">
              <table className="w-full text-sm">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="table-th">Product</th>
                    <th className="table-th w-28">Qty</th>
                    <th className="table-th w-32 text-right">Unit Price</th>
                    <th className="table-th w-32 text-right">Subtotal</th>
                    <th className="w-10" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {lineItems.map((line, i) => {
                    const prod = products.find((p) => p.id === line.product_id)
                    const sub = prod ? prod.price * (parseInt(line.quantity) || 0) : 0
                    return (
                      <tr key={i}>
                        <td className="px-4 py-2">
                          <select className="input"
                            value={line.product_id}
                            onChange={(e) => updateLine(i, 'product_id', e.target.value)}>
                            <option value="">Select product…</option>
                            {products.map((p) => (
                              <option key={p.id} value={p.id}>
                                {p.name} (Stock: {p.stock})
                              </option>
                            ))}
                          </select>
                        </td>
                        <td className="px-4 py-2">
                          <input type="number" className="input" min="1" value={line.quantity}
                            onChange={(e) => updateLine(i, 'quantity', e.target.value)} />
                        </td>
                        <td className="px-4 py-2 text-right text-gray-500">
                          {prod ? formatCurrency(prod.price) : '—'}
                        </td>
                        <td className="px-4 py-2 text-right font-medium">
                          {sub > 0 ? formatCurrency(sub) : '—'}
                        </td>
                        <td className="px-2 py-2">
                          {lineItems.length > 1 && (
                            <button type="button" onClick={() => removeLine(i)}
                              className="text-gray-400 hover:text-red-500 transition-colors">
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
                    <td colSpan={3} className="px-4 py-2 text-right text-sm font-medium text-gray-600">
                      Subtotal
                    </td>
                    <td className="px-4 py-2 text-right font-semibold">{formatCurrency(subtotal)}</td>
                    <td />
                  </tr>
                  {invoiceForm.discount > 0 && (
                    <tr>
                      <td colSpan={3} className="px-4 py-2 text-right text-sm text-red-600">
                        Discount
                      </td>
                      <td className="px-4 py-2 text-right text-red-600">
                        − {formatCurrency(parseFloat(invoiceForm.discount))}
                      </td>
                      <td />
                    </tr>
                  )}
                  <tr>
                    <td colSpan={3} className="px-4 py-2 text-right text-sm font-bold text-gray-900">
                      Estimated Total
                    </td>
                    <td className="px-4 py-2 text-right font-bold text-brand-700">
                      {formatCurrency(Math.max(0, subtotal - parseFloat(invoiceForm.discount || 0)))}
                    </td>
                    <td />
                  </tr>
                </tfoot>
              </table>
            </div>
          </div>

          <div className="flex justify-end gap-3 pt-2">
            <button type="button" className="btn-secondary" onClick={() => setCreateOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary" disabled={saving}>
              {saving ? 'Creating…' : 'Create Invoice'}
            </button>
          </div>
        </form>
      </Modal>

      {/* View Invoice Modal */}
      <Modal
        open={!!viewInvoice}
        onClose={() => setViewInvoice(null)}
        title={`Invoice ${viewInvoice?.invoice_number}`}
        size="lg"
      >
        {viewInvoice && (
          <div className="space-y-5">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-gray-500">Customer</p>
                <p className="font-medium">{viewInvoice.customer?.name}</p>
                {viewInvoice.customer?.company && (
                  <p className="text-gray-400 text-xs">{viewInvoice.customer.company}</p>
                )}
              </div>
              <div>
                <p className="text-gray-500">Status</p>
                <Badge className={`mt-1 ${statusColors[viewInvoice.status]}`}>{viewInvoice.status}</Badge>
              </div>
              <div>
                <p className="text-gray-500">Issued</p>
                <p className="font-medium">{formatDate(viewInvoice.issued_at)}</p>
              </div>
              <div>
                <p className="text-gray-500">Due</p>
                <p className="font-medium">{formatDate(viewInvoice.due_at)}</p>
              </div>
              {viewInvoice.notes && (
                <div className="col-span-2">
                  <p className="text-gray-500">Notes</p>
                  <p className="text-gray-700">{viewInvoice.notes}</p>
                </div>
              )}
            </div>

            <table className="w-full text-sm border border-gray-200 rounded-lg overflow-hidden">
              <thead className="bg-gray-50">
                <tr>
                  <th className="table-th">Product</th>
                  <th className="table-th text-right">Qty</th>
                  <th className="table-th text-right">Unit Price</th>
                  <th className="table-th text-right">Total</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {viewInvoice.items?.map((item) => (
                  <tr key={item.id}>
                    <td className="table-td">{item.product?.name ?? '—'}</td>
                    <td className="table-td text-right">{item.quantity}</td>
                    <td className="table-td text-right">{formatCurrency(item.unit_price)}</td>
                    <td className="table-td text-right font-medium">{formatCurrency(item.total)}</td>
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
                    <td className="px-4 py-2 text-right text-red-500">− {formatCurrency(viewInvoice.discount)}</td>
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
