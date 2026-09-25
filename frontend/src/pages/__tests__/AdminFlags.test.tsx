import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent, within } from '@testing-library/react'
import '../../i18n'

vi.mock('../../services/api', () => ({
  adminAPI: {
    listFlags: vi.fn().mockResolvedValue([
      { key: 'video_calls', description: 'WebRTC calls', defaultState: false, adminOnly: true, betaAccess: false, stable: false, createdAt: '', updatedAt: '' },
      { key: 'word_collector', description: 'Tap-to-save', defaultState: false, adminOnly: false, betaAccess: true, stable: true, createdAt: '', updatedAt: '' },
    ]),
    updateFlagTiers: vi.fn().mockResolvedValue({ ok: true }),
    setFlagOverride: vi.fn().mockResolvedValue({ ok: true }),
    deleteFlagOverride: vi.fn().mockResolvedValue({ ok: true }),
    previewUserFlags: vi.fn().mockResolvedValue({ video_calls: false, word_collector: true }),
    listUsers: vi.fn().mockResolvedValue({ users: [{ id: 'u1', email: 'a@x.test', displayName: 'A' }], total: 1 }),
  },
}))

import AdminFlags from '../AdminFlags'
import { adminAPI } from '../../services/api'

beforeEach(() => vi.clearAllMocks())

describe('AdminFlags console', () => {
  it('lists flags with tier toggles and saves on change', async () => {
    render(<AdminFlags />)
    const row = await screen.findByTestId('flag-row-video_calls')
    expect(row).toBeTruthy()
    expect(screen.getByTestId('flag-row-word_collector')).toBeTruthy()

    // Flip stable on for video_calls, then save (checkboxes render
    // admin/beta/stable in order within each row).
    const boxes = row.querySelectorAll('input[type="checkbox"]')
    fireEvent.click(boxes[2])
    const saveBtn = Array.from(row.querySelectorAll('button')).find(b => b.textContent === 'Save')!
    fireEvent.click(saveBtn)
    await waitFor(() =>
      expect(adminAPI.updateFlagTiers).toHaveBeenCalledWith('video_calls', {
        adminOnly: true,
        betaAccess: false,
        stable: true,
      })
    )
  })

  it('previews resolved flags for a user id', async () => {
    render(<AdminFlags />)
    await screen.findByTestId('flag-row-video_calls')
    fireEvent.change(screen.getByTestId('preview-user-id'), { target: { value: 'user-9' } })
    fireEvent.click(screen.getByText('Preview'))
    await waitFor(() => expect(adminAPI.previewUserFlags).toHaveBeenCalledWith('user-9'))
    const grid = await screen.findByTestId('preview-grid')
    expect(within(grid).getByText('word_collector')).toBeTruthy()
    expect(within(grid).getByText('ON')).toBeTruthy()
  })

  it('sets a per-user override', async () => {
    render(<AdminFlags />)
    await screen.findByTestId('flag-row-video_calls')
    const combo = screen.getByDisplayValue('Select flag…') as HTMLSelectElement
    fireEvent.change(combo, { target: { value: 'video_calls' } })
    fireEvent.change(screen.getByTestId('override-user-id'), { target: { value: 'user-9' } })
    fireEvent.click(screen.getByText('Set'))
    await waitFor(() =>
      expect(adminAPI.setFlagOverride).toHaveBeenCalledWith('video_calls', 'user-9', true)
    )
  })
})
