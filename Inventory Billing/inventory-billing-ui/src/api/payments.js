import api from './axios'

export const getPaymentsByInvoice = (invoiceId) =>
  api.get(`/invoices/${invoiceId}/payments`)

export const recordPayment = (invoiceId, data) =>
  api.post(`/invoices/${invoiceId}/payments`, data)

export const deletePayment = (id) => api.delete(`/payments/${id}`)
