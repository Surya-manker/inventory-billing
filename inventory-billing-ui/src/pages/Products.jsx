import { useEffect, useState, useCallback } from 'react'
import { getProducts, createProduct, deleteProduct, adjustStock } from '../api/products'
import { useAuth } from '../context/AuthContext'
import Modal from '../components/common/Modal'
import ConfirmDialog from '../components/common/ConfirmDialog'
import LoadingSpinner from '../components/common/LoadingSpinner'
import Pagination from '../components/common/Pagination'
import { formatCurrency } from '../utils/formatters'

const LIMIT = 10
const EMPTY = { name: '', description: '', sku: '', price: '', cost_price: '', stock: '' }

export default function Products() {
  const { isAdmin } = useAuth()
  const [items, setItems] = useState([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [addOpen, setAddOpen] = useState(false)
  const [form, setForm] = useState(EMPTY)
  const [formError, setFormError] = useState('')
  const [saving, setSaving] = useState(false)

  const [deleteTarget, setDeleteTarget] = useState(null)
  const [deleting, setDeleting] = useState(false)

  const [stockTarget, setStockTarget] = useState(null)
  const [stockForm, setStockForm] = useState({ quantity_change: '', change_type: 'adjustment', note: '' })
  const [stockSaving, setStockSaving] = useState(false)
  const [stockError, setStockError] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const res = await getProducts({ limit: LIMIT, offset, search: search || undefined })
      setItems(res.data.data.items ?? [])
      setTotal(res.data.data.total)
    } catch {
      setError('Failed to load products.')
    } finally {
      setLoading(false)
    }
  }, [offset, search])

  useEffect(() => { load() }, [load])

  const handleSearch = (e) => { setSearch(e.target.value); setOffset(0) }

  const handleAdd = async (e) => {
    e.preventDefault()
    setFormError('')
    setSaving(true)
    try {
      await createProduct({
        name:        form.name,
        description: form.description  || undefined,
        sku:         form.sku,
        price:       parseFloat(form.price),
        cost_price:  form.cost_price ? parseFloat(form.cost_price) : 0,
        stock:       form.stock ? parseInt(form.stock) : 0,
      })
      setAddOpen(false); setForm(EMPTY); setOffset(0); load()
    } catch (err) {
      setFormError(err.response?.data?.message || 'Failed to create product.')
    } finally { setSaving(false) }
  }

  const handleDelete = async () => {
    setDeleting(true)
    try {
      await deleteProduct(deleteTarget.id)
      setDeleteTarget(null); load()
    } catch (err) {
      setError(err.response?.data?.message || 'Failed to delete product.')
      setDeleteTarget(null)
    } finally { setDeleting(false) }
  }

  const handleStockAdjust = async (e) => {
    e.preventDefault()
    setStockError('')
    setStockSaving(true)
    try {
      await adjustStock(stockTarget.id, {
        quantity_change: parseInt(stockForm.quantity_change),
        change_type: stockForm.change_type,
        note: stockForm.note || undefined,
      })
      setStockTarget(null)
      setStockForm({ quantity_change: '', change_type: 'adjustment', note: '' })
      load()
    } catch (err) {
      setStockError(err.response?.data?.message || 'Failed to adjust stock.')
    } finally { setStockSaving(false) }
  }

  return (
    <div className="p-4 sm:p-6 lg:p-8 space-y-5">

      {/* ── Header ── */}
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-xl sm:text-2xl font-bold text-gray-900 truncate">Products</h1>
          <p className="text-gray-500 text-sm mt-0.5">{total} total</p>
        </div>
        {isAdmin && (
          <button className="btn-primary flex-shrink-0" onClick={() => setAddOpen(true)}>
            <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            <span className="hidden sm:inline">Add Product</span>
            <span className="sm:hidden">Add</span>
          </button>
        )}
      </div>

      {/* ── Search ── */}
      <div className="relative w-full sm:max-w-xs">
        <svg className="absolute left-3 top-2.5 h-4 w-4 text-gray-400 pointer-events-none" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
        <input type="text" className="input pl-9" placeholder="Search products…" value={search} onChange={handleSearch} />
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-xl text-sm text-red-700">{error}</div>
      )}

      {/* ── Table ── */}
      <div className="card overflow-hidden">
        {loading ? (
          <LoadingSpinner className="py-16" />
        ) : items.length === 0 ? (
          <div className="py-16 text-center text-gray-400 text-sm px-4">
            {search ? 'No products match your search.' : 'No products yet. Add your first product.'}
          </div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[480px]">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="table-th">Name</th>
                    <th className="table-th hidden sm:table-cell">SKU</th>
                    <th className="table-th">Price</th>
                    <th className="table-th hidden sm:table-cell">Cost</th>
                    <th className="table-th">Stock</th>
                    <th className="table-th hidden md:table-cell">Tax</th>
                    {isAdmin && <th className="table-th text-right">Actions</th>}
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {items.map((p) => (
                    <tr key={p.id} className="hover:bg-gray-50/70 transition-colors">
                      <td className="table-td">
                        <p className="font-medium text-gray-900 leading-tight">{p.name}</p>
                        {/* SKU shown inline on mobile */}
                        <p className="text-xs text-gray-400 font-mono mt-0.5 sm:hidden">{p.sku}</p>
                        {p.description && (
                          <p className="text-xs text-gray-400 mt-0.5 truncate max-w-[180px] sm:max-w-xs hidden sm:block">
                            {p.description}
                          </p>
                        )}
                      </td>
                      <td className="table-td font-mono text-xs text-gray-500 hidden sm:table-cell">{p.sku}</td>
                      <td className="table-td font-semibold text-gray-800 whitespace-nowrap">
                        {formatCurrency(p.price)}
                      </td>
                      <td className="table-td text-gray-500 text-sm hidden sm:table-cell">
                        {p.cost_price > 0 ? formatCurrency(p.cost_price) : <span className="text-gray-300">—</span>}
                      </td>
                      <td className="table-td">
                        <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold ${p.stock <= (p.low_stock_threshold ?? 5)
                            ? 'bg-red-100 text-red-700'
                            : 'bg-green-100 text-green-700'
                          }`}>
                          {p.stock}
                        </span>
                      </td>
                      <td className="table-td text-gray-500 hidden md:table-cell">{p.tax_rate ?? 0}%</td>
                      {isAdmin && (
                        <td className="table-td text-right">
                          <div className="flex items-center justify-end gap-1.5">
                            <button
                              className="btn-secondary btn-sm"
                              onClick={() => {
                                setStockTarget(p)
                                setStockForm({ quantity_change: '', change_type: 'adjustment', note: '' })
                                setStockError('')
                              }}
                            >
                              Stock
                            </button>
                            <button className="btn-danger btn-sm" onClick={() => setDeleteTarget(p)}>
                              Delete
                            </button>
                          </div>
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <Pagination total={total} limit={LIMIT} offset={offset} onPageChange={setOffset} />
          </>
        )}
      </div>

      {/* ── Add Product Modal ── */}
      <Modal open={addOpen} onClose={() => { setAddOpen(false); setFormError(''); setForm(EMPTY) }} title="Add Product">
        <form onSubmit={handleAdd} className="space-y-4">
          {formError && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">{formError}</div>
          )}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="sm:col-span-2">
              <label className="label">Product Name *</label>
              <input className="input" required value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="e.g. USB-C Cable" />
            </div>
            <div>
              <label className="label">SKU *</label>
              <input className="input" required value={form.sku}
                onChange={(e) => setForm({ ...form, sku: e.target.value })} placeholder="PROD-001" />
            </div>
            <div>
              <label className="label">Selling Price (₹) *</label>
              <input className="input" type="number" min="0.01" step="0.01" required value={form.price}
                onChange={(e) => setForm({ ...form, price: e.target.value })} placeholder="0.00" />
            </div>
            <div>
              <label className="label">Cost Price (₹)</label>
              <input className="input" type="number" min="0" step="0.01" value={form.cost_price}
                onChange={(e) => setForm({ ...form, cost_price: e.target.value })} placeholder="0.00" />
            </div>
            <div>
              <label className="label">Initial Stock</label>
              <input className="input" type="number" min="0" value={form.stock}
                onChange={(e) => setForm({ ...form, stock: e.target.value })} placeholder="0" />
            </div>
            <div className="sm:col-span-2">
              <label className="label">Description</label>
              <textarea className="input" rows={2} value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
                placeholder="Optional description" />
            </div>
          </div>
          <div className="flex gap-3 pt-1">
            <button type="button" className="btn-secondary flex-1 sm:flex-none"
              onClick={() => { setAddOpen(false); setForm(EMPTY); setFormError('') }}>
              Cancel
            </button>
            <button type="submit" className="btn-primary flex-1 sm:flex-none" disabled={saving}>
              {saving ? 'Saving…' : 'Add Product'}
            </button>
          </div>
        </form>
      </Modal>

      {/* ── Stock Adjust Modal ── */}
      <Modal
        open={!!stockTarget}
        onClose={() => { setStockTarget(null); setStockError('') }}
        title={`Adjust Stock`}
        size="sm"
      >
        <form onSubmit={handleStockAdjust} className="space-y-4">
          {stockError && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">{stockError}</div>
          )}
          <div className="p-3 bg-gray-50 rounded-lg">
            <p className="text-xs text-gray-500 mb-0.5">Product</p>
            <p className="font-medium text-gray-900 text-sm">{stockTarget?.name}</p>
            <p className="text-xs text-gray-500 mt-1">
              Current stock: <span className="font-semibold text-gray-900">{stockTarget?.stock}</span>
            </p>
          </div>
          <div>
            <label className="label">Quantity Change *</label>
            <input className="input" type="number" required
              placeholder="e.g. 10 or -5"
              value={stockForm.quantity_change}
              onChange={(e) => setStockForm({ ...stockForm, quantity_change: e.target.value })} />
            <p className="text-xs text-gray-400 mt-1">Positive = add stock · Negative = remove stock</p>
          </div>
          <div>
            <label className="label">Change Type *</label>
            <select className="input" value={stockForm.change_type}
              onChange={(e) => setStockForm({ ...stockForm, change_type: e.target.value })}>
              <option value="purchase">Purchase</option>
              <option value="adjustment">Adjustment</option>
              <option value="return">Return</option>
            </select>
          </div>
          <div>
            <label className="label">Note</label>
            <input className="input" value={stockForm.note}
              onChange={(e) => setStockForm({ ...stockForm, note: e.target.value })}
              placeholder="Optional note" />
          </div>
          <div className="flex gap-3 pt-1">
            <button type="button" className="btn-secondary flex-1" onClick={() => setStockTarget(null)}>Cancel</button>
            <button type="submit" className="btn-primary flex-1" disabled={stockSaving}>
              {stockSaving ? 'Saving…' : 'Update Stock'}
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        loading={deleting}
        title="Delete Product"
        message={`Delete "${deleteTarget?.name}"? This cannot be undone.`}
      />
    </div>
  )
}
