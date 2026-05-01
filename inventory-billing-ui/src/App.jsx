import { Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './context/AuthContext'
import Layout from './components/layout/Layout'
import Login from './pages/Login'
import Register from './pages/Register'
import Dashboard from './pages/Dashboard'
import Products from './pages/Products'
import Customers from './pages/Customers'
import Invoices from './pages/Invoices'
import Profile from './pages/Profile'
import Categories from './pages/Categories'
import Vendors from './pages/Vendors'
import PurchaseOrders from './pages/PurchaseOrders'
import StockLogs from './pages/StockLogs'
import Reports from './pages/Reports'

export default function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login"    element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route element={<Layout />}>
          <Route path="/dashboard"       element={<Dashboard />} />
          <Route path="/products"        element={<Products />} />
          <Route path="/customers"       element={<Customers />} />
          <Route path="/invoices"        element={<Invoices />} />
          <Route path="/categories"      element={<Categories />} />
          <Route path="/vendors"         element={<Vendors />} />
          <Route path="/purchase-orders" element={<PurchaseOrders />} />
          <Route path="/stock-logs"      element={<StockLogs />} />
          <Route path="/reports"         element={<Reports />} />
          <Route path="/profile"         element={<Profile />} />
        </Route>
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </AuthProvider>
  )
}
