import { Link } from 'react-router-dom'
import { useStore } from '../store'

interface AppHeaderProps {
  title?: string
  showAvatar?: boolean
  onTranslate?: () => void
}

export default function AppHeader({ title = 'Chorus', showAvatar = true, onTranslate }: AppHeaderProps) {
  const { user } = useStore()

  return (
    <header className="bg-surface shadow-sm z-40 shrink-0">
      <div className="flex justify-between items-center w-full px-margin-mobile py-stack-sm">
        <Link to="/chat" aria-label="Home" className="flex items-center gap-2">
          {showAvatar && (
            <div className="w-8 h-8 rounded-full bg-primary-container text-on-primary-container flex items-center justify-center font-bold text-sm">
              {user?.displayName?.charAt(0).toUpperCase() || user?.username?.charAt(0).toUpperCase() || 'C'}
            </div>
          )}
          <span className="font-headline-md text-headline-md font-bold text-primary tracking-tight">{title}</span>
        </Link>
        <button
          aria-label="Translate"
          onClick={onTranslate}
          className="text-primary hover:bg-surface-container transition-colors active:scale-95 duration-150 p-2 rounded-full flex items-center justify-center"
        >
          <span className="material-symbols-outlined">translate</span>
        </button>
      </div>
    </header>
  )
}
