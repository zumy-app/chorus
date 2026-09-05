import { ReactNode } from 'react'
import LanguageSelector from './LanguageSelector'

interface AuthShellProps {
  children: ReactNode
  title?: string
  tagline?: ReactNode
  bottom?: ReactNode
  selectedLang: string
  onLanguageChange: (code: string) => void
}

export default function AuthShell({
  children,
  title = 'Chorus',
  tagline,
  bottom,
  selectedLang,
  onLanguageChange,
}: AuthShellProps) {
  return (
    <div className="min-h-screen bg-surface text-on-surface flex flex-col relative overflow-x-hidden">
      {/* Ambient background pattern */}
      <div
        className="absolute inset-0 pointer-events-none -z-10"
        style={{ backgroundImage: 'radial-gradient(circle at 50% 0%, rgba(37,99,235,0.08) 0%, transparent 60%)' }}
      />

      {/* Language selector top-right */}
      <div className="fixed top-4 right-4 z-50">
        <LanguageSelector
          currentLang={selectedLang}
          onLanguageChange={onLanguageChange}
          variant="navbar"
        />
      </div>

      <main className="flex-grow flex flex-col px-margin-mobile pt-12 pb-8 w-full max-w-md mx-auto relative z-10 justify-center min-h-screen">
        {/* Brand header */}
        <header className="flex flex-col items-center justify-center mb-10">
          <div className="w-16 h-16 bg-primary-container rounded-2xl flex items-center justify-center mb-6 shadow-[0px_8px_24px_rgba(37,99,235,0.2)] rotate-[-10deg] hover:rotate-0 transition-transform duration-300">
            <span className="material-symbols-outlined text-on-primary-container text-4xl" style={{ fontVariationSettings: "'FILL' 1" }}>
              forum
            </span>
          </div>
          <h1 className="font-headline-lg text-headline-lg text-primary text-center mb-2">{title}</h1>
          {tagline && (
            <p className="font-body-md text-body-md text-on-surface-variant text-center px-4">{tagline}</p>
          )}
        </header>

        {children}

        {bottom && (
          <div className="mt-8 text-center flex flex-col items-center gap-2">{bottom}</div>
        )}
      </main>
    </div>
  )
}
