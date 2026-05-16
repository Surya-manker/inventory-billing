import api from './axios'

export const getPurchaseOrders = (params = {}) =>
  api.get('/purchase-orders', { params })

export const getPurchaseOrder = (id) => api.get(`/purchase-orders/${id}`)

export const createPurchaseOrder = (data) => api.post('/purchase-orders', data)

export const updatePurchaseOrderStatus = (id, status) =>
  api.patch(`/purchase-orders/${id}/status`, { status })

export const deletePurchaseOrder = (id) => api.delete(`/purchase-orders/${id}`)
