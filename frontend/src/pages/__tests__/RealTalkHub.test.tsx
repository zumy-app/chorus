import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import '../../i18n'

vi.mock('react-router-dom', () => ({ useNavigate: () => vi.fn(), useParams: () => ({}) }))
vi.mock('../../store', () => ({ useStore: (s: any) => s({ user: { id: 'u1', targetLanguages: ['es'], nativeLanguage: 'en', displayName: 'Test' } }) }))
vi.mock('../../components/AppHeader', () => ({ default: () => null }))
vi.mock('../../components/BottomNav', () => ({ default: () => null }))
vi.mock('../../services/api', () => ({
  learningAPI: {
    getRealTalkPrompts: vi.fn().mockResolvedValue([
      { id: 'p1', category: 'Task-Based', text: 'Politely state your perspective when offering a different opinion.', targetPhrase: 'Desde mi punto de vista, depende bastante.', whyUseful: 'Great for debates.' },
    ]),
    getDashboard: vi.fn().mockResolvedValue(null),
    markRealTalkUsed: vi.fn().mockResolvedValue({}),
  },
}))

import RealTalkHub from '../RealTalkHub'
import { learningAPI } from '../../services/api'

beforeEach(() => vi.clearAllMocks())

describe('RealTalkHub', () => {
  it('headlines the sendable target phrase, not the instruction', async () => {
    render(<RealTalkHub />)
    await waitFor(() => expect(screen.getByText(/Desde mi punto de vista/)).toBeTruthy())
    // Instruction is shown as context, not as the message.
    expect(screen.getByText(/Politely state your perspective/)).toBeTruthy()
    expect(screen.getByText(/Great for debates/)).toBeTruthy()
  })

  it('drafts the target phrase into chat, never the instruction', async () => {
    render(<RealTalkHub />)
    await waitFor(() => expect(screen.getByText(/Desde mi punto de vista/)).toBeTruthy())
    fireEvent.click(screen.getByText(/Use in Chat/))
    await waitFor(() => expect(learningAPI.markRealTalkUsed).toHaveBeenCalledWith('p1'))
    expect(localStorage.getItem('realTalkDraft')).toBe('Desde mi punto de vista, depende bastante.')
  })
})
