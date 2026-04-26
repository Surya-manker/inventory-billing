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

export default function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login"    element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route element={<Layout />}>
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/products"  element={<Products />} />
          <Route path="/customers" element={<Customers />} />
          <Route path="/invoices"  element={<Invoices />} />
          <Route path="/profile"   element={<Profile />} />
        </Route>
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </AuthProvider>
  )
}
