import api from './axios'

export const issueCreditNote       = (invoiceId, data) =>
  api.post(`/invoices/${invoiceId}/credit-notes`, data)

export const getCreditNotesByInvoice = (invoiceId) =>
  api.get(`/invoices/${invoiceId}/credit-notes`)

export const getCreditNotesByCustomer = (customerId) =>
  api.get(`/customers/${customerId}/credit-notes`)

export const getCreditNote  = (id) => api.get(`/credit-notes/${id}`)
export const voidCreditNote = (id) => api.patch(`/credit-notes/${id}/void`)
