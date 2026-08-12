import { useEffect, useState } from 'react'
import { Routes, Route, Navigate, useNavigate } from 'react-router-dom'
import Landing from './pages/Landing'
import Login from './pages/Login'
import Register from './pages/Register'
import ForgotPassword from './pages/ForgotPassword'
import ResetPassword from './pages/ResetPassword'
import Waitlist from './pages/Waitlist'
import AdminWaitlist from './pages/AdminWaitlist'
import Chat from './pages/Chat'
import { authAPI } from './services/api'
import { wsService } from './services/websocket'
import { useStore } from './store'

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const { isAdmin, isModerator, setUser, setAdmin, refreshAdminStatus } = useStore()
  const navigate = useNavigate()

  useEffect(() => {
    const checkAuth = async () => {
      const token = localStorage.getItem('accessToken')
      if (token) {
        try {
          const user = await authAPI.getMe()
          setUser(user)
          setIsAuthenticated(true)
          refreshAdminStatus()
          wsService.connect(token)
        } catch (error) {
          localStorage.removeItem('accessToken')
          localStorage.removeItem('refreshToken')
          setAdmin(false)
          setIsAuthenticated(false)
        }
      }
      setIsLoading(false)
    }

    checkAuth()
  }, [setUser, setAdmin, refreshAdminStatus])

  const handleLogin = async (tokens: { accessToken: string; refreshToken: string }) => {
    localStorage.setItem('accessToken', tokens.accessToken)
    localStorage.setItem('refreshToken', tokens.refreshToken)

    const user = await authAPI.getMe()
    setUser(user)
    setIsAuthenticated(true)
    refreshAdminStatus()
    wsService.connect(tokens.accessToken)
    navigate('/chat')
  }

  const handleLogout = () => {
    localStorage.removeItem('accessToken')
    localStorage.removeItem('refreshToken')
    setUser(null)
    setAdmin(false)
    setIsAuthenticated(false)
    wsService.disconnect()
    navigate('/login')
  }

  if (isLoading) {
    return (
      <div className="h-screen flex items-center justify-center">
        <div className="text-xl">Loading...</div>
      </div>
    )
  }

  return (
    <>


      <Routes>
        <Route path="/" element={<Landing />} />
        <Route path="/waitlist" element={<Waitlist />} />
        <Route
          path="/admin/waitlist"
          element={
            isAuthenticated && isAdmin
              ? <AdminWaitlist defaultTab="waitlist" />
              : <Navigate to={isAuthenticated ? (isModerator ? '/admin' : '/') : '/login'} />
          }
        />
        <Route
          path="/admin"
          element={
            isAuthenticated && isModerator
              ? <AdminWaitlist defaultTab="users" />
              : <Navigate to={isAuthenticated ? '/' : '/login'} />
          }
        />
        <Route
          path="/admin/:tab"
          element={
            isAuthenticated && isModerator
              ? <AdminWaitlist />
              : <Navigate to={isAuthenticated ? '/' : '/login'} />
          }
        />
        <Route
          path="/login"
          element={
            isAuthenticated ? <Navigate to="/chat" /> : <Login onLogin={handleLogin} />
          }
        />
        <Route
          path="/register"
          element={
            isAuthenticated ? <Navigate to="/chat" /> : <Register onRegister={handleLogin} />
          }
        />
        <Route
          path="/forgot-password"
          element={isAuthenticated ? <Navigate to="/chat" /> : <ForgotPassword />}
        />
        <Route
          path="/reset-password"
          element={isAuthenticated ? <Navigate to="/chat" /> : <ResetPassword />}
        />
        <Route
          path="/chat"
          element={
            isAuthenticated ? <Chat onLogout={handleLogout} /> : <Navigate to="/login" />
          }
        />
        <Route
          path="/chat/:slug"
          element={
            isAuthenticated ? <Chat onLogout={handleLogout} /> : <Navigate to="/login" />
          }
        />
      </Routes>
    </>
  )
}

export default App
