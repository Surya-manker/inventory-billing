import api from './axios'

export const getRevenueReport    = (params = {}) => api.get('/reports/revenue',         { params })
export const getTopProducts      = (params = {}) => api.get('/reports/top-products',    { params })
export const getInvoiceSummary   = (params = {}) => api.get('/reports/invoices',        { params })
export const getInventoryValue   = ()            => api.get('/reports/inventory-value')
export const getAgingReport      = ()            => api.get('/reports/aging')
