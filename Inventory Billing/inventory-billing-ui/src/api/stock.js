import api from './axios'

export const getStockAlerts     = ()              => api.get('/stock/alerts')
export const getStockLogs       = (params = {})   => api.get('/stock/logs',              { params })
export const getProductStockLogs = (id, params={}) => api.get(`/stock/logs/product/${id}`, { params })
