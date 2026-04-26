import { useEffect, useState, useCallback } from 'react'
import { getCustomers, createCustomer, deleteCustomer } from '../api/customers'
import { useAuth } from '../context/AuthContext'
import Modal from '../components/common/Modal'
import ConfirmDialog from '../components/common/ConfirmDialog'
import LoadingSpinner from '../components/common/LoadingSpinner'
import Pagination from '../components/common/Pagination'

const LIMIT = 10
const EMPTY = { name: '', email: '', phone: '', address: '', company: '', gstin: '' }

export default function Customers() {
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

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const res = await getCustomers({ limit: LIMIT, offset, search: search || undefined })
      setItems(res.data.data.items ?? [])
      setTotal(res.data.data.total)
    } catch {
      setError('Failed to load customers.')
    } finally {
      setLoading(false)
    }
  }, [offset, search])

  useEffect(() => { load() }, [load])

  const handleSearch = (e) => {
    setSearch(e.target.value)
    setOffset(0)
  }

  const handleAdd = async (e) => {
    e.preventDefault()
    setFormError('')
    setSaving(true)
    try {
      await createCustomer({
        name: form.name,
        email: form.email,
        phone: form.phone || undefined,
        address: form.address || undefined,
        company: form.company || undefined,
        gstin: form.gstin || undefined,
      })
      setAddOpen(false)
      setForm(EMPTY)
      setOffset(0)
      load()
    } catch (err) {
      setFormError(err.response?.data?.message || 'Failed to create customer.')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    setDeleting(true)
    try {
      await deleteCustomer(deleteTarget.id)
      setDeleteTarget(null)
      load()
    } catch (err) {
      setError(err.response?.data?.message || 'Failed to delete customer.')
      setDeleteTarget(null)
    } finally {
      setDeleting(false)
    }
  }

  const f = (k) => (e) => setForm({ ...form, [k]: e.target.value })

  return (
    <div className="p-8 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Customers</h1>
          <p className="text-gray-500 text-sm mt-1">{total} customers total</p>
        </div>
        {isAdmin && (
          <button className="btn-primary" onClick={() => setAddOpen(true)}>
            <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Add Customer
          </button>
        )}
      </div>

      <div className="relative max-w-xs">
        <svg className="absolute left-3 top-2.5 h-4 w-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
        <input type="text" className="input pl-9" placeholder="Search customers…"
          value={search} onChange={handleSearch} />
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">{error}</div>
      )}

      <div className="card overflow-hidden">
        {loading ? (
          <LoadingSpinner className="py-20" />
        ) : items.length === 0 ? (
          <div className="py-20 text-center text-gray-400 text-sm">
            {search ? 'No customers match your search.' : 'No customers yet.'}
          </div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="table-th">Name</th>
                    <th className="table-th">Email</th>
                    <th className="table-th">Phone</th>
                    <th className="table-th">Company</th>
                    <th className="table-th">GSTIN</th>
                    {isAdmin && <th className="table-th text-right">Actions</th>}
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {items.map((c) => (
                    <tr key={c.id} className="hover:bg-gray-50 transition-colors">
                      <td className="table-td font-medium text-gray-900">{c.name}</td>
                      <td className="table-td text-gray-500">{c.email}</td>
                      <td className="table-td text-gray-500">{c.phone || '—'}</td>
                      <td className="table-td text-gray-500">{c.company || '—'}</td>
                      <td className="table-td font-mono text-xs text-gray-400">{c.gstin || '—'}</td>
                      {isAdmin && (
                        <td className="table-td text-right">
                          <button className="btn-danger btn-sm" onClick={() => setDeleteTarget(c)}>
                            Delete
                          </button>
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

      {/* Add Customer Modal */}
      <Modal
        open={addOpen}
        onClose={() => { setAddOpen(false); setFormError(''); setForm(EMPTY) }}
        title="Add Customer"
      >
        <form onSubmit={handleAdd} className="space-y-4">
          {formError && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">{formError}</div>
          )}
          <div className="grid grid-cols-2 gap-4">
            <div className="col-span-2">
              <label className="label">Full Name *</label>
              <input className="input" required value={form.name} onChange={f('name')} />
            </div>
            <div className="col-span-2">
              <label className="label">Email *</label>
              <input className="input" type="email" required value={form.email} onChange={f('email')} />
            </div>
            <div>
              <label className="label">Phone</label>
              <input className="input" type="tel" value={form.phone} onChange={f('phone')} placeholder="+91 …" />
            </div>
            <div>
              <label className="label">Company</label>
              <input className="input" value={form.company} onChange={f('company')} />
            </div>
            <div>
              <label className="label">GSTIN</label>
              <input className="input" value={form.gstin} onChange={f('gstin')} placeholder="22AAAAA0000A1Z5" />
            </div>
            <div className="col-span-2">
              <label className="label">Address</label>
              <textarea className="input" rows={2} value={form.address} onChange={f('address')} />
            </div>
          </div>
          <div className="flex justify-end gap-3 pt-2">
            <button type="button" className="btn-secondary"
              onClick={() => { setAddOpen(false); setForm(EMPTY); setFormError('') }}>
              Cancel
            </button>
            <button type="submit" className="btn-primary" disabled={saving}>
              {saving ? 'Saving…' : 'Add Customer'}
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        loading={deleting}
        title="Delete Customer"
        message={`Are you sure you want to delete "${deleteTarget?.name}"? This cannot be undone.`}
      />
    </div>
  )
}
