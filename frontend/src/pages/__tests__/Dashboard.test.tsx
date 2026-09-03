import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import '../../i18n'

vi.mock('react-router-dom', () => ({ useNavigate: () => vi.fn(), useParams: () => ({}) }))
vi.mock('../../store', () => ({ useStore: (s:any)=>s({ user: { id:'u1', targetLanguages:['es'], nativeLanguage:'en', displayName:'Test' } }) }))
vi.mock('../../components/AppHeader', () => ({ default: () => null }))
vi.mock('../../components/BottomNav', () => ({ default: () => null }))
vi.mock('../../services/api', () => ({
  chatAPI: { getChats: vi.fn().mockResolvedValue([{ id:'c1', type:'direct', participants:[{userId:'u1'},{userId:'u2', user:{ displayName:'Ana' }}], lastMessage:{ text:'Hola' }, unreadCount:2 },{ id:'c2', type:'group', name:'Study Group', participants:[], lastMessage:{ text:'Hello' } }]) },
  learningAPI: { getDashboard: vi.fn().mockResolvedValue({ dailyGoal:{ targetItems:10, completedItems:4, percent:40 }, streak:{ days:5, atRisk:false, canRecover:false }, fluency:{ readinessScore:300, label:'A2' }, vocabulary:{ total:42, dueToday:5, mastered:10 }, recommendedActivities:[{ id:'a1', title:'Review verbs', description:'Practice past tense' }], monthlyActivity:[], weeklyActivity:[] }) },
  api: { get: vi.fn().mockResolvedValue({ data: [{ id:'call1', chatId:'c1', type:'video', status:'ended', startedAt: new Date().toISOString(), participants:['u1','u2'] }] }) },
}))

import Dashboard from '../Dashboard'

beforeEach(()=> vi.clearAllMocks())

describe('Dashboard', () => {
  it('renders all panels with data', async () => {
    render(<Dashboard />)
    await waitFor(()=> expect(screen.getByTestId('dashboard-page')).toBeTruthy())
    expect(screen.getByTestId('dashboard-chats-panel')).toBeTruthy()
    expect(screen.getByTestId('dashboard-calls-panel')).toBeTruthy()
    expect(screen.getByTestId('dashboard-learning-panel')).toBeTruthy()
    expect(await screen.findByText('Active chats')).toBeTruthy()
    expect(screen.getByText('Recent calls')).toBeTruthy()
    expect(screen.getByText('Learning')).toBeTruthy()
  })
  it('shows responsive grid layout', async () => {
    render(<Dashboard />)
    await waitFor(()=> expect(screen.getByTestId('dashboard-page')).toBeTruthy())
    const page = screen.getByTestId('dashboard-page')
    expect(page.innerHTML).toContain('lg:grid-cols-12')
    expect(page.innerHTML).toContain('lg:grid-cols-4')
  })
})
