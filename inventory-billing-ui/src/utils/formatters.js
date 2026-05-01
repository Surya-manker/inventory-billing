export const formatCurrency = (amount) =>
  new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(amount ?? 0)

export const formatDate = (dateStr) => {
  if (!dateStr) return '—'
  return new Intl.DateTimeFormat('en-IN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(new Date(dateStr))
}

export const formatDateTime = (dateStr) => {
  if (!dateStr) return '—'
  return new Intl.DateTimeFormat('en-IN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(dateStr))
}

export const statusColors = {
  // invoice
  pending:   'bg-yellow-100 text-yellow-800',
  partial:   'bg-blue-100   text-blue-800',
  overdue:   'bg-red-100    text-red-700',
  paid:      'bg-green-100  text-green-800',
  canceled:  'bg-gray-100   text-gray-600',
  // purchase orders
  draft:     'bg-gray-100   text-gray-600',
  confirmed: 'bg-blue-100   text-blue-700',
  received:  'bg-green-100  text-green-800',
}
