import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { useStore } from '../../store'
import AppHeader from '../AppHeader'

beforeEach(() => {
  useStore.setState({ user: { id: 'u1', displayName: 'Me' } as any })
})

describe('AppHeader', () => {
  it('renders the Home button as a link to the home screen', () => {
    render(
      <MemoryRouter>
        <AppHeader />
      </MemoryRouter>,
    )

    const home = screen.getByRole('link', { name: 'Home' })
    expect(home).toHaveAttribute('href', '/chat')
  })

  it('navigates to the home screen when Home is clicked from the dashboard', () => {
    render(
      <MemoryRouter initialEntries={['/learn']}>
        <Routes>
          <Route path="/learn" element={<AppHeader />} />
          <Route path="/chat" element={<div>home-screen</div>} />
        </Routes>
      </MemoryRouter>,
    )

    fireEvent.click(screen.getByRole('link', { name: 'Home' }))

    expect(screen.getByText('home-screen')).toBeInTheDocument()
  })
})
