import api from './axios'

export const getInvoices = (params = {}) =>
  api.get('/invoices', { params })

export const getInvoice = (id) => api.get(`/invoices/${id}`)

export const createInvoice = (data) => api.post('/invoices', data)

export const updateInvoiceStatus = (id, status) =>
  api.patch(`/invoices/${id}/status`, { status })

export const deleteInvoice = (id) => api.delete(`/invoices/${id}`)
