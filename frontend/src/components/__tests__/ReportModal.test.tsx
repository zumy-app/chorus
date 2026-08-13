import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18n from '../../i18n'
import { moderationAPI } from '../../services/api'
import ReportModal from '../ReportModal'

const makeReport = (over = {}) => ({
  id: 'r1',
  type: 'user' as const,
  reporterId: 'u1',
  reportedUserId: 'u2',
  reason: 'spam',
  status: 'open',
  createdAt: new Date().toISOString(),
  ...over,
})

describe('ReportModal', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    vi.spyOn(i18n, 'changeLanguage').mockImplementation(() => Promise.resolve())
  })
  afterEach(() => vi.restoreAllMocks())

  const renderUserModal = (over: Partial<React.ComponentProps<typeof ReportModal>> = {}) =>
    render(
      <ReportModal
        targetType="user"
        targetUserId="u2"
        reportedUserName="Alice"
        onClose={vi.fn()}
        {...over}
      />
    )

  it('renders the user report form with all reason options', () => {
    renderUserModal()
    expect(screen.getByRole('heading', { name: /report user/i })).toBeInTheDocument()
    for (const label of ['Spam', 'Harassment', 'Inappropriate content', 'Scam or fraud', 'Something else']) {
      expect(screen.getByRole('radio', { name: label })).toBeInTheDocument()
    }
    // Spam is preselected.
    expect(screen.getByRole('radio', { name: 'Spam' })).toBeChecked()
  })

  it('renders the message report variant', () => {
    renderUserModal({ targetType: 'message', messageId: 'm1', chatId: 'chat-1' })
    expect(screen.getByRole('heading', { name: /report message/i })).toBeInTheDocument()
  })

  it('submits a plain reason when no detail is entered', async () => {
    const user = userEvent.setup()
    const reportSpy = vi.spyOn(moderationAPI, 'report').mockResolvedValue(makeReport() as any)
    renderUserModal()

    await user.click(screen.getByRole('radio', { name: 'Harassment' }))
    await user.click(screen.getByRole('button', { name: 'Submit report' }))

    await waitFor(() =>
      expect(reportSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'user',
          reportedUserId: 'u2',
          messageId: undefined,
          reason: 'harassment',
        })
      )
    )
    await waitFor(() =>
      expect(screen.getByText(/Thanks for reporting/)).toBeInTheDocument()
    )
  })

  it('appends detail to the selected reason', async () => {
    const user = userEvent.setup()
    const reportSpy = vi.spyOn(moderationAPI, 'report').mockResolvedValue(makeReport() as any)
    renderUserModal()

    await user.type(screen.getByPlaceholderText(/Add specifics/), 'called me names')
    await user.click(screen.getByRole('button', { name: 'Submit report' }))

    await waitFor(() =>
      expect(reportSpy).toHaveBeenCalledWith(
        expect.objectContaining({ reason: 'spam: called me names' })
      )
    )
  })

  it('sends messageId for a message report without a reported user id', async () => {
    const user = userEvent.setup()
    const reportSpy = vi.spyOn(moderationAPI, 'report').mockResolvedValue(makeReport({ type: 'message', messageId: 'm1' }) as any)
    renderUserModal({ targetType: 'message', messageId: 'm1', chatId: 'chat-1' })

    await user.click(screen.getByRole('button', { name: 'Submit report' }))

    await waitFor(() =>
      expect(reportSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'message',
          messageId: 'm1',
          chatId: 'chat-1',
          reportedUserId: undefined,
          reason: 'spam',
        })
      )
    )
  })

  it('surfaces the server error when the report fails', async () => {
    const user = userEvent.setup()
    vi.spyOn(moderationAPI, 'report').mockRejectedValue({
      response: { data: { error: 'Cannot report yourself' } },
    })
    renderUserModal()

    await user.click(screen.getByRole('button', { name: 'Submit report' }))

    await waitFor(() => expect(screen.getByText('Cannot report yourself')).toBeInTheDocument())
  })

  it('shows a fallback error when the server response has no message', async () => {
    const user = userEvent.setup()
    vi.spyOn(moderationAPI, 'report').mockRejectedValue(new Error('network down'))
    renderUserModal()

    await user.click(screen.getByRole('button', { name: 'Submit report' }))

    await waitFor(() => expect(screen.getByText('Could not submit this report.')).toBeInTheDocument())
  })

  it('closes the modal when the backdrop is clicked', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    const { container } = renderUserModal({ onClose })

    await user.click(container.firstElementChild as HTMLElement)

    expect(onClose).toHaveBeenCalledTimes(1)
  })
})