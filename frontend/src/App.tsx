import { useEffect, useState } from 'react'
import { Routes, Route, Navigate, useNavigate } from 'react-router-dom'
import Login from './pages/Login'
import Register from './pages/Register'
import Chat from './pages/Chat'
import { authAPI } from './services/api'
import { wsService } from './services/websocket'
import { useStore } from './store'

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const { setUser } = useStore()
  const navigate = useNavigate()

  useEffect(() => {
    const checkAuth = async () => {
      const token = localStorage.getItem('accessToken')
      if (token) {
        try {
          const user = await authAPI.getMe()
          setUser(user)
          setIsAuthenticated(true)
          wsService.connect(token)
        } catch (error) {
          localStorage.removeItem('accessToken')
          localStorage.removeItem('refreshToken')
          setIsAuthenticated(false)
        }
      }
      setIsLoading(false)
    }

    checkAuth()
  }, [setUser])

  const handleLogin = async (tokens: { accessToken: string; refreshToken: string }) => {
    localStorage.setItem('accessToken', tokens.accessToken)
    localStorage.setItem('refreshToken', tokens.refreshToken)
    
    const user = await authAPI.getMe()
    setUser(user)
    setIsAuthenticated(true)
    wsService.connect(tokens.accessToken)
    navigate('/chat')
  }

  const handleLogout = () => {
    localStorage.removeItem('accessToken')
    localStorage.removeItem('refreshToken')
    setUser(null)
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
    <Routes>
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
        path="/chat"
        element={
          isAuthenticated ? <Chat onLogout={handleLogout} /> : <Navigate to="/login" />
        }
      />
      <Route path="/" element={<Navigate to="/chat" />} />
    </Routes>
  )
}

export default App
