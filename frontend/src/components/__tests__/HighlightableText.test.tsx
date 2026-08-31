import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import i18n from '../../i18n'
import { vocabularyAPI } from '../../services/api'
import HighlightableText from '../HighlightableText'

function renderWithRouter(ui: React.ReactElement) {
  return render(<MemoryRouter>{ui}</MemoryRouter>)
}

describe('HighlightableText', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    vi.spyOn(i18n, 'changeLanguage').mockImplementation(() => Promise.resolve())
  })
  afterEach(() => vi.restoreAllMocks())

  it('highlights unknown words and dims words already in the word bank', () => {
    renderWithRouter(
      <HighlightableText
        text="Hola amigo"
        language="es"
        messageId="m1"
        knownWords={new Set(['hola'])}
      />
    )

    // 'hola' is known → rendered as a plain (dimmed) span, not a tappable button.
    const knownWord = screen.getByText('Hola')
    expect(knownWord.tagName).toBe('SPAN')
    expect(knownWord).toHaveClass('opacity-60')

    // 'amigo' is new → a tappable highlighted button.
    const newWord = screen.getByRole('button', { name: /amigo/i })
    expect(newWord).toHaveClass('word-highlight')
  })

  it('ignores short function words', () => {
    renderWithRouter(
      <HighlightableText
        text="mi amigo"
        language="es"
        messageId="m1"
        knownWords={new Set()}
      />
    )

    // 'mi' is only 2 characters → not highlighted.
    expect(screen.getByText('mi').tagName).toBe('SPAN')
    expect(screen.getByRole('button', { name: /amigo/i })).toBeInTheDocument()
  })

  it('opens the practice affordance and links to the vocabulary review', async () => {
    const user = userEvent.setup()
    renderWithRouter(
      <HighlightableText
        text="amigo"
        language="es"
        messageId="m1"
        knownWords={new Set()}
      />
    )

    await user.click(screen.getByRole('button', { name: /amigo/i }))

    expect(screen.getByRole('button', { name: /add to word bank/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Practice' })).toHaveAttribute('href', '/learn/vocabulary')
  })

  it('adds a highlighted word to the word bank (FR-27) and notifies the parent', async () => {
    const user = userEvent.setup()
    const save = vi.spyOn(vocabularyAPI, 'save').mockResolvedValue({} as any)
    const onWordSaved = vi.fn()

    renderWithRouter(
      <HighlightableText
        text="amigo"
        language="es"
        messageId="m1"
        knownWords={new Set()}
        onWordSaved={onWordSaved}
      />
    )

    await user.click(screen.getByRole('button', { name: /amigo/i }))
    await user.click(screen.getByRole('button', { name: /add to word bank/i }))

    await waitFor(() =>
      expect(save).toHaveBeenCalledWith('amigo', 'es', 'm1')
    )
    await waitFor(() => expect(onWordSaved).toHaveBeenCalledWith('amigo'))
  })
})
