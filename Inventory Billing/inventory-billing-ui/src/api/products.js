import api from './axios'

const noCache = () => ({
  headers: {
    'Cache-Control': 'no-cache',
    Pragma: 'no-cache',
  },
})

export const getProducts = (params = {}) =>
  api.get('/products', {
    params: { ...params, _: Date.now() },
    ...noCache(),
  })

export const getProduct = (id) =>
  api.get(`/products/${id}`, {
    params: { _: Date.now() },
    ...noCache(),
  })

export const createProduct = (data) => api.post('/products', data)

export const updateProduct = (id, data) => api.put(`/products/${id}`, data)

export const deleteProduct = (id) => api.delete(`/products/${id}`)

export const adjustStock = (id, data) =>
  api.patch(`/products/${id}/stock`, data)
