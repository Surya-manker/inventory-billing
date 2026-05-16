import api from './axios'

export const getDashboardStats = async () => {
  const [products, customers, allInvoices, pending, paid, canceled, stockAlerts] =
    await Promise.all([
      api.get('/products',  { params: { limit: 1 } }),
      api.get('/customers', { params: { limit: 1 } }),
      api.get('/invoices',  { params: { limit: 1 } }),
      api.get('/invoices',  { params: { limit: 1, status: 'pending'  } }),
      api.get('/invoices',  { params: { limit: 1, status: 'paid'     } }),
      api.get('/invoices',  { params: { limit: 1, status: 'canceled' } }),
      api.get('/stock/alerts'),
    ])

  return {
    totalProducts:    products.data.data.total,
    totalCustomers:   customers.data.data.total,
    totalInvoices:    allInvoices.data.data.total,
    pendingInvoices:  pending.data.data.total,
    paidInvoices:     paid.data.data.total,
    canceledInvoices: canceled.data.data.total,
    lowStockCount:    stockAlerts.data.data.count,
    lowStockItems:    stockAlerts.data.data.items ?? [],
  }
}
