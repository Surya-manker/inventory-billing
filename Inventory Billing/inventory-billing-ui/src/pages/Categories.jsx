import { useEffect, useState, useCallback } from 'react'
import { getCategories, createCategory, updateCategory, deleteCategory } from '../api/categories'
import { useAuth } from '../context/AuthContext'
import Modal from '../components/common/Modal'
import ConfirmDialog from '../components/common/ConfirmDialog'
import LoadingSpinner from '../components/common/LoadingSpinner'
import { formatDate } from '../utils/formatters'

const LIMIT = 20
const EMPTY = { name: '', description: '' }

export default function Categories() {
  const { isAdmin } = useAuth()
  const [items, setItems]       = useState([])
  const [total, setTotal]       = useState(0)
  const [loading, setLoading]   = useState(true)
  const [error, setError]       = useState('')

  const [modalOpen, setModalOpen]   = useState(false)
  const [editTarget, setEditTarget] = useState(null) // null = create, object = edit
  const [form, setForm]             = useState(EMPTY)
  const [formError, setFormError]   = useState('')
  const [saving, setSaving]         = useState(false)

  const [deleteTarget, setDeleteTarget] = useState(null)
  const [deleting, setDeleting]         = useState(false)

  const load = useCallback(async () => {
    setLoading(true); setError('')
    try {
      const res = await getCategories({ limit: LIMIT })
      setItems(res.data.data.items ?? [])
      setTotal(res.data.data.total)
    } catch { setError('Failed to load categories.') }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { load() }, [load])

  const openCreate = () => {
    setEditTarget(null); setForm(EMPTY); setFormError(''); setModalOpen(true)
  }
  const openEdit = (cat) => {
    setEditTarget(cat)
    setForm({ name: cat.name, description: cat.description ?? '' })
    setFormError(''); setModalOpen(true)
  }

  const handleSave = async (e) => {
    e.preventDefault(); setFormError(''); setSaving(true)
    try {
      if (editTarget) {
        await updateCategory(editTarget.id, { name: form.name, description: form.description || undefined })
      } else {
        await createCategory({ name: form.name, description: form.description || undefined })
      }
      setModalOpen(false); load()
    } catch (err) {
      setFormError(err.response?.data?.message || 'Failed to save category.')
    } finally { setSaving(false) }
  }

  const handleDelete = async () => {
    setDeleting(true)
    try { await deleteCategory(deleteTarget.id); setDeleteTarget(null); load() }
    catch (err) { setError(err.response?.data?.message || 'Failed to delete category.'); setDeleteTarget(null) }
    finally { setDeleting(false) }
  }

  return (
    <div className="p-4 sm:p-6 lg:p-8 space-y-5">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h1 className="text-xl sm:text-2xl font-bold text-gray-900">Categories</h1>
          <p className="text-gray-500 text-sm mt-0.5">{total} total</p>
        </div>
        {isAdmin && (
          <button className="btn-primary" onClick={openCreate}>
            <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Add Category
          </button>
        )}
      </div>

      {error && <div className="p-3 bg-red-50 border border-red-200 rounded-xl text-sm text-red-700">{error}</div>}

      <div className="card overflow-hidden">
        {loading ? <LoadingSpinner className="py-16" /> : items.length === 0 ? (
          <div className="py-16 text-center text-gray-400 text-sm">No categories yet.</div>
        ) : (
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="table-th">Name</th>
                <th className="table-th hidden sm:table-cell">Description</th>
                <th className="table-th hidden md:table-cell">Created</th>
                {isAdmin && <th className="table-th text-right">Actions</th>}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {items.map((cat) => (
                <tr key={cat.id} className="hover:bg-gray-50/70 transition-colors">
                  <td className="table-td font-medium text-gray-900">{cat.name}</td>
                  <td className="table-td text-gray-500 text-sm hidden sm:table-cell">
                    {cat.description || <span className="text-gray-300">—</span>}
                  </td>
                  <td className="table-td text-gray-400 text-xs hidden md:table-cell">{formatDate(cat.created_at)}</td>
                  {isAdmin && (
                    <td className="table-td text-right">
                      <div className="flex items-center justify-end gap-1.5">
                        <button className="btn-secondary btn-sm" onClick={() => openEdit(cat)}>Edit</button>
                        <button className="btn-danger btn-sm" onClick={() => setDeleteTarget(cat)}>Delete</button>
                      </div>
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <Modal open={modalOpen} onClose={() => setModalOpen(false)}
        title={editTarget ? 'Edit Category' : 'Add Category'} size="sm">
        <form onSubmit={handleSave} className="space-y-4">
          {formError && <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">{formError}</div>}
          <div>
            <label className="label">Name *</label>
            <input className="input" required value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="e.g. Electronics" />
          </div>
          <div>
            <label className="label">Description</label>
            <textarea className="input" rows={2} value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })} placeholder="Optional" />
          </div>
          <div className="flex gap-3 pt-1">
            <button type="button" className="btn-secondary flex-1" onClick={() => setModalOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary flex-1" disabled={saving}>
              {saving ? 'Saving…' : editTarget ? 'Update' : 'Create'}
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete} loading={deleting} title="Delete Category"
        message={`Delete "${deleteTarget?.name}"? Products in this category will become uncategorized.`} />
    </div>
  )
}
