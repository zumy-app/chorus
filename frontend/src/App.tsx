import { useEffect, useState } from 'react'
import { Routes, Route, Navigate, useLocation, useNavigate } from 'react-router-dom'
import Landing from './pages/Landing'
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
  const location = useLocation()

  const handleBack = () => {
    if (location.pathname === '/' && window.location.hash) {
      navigate('/', { replace: true })
      window.scrollTo({ top: 0, behavior: 'smooth' })
      return
    }

    if (window.history.length > 1) {
      navigate(-1)
      return
    }

    navigate('/')
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const handleHome = () => {
    navigate('/')
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

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
    <>
      <div className="fixed bottom-5 right-5 z-[70] flex gap-2">
        <button
          type="button"
          onClick={handleBack}
          className="px-4 py-3 rounded-full bg-white border border-gray-300 text-gray-800 shadow-lg hover:bg-gray-50 font-semibold"
          aria-label="Go back"
        >
          ← Back
        </button>
        <button
          type="button"
          onClick={handleHome}
          className="px-4 py-3 rounded-full bg-primary text-white shadow-lg hover:bg-primary/90 font-semibold"
          aria-label="Go home"
        >
          🏠 Home
        </button>
      </div>

      <Routes>
        <Route path="/" element={<Landing />} />
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
      </Routes>
    </>
  )
}

export default App
