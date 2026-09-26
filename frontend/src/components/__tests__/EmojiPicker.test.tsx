import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import EmojiPicker from '../EmojiPicker'

describe('EmojiPicker', () => {
  afterEach(() => vi.restoreAllMocks())

  it('shows the first category grid and switches categories on tab click', async () => {
    const user = userEvent.setup()
    render(<EmojiPicker onSelect={() => {}} onClose={() => {}} />)

    // Default category is Smileys.
    expect(screen.getByRole('button', { name: '😀' })).toBeInTheDocument()

    // Switch to the Food category via its tab.
    await user.click(screen.getByRole('button', { name: 'Food' }))
    expect(screen.getByRole('button', { name: '🍕' })).toBeInTheDocument()

    // The Food tab is now marked active.
    expect(screen.getByRole('button', { name: 'Food' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('calls onSelect with the clicked emoji', async () => {
    const user = userEvent.setup()
    const onSelect = vi.fn()
    render(<EmojiPicker onSelect={onSelect} onClose={() => {}} />)

    await user.click(screen.getByRole('button', { name: '😀' }))
    expect(onSelect).toHaveBeenCalledWith('😀')
  })

  it('closes on Escape', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    render(<EmojiPicker onSelect={() => {}} onClose={onClose} />)

    await user.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })
})
