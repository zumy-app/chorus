import { NavLink } from 'react-router-dom'
import { useFeatureFlag } from '../hooks/useFeatureFlag'

const TABS = [
  { to: '/chat', label: 'Chats', icon: 'chat' },
  { to: '/learn', label: 'Learn', icon: 'school' },
  { to: '/tutors', label: 'Tutors', icon: 'cast_for_education', flag: 'teacher_marketplace' as const },
  { to: '/profile', label: 'Profile', icon: 'person' },
]

export default function BottomNav() {
  // Teacher marketplace tab is release-gated (hidden for general users).
  const { enabled: marketEnabled } = useFeatureFlag('teacher_marketplace')
  const tabs = TABS.filter((t) => !('flag' in t) || marketEnabled)
  return (
    <nav className="fixed bottom-0 w-full z-50 rounded-t-xl bg-surface-container md:hidden shadow-[0px_-4px_12px_rgba(0,0,0,0.05)] pb-[env(safe-area-inset-bottom,0px)]">
      <div className="flex justify-around items-center h-20 px-4 w-full">
        {tabs.map((tab) => (
          <NavLink
            key={tab.to}
            to={tab.to}
            aria-label={tab.label}
            className={({ isActive }) =>
              `flex flex-col items-center justify-center px-5 py-1 rounded-full transition-all duration-300 ease-out active:scale-90 ${
                isActive
                  ? 'bg-primary-container text-on-primary-container'
                  : 'text-on-surface-variant hover:bg-surface-container-high'
              }`
            }
          >
            {({ isActive }) => (
              <>
                <span
                  className="material-symbols-outlined"
                  style={{ fontVariationSettings: `'FILL' ${isActive ? 1 : 0}` }}
                >
                  {tab.icon}
                </span>
                <span className="font-label-md text-label-md mt-1">{tab.label}</span>
              </>
            )}
          </NavLink>
        ))}
      </div>
    </nav>
  )
}
