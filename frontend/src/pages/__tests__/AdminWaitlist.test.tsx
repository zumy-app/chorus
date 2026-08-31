import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { adminAPI } from '../../services/api'

vi.mock('../../store', () => ({
  useStore: (selector: any) => selector({ userRole: 'admin' }),
}))

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

vi.mock('../../services/api', () => ({
  adminAPI: {
    listWaitlist: vi.fn(),
    listUsers: vi.fn(),
    listTranslations: vi.fn(),
    translationHealth: vi.fn(),
    emails: vi.fn(),
    stats: vi.fn(),
    premiumUsers: vi.fn(),
    premiumAnalytics: vi.fn(),
    listReports: vi.fn(),
    reportStats: vi.fn(),
  },
}))

import AdminWaitlist from '../AdminWaitlist'

beforeEach(() => {
  vi.clearAllMocks()
  adminAPI.listWaitlist.mockResolvedValue([])
  adminAPI.listUsers.mockResolvedValue({ users: [], total: 0 })
  adminAPI.listTranslations.mockResolvedValue({ jobs: [] })
  adminAPI.emails.mockResolvedValue([])
  adminAPI.stats.mockResolvedValue({})
  adminAPI.premiumUsers.mockResolvedValue({ users: [], total: 0 })
  adminAPI.premiumAnalytics.mockResolvedValue({})
  adminAPI.listReports.mockResolvedValue({ reports: [], total: 0 })
})

describe('AdminWaitlist (web)', () => {
  it('provides a back affordance to the dashboard', async () => {
    render(
      <MemoryRouter initialEntries={['/admin/waitlist']}>
        <Routes>
          <Route path="/admin/waitlist" element={<AdminWaitlist defaultTab="waitlist" />} />
        </Routes>
      </MemoryRouter>,
    )

    const back = screen.getByRole('link', { name: 'admin.back' })
    expect(back).toHaveAttribute('href', '/chat')

    await waitFor(() => expect(adminAPI.listWaitlist).toHaveBeenCalled())
  })
})
