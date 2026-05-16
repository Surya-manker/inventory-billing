import { useEffect, useState, useCallback } from 'react'
import { getVendors, createVendor, updateVendor, deactivateVendor, deleteVendor } from '../api/vendors'
import Modal from '../components/common/Modal'
import ConfirmDialog from '../components/common/ConfirmDialog'
import LoadingSpinner from '../components/common/LoadingSpinner'
import Pagination from '../components/common/Pagination'
import Badge from '../components/common/Badge'
import { formatDate } from '../utils/formatters'

const LIMIT = 15
const EMPTY = { name: '', email: '', phone: '', company: '', gstin: '', address: '' }

export default function Vendors() {
  const [items, setItems]   = useState([])
  const [total, setTotal]   = useState(0)
  const [offset, setOffset] = useState(0)
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError]   = useState('')

  const [modalOpen, setModalOpen]   = useState(false)
  const [editTarget, setEditTarget] = useState(null)
  const [form, setForm]             = useState(EMPTY)
  const [formError, setFormError]   = useState('')
  const [saving, setSaving]         = useState(false)

  const [deactivateTarget, setDeactivateTarget] = useState(null)
  const [deactivating, setDeactivating]         = useState(false)
  const [deleteTarget, setDeleteTarget]         = useState(null)
  const [deleting, setDeleting]                 = useState(false)

  const load = useCallback(async () => {
    setLoading(true); setError('')
    try {
      const res = await getVendors({ limit: LIMIT, offset, search: search || undefined })
      setItems(res.data.data.items ?? [])
      setTotal(res.data.data.total)
    } catch { setError('Failed to load vendors.') }
    finally { setLoading(false) }
  }, [offset, search])

  useEffect(() => { load() }, [load])

  const openCreate = () => { setEditTarget(null); setForm(EMPTY); setFormError(''); setModalOpen(true) }
  const openEdit   = (v) => {
    setEditTarget(v)
    setForm({ name: v.name, email: v.email ?? '', phone: v.phone ?? '',
              company: v.company ?? '', gstin: v.gstin ?? '', address: v.address ?? '' })
    setFormError(''); setModalOpen(true)
  }

  const handleSave = async (e) => {
    e.preventDefault(); setFormError(''); setSaving(true)
    try {
      const payload = {
        name:    form.name,
        email:   form.email   || undefined,
        phone:   form.phone   || undefined,
        company: form.company || undefined,
        gstin:   form.gstin   || undefined,
        address: form.address || undefined,
      }
      if (editTarget) await updateVendor(editTarget.id, payload)
      else            await createVendor(payload)
      setModalOpen(false); load()
    } catch (err) {
      setFormError(err.response?.data?.message || 'Failed to save vendor.')
    } finally { setSaving(false) }
  }

  const handleDeactivate = async () => {
    setDeactivating(true)
    try { await deactivateVendor(deactivateTarget.id); setDeactivateTarget(null); load() }
    catch (err) { setError(err.response?.data?.message || 'Failed to deactivate.'); setDeactivateTarget(null) }
    finally { setDeactivating(false) }
  }

  const handleDelete = async () => {
    setDeleting(true)
    try { await deleteVendor(deleteTarget.id); setDeleteTarget(null); load() }
    catch (err) { setError(err.response?.data?.message || 'Failed to delete vendor.'); setDeleteTarget(null) }
    finally { setDeleting(false) }
  }

  return (
    <div className="p-4 sm:p-6 lg:p-8 space-y-5">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h1 className="text-xl sm:text-2xl font-bold text-gray-900">Vendors</h1>
          <p className="text-gray-500 text-sm mt-0.5">{total} total</p>
        </div>
        <button className="btn-primary" onClick={openCreate}>
          <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          <span className="hidden sm:inline">Add Vendor</span>
          <span className="sm:hidden">Add</span>
        </button>
      </div>

      <div className="relative w-full sm:max-w-xs">
        <svg className="absolute left-3 top-2.5 h-4 w-4 text-gray-400 pointer-events-none" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
        <input type="text" className="input pl-9" placeholder="Search vendors…" value={search}
          onChange={(e) => { setSearch(e.target.value); setOffset(0) }} />
      </div>

      {error && <div className="p-3 bg-red-50 border border-red-200 rounded-xl text-sm text-red-700">{error}</div>}

      <div className="card overflow-hidden">
        {loading ? <LoadingSpinner className="py-16" /> : items.length === 0 ? (
          <div className="py-16 text-center text-gray-400 text-sm">
            {search ? 'No vendors match your search.' : 'No vendors yet.'}
          </div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[480px]">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="table-th">Vendor</th>
                    <th className="table-th hidden sm:table-cell">Contact</th>
                    <th className="table-th hidden md:table-cell">GSTIN</th>
                    <th className="table-th">Status</th>
                    <th className="table-th text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {items.map((v) => (
                    <tr key={v.id} className="hover:bg-gray-50/70 transition-colors">
                      <td className="table-td">
                        <p className="font-medium text-gray-900">{v.name}</p>
                        {v.company && <p className="text-xs text-gray-400 mt-0.5">{v.company}</p>}
                      </td>
                      <td className="table-td hidden sm:table-cell text-sm text-gray-600">
                        <p>{v.email ?? '—'}</p>
                        {v.phone && <p className="text-xs text-gray-400">{v.phone}</p>}
                      </td>
                      <td className="table-td font-mono text-xs text-gray-500 hidden md:table-cell">
                        {v.gstin || '—'}
                      </td>
                      <td className="table-td">
                        <Badge className={v.is_active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-500'}>
                          {v.is_active ? 'Active' : 'Inactive'}
                        </Badge>
                      </td>
                      <td className="table-td text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          <button className="btn-secondary btn-sm" onClick={() => openEdit(v)}>Edit</button>
                          {v.is_active && (
                            <button className="btn-secondary btn-sm text-orange-600 border-orange-200 hover:bg-orange-50"
                              onClick={() => setDeactivateTarget(v)}>
                              Deactivate
                            </button>
                          )}
                          <button className="btn-danger btn-sm" onClick={() => setDeleteTarget(v)}>Delete</button>
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

      {/* Add / Edit Modal */}
      <Modal open={modalOpen} onClose={() => setModalOpen(false)}
        title={editTarget ? 'Edit Vendor' : 'Add Vendor'}>
        <form onSubmit={handleSave} className="space-y-4">
          {formError && <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">{formError}</div>}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="sm:col-span-2">
              <label className="label">Vendor Name *</label>
              <input className="input" required value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="e.g. Reliance Retail" />
            </div>
            <div>
              <label className="label">Email</label>
              <input className="input" type="email" value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })} placeholder="vendor@example.com" />
            </div>
            <div>
              <label className="label">Phone</label>
              <input className="input" value={form.phone}
                onChange={(e) => setForm({ ...form, phone: e.target.value })} placeholder="+91 98765 43210" />
            </div>
            <div>
              <label className="label">Company</label>
              <input className="input" value={form.company}
                onChange={(e) => setForm({ ...form, company: e.target.value })} placeholder="Legal entity name" />
            </div>
            <div>
              <label className="label">GSTIN</label>
              <input className="input font-mono" value={form.gstin} maxLength={15}
                onChange={(e) => setForm({ ...form, gstin: e.target.value.toUpperCase() })} placeholder="29ABCDE1234F1Z5" />
            </div>
            <div className="sm:col-span-2">
              <label className="label">Address</label>
              <textarea className="input" rows={2} value={form.address}
                onChange={(e) => setForm({ ...form, address: e.target.value })} placeholder="Full address" />
            </div>
          </div>
          <div className="flex gap-3 pt-1">
            <button type="button" className="btn-secondary flex-1 sm:flex-none" onClick={() => setModalOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary flex-1 sm:flex-none" disabled={saving}>
              {saving ? 'Saving…' : editTarget ? 'Update Vendor' : 'Add Vendor'}
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog open={!!deactivateTarget} onClose={() => setDeactivateTarget(null)}
        onConfirm={handleDeactivate} loading={deactivating} title="Deactivate Vendor"
        message={`Deactivate "${deactivateTarget?.name}"? New purchase orders cannot be placed with inactive vendors.`} />

      <ConfirmDialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete} loading={deleting} title="Delete Vendor"
        message={`Permanently delete "${deleteTarget?.name}"? This cannot be undone.`} />
    </div>
  )
}
