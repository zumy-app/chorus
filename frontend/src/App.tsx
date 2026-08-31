import { useEffect, useRef, useState } from 'react'
import { Routes, Route, Navigate, useNavigate } from 'react-router-dom'
import Landing from './pages/Landing'
import Pricing from './pages/Pricing'
import Premium from './pages/Premium'
import Login from './pages/Login'
import Register from './pages/Register'
import ForgotPassword from './pages/ForgotPassword'
import ResetPassword from './pages/ResetPassword'
import Waitlist from './pages/Waitlist'
import AdminWaitlist from './pages/AdminWaitlist'
import Chat from './pages/Chat'
import Learn from './pages/Learn'
import Placement from './pages/Placement'
import LessonSession from './pages/LessonSession'
import VocabularyReview from './pages/VocabularyReview'
import Scenarios from './pages/Scenarios'
import ScenarioRoleplay from './pages/ScenarioRoleplay'
import LearningRoadmap from './pages/LearningRoadmap'
import Profile from './pages/Profile'
import { authAPI, presenceAPI } from './services/api'
import { wsService } from './services/websocket'
import { useStore } from './store'

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const { isAdmin, isModerator, setUser, setEntitlements, setAdmin, refreshAdminStatus, refreshEntitlements } = useStore()
  const navigate = useNavigate()
  const presenceHeartbeatRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // Report the web client as online so other users see real presence (FR-9),
  // and refresh the backend's Redis presence TTL so it doesn't go stale.
  // The backend expires presence keys after 5 minutes, so we ping every 4.
  const startPresenceReporting = () => {
    presenceAPI.update({ status: 'online', deviceType: 'web' }).catch(() => {})
    if (presenceHeartbeatRef.current) clearInterval(presenceHeartbeatRef.current)
    presenceHeartbeatRef.current = setInterval(() => {
      presenceAPI.heartbeat('web').catch(() => {})
    }, 4 * 60 * 1000)
  }

  const stopPresenceReporting = () => {
    if (presenceHeartbeatRef.current) {
      clearInterval(presenceHeartbeatRef.current)
      presenceHeartbeatRef.current = null
    }
    presenceAPI.update({ status: 'offline' }).catch(() => {})
  }

  useEffect(() => {
    const checkAuth = async () => {
      const token = localStorage.getItem('accessToken')
      if (token) {
        try {
          const user = await authAPI.getMe()
          setUser(user)
          setIsAuthenticated(true)
          refreshAdminStatus()
          refreshEntitlements()
          wsService.connect(token)
          startPresenceReporting()
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
  }, [setUser, setAdmin, refreshAdminStatus, refreshEntitlements])

  const handleLogin = async (tokens: { accessToken: string; refreshToken: string }) => {
    localStorage.setItem('accessToken', tokens.accessToken)
    localStorage.setItem('refreshToken', tokens.refreshToken)

    const user = await authAPI.getMe()
    setUser(user)
    setIsAuthenticated(true)
    refreshAdminStatus()
    refreshEntitlements()
    wsService.connect(tokens.accessToken)
    startPresenceReporting()
    navigate('/chat')
  }

  const handleLogout = () => {
    localStorage.removeItem('accessToken')
    localStorage.removeItem('refreshToken')
    setUser(null)
    setEntitlements(null)
    setAdmin(false)
    setIsAuthenticated(false)
    wsService.disconnect()
    stopPresenceReporting()
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
        <Route path="/pricing" element={<Pricing />} />
        <Route path="/premium" element={<Premium />} />
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
        <Route
          path="/learn"
          element={
            isAuthenticated ? <Learn /> : <Navigate to="/login" />
          }
        />
        <Route
          path="/learn/placement"
          element={
            isAuthenticated ? <Placement /> : <Navigate to="/login" />
          }
        />
        <Route
          path="/learn/session"
          element={
            isAuthenticated ? <LessonSession /> : <Navigate to="/login" />
          }
        />
        <Route
          path="/learn/vocabulary"
          element={
            isAuthenticated ? <VocabularyReview /> : <Navigate to="/login" />
          }
        />
        <Route
          path="/learn/scenarios"
          element={
            isAuthenticated ? <Scenarios /> : <Navigate to="/login" />
          }
        />
        <Route
          path="/learn/scenarios/:scenarioId"
          element={
            isAuthenticated ? <ScenarioRoleplay /> : <Navigate to="/login" />
          }
        />
        <Route
          path="/learn/roadmap"
          element={
            isAuthenticated ? <LearningRoadmap /> : <Navigate to="/login" />
          }
        />
        <Route
          path="/profile"
          element={
            isAuthenticated ? <Profile onLogout={handleLogout} /> : <Navigate to="/login" />
          }
        />
      </Routes>
    </>
  )
}

export default App
