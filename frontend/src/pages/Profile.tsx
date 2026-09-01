import { useTranslation } from 'react-i18next'
import { useStore } from '../store'
import BottomNav from '../components/BottomNav'
import AppHeader from '../components/AppHeader'
import PrivacySettings from '../components/PrivacySettings'

interface ProfileProps {
  onLogout: () => void
}

export default function Profile({ onLogout }: ProfileProps) {
  const { t } = useTranslation()
  const user = useStore((s) => s.user)
  const entitlements = useStore((s) => s.entitlements)

  const nativeLang = user?.nativeLanguage?.toUpperCase() || 'EN'
  const targetLang = user?.targetLanguages?.[0]?.toUpperCase() || 'ES'
  const planLabel = entitlements?.effectivePlan === 'premium' ? 'Pro' : entitlements?.effectivePlan === 'free' ? t('plan.free') : t('plan.unlimited')

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-margin-mobile py-stack-lg space-y-stack-lg pb-32 max-w-md w-full mx-auto">
        <div className="mb-stack-lg">
          <h2 className="font-headline-sm text-headline-sm text-on-surface mb-unit">{t('profile.title')}</h2>
          <p className="font-body-sm text-body-sm text-on-surface-variant">{t('profile.subtitle')}</p>
        </div>

        {/* Account Section */}
        <section className="bg-surface-container-lowest rounded-xl shadow-sm border border-outline-variant/40 overflow-hidden">
          <h3 className="font-label-sm text-label-sm text-primary uppercase tracking-wider px-stack-md py-stack-sm bg-surface-container-low border-b border-outline-variant/40">{t('profile.account')}</h3>
          <div className="flex items-center justify-between px-stack-md py-stack-md border-b border-outline-variant/40 hover:bg-surface-container-low transition-colors cursor-pointer">
            <div className="flex items-center gap-stack-md">
              <span className="material-symbols-outlined text-outline">person</span>
              <span className="font-body-md text-body-md text-on-surface">{t('profile.profile')}</span>
            </div>
            <span className="material-symbols-outlined text-outline-variant">chevron_right</span>
          </div>
          <div className="flex items-center justify-between px-stack-md py-stack-md hover:bg-surface-container-low transition-colors cursor-pointer">
            <div className="flex items-center gap-stack-md">
              <span className="material-symbols-outlined text-outline">star</span>
              <span className="font-body-md text-body-md text-on-surface">{t('profile.subscription')}</span>
            </div>
            <div className="flex items-center gap-unit">
              <span className="font-label-sm text-label-sm bg-primary-container text-on-primary-container px-2 py-1 rounded-full">{planLabel}</span>
              <span className="material-symbols-outlined text-outline-variant">chevron_right</span>
            </div>
          </div>
        </section>

        {/* Language Section */}
        <section className="bg-surface-container-lowest rounded-xl shadow-sm border border-outline-variant/40 overflow-hidden">
          <h3 className="font-label-sm text-label-sm text-primary uppercase tracking-wider px-stack-md py-stack-sm bg-surface-container-low border-b border-outline-variant/40">{t('profile.language')}</h3>
          <div className="flex items-center justify-between px-stack-md py-stack-md border-b border-outline-variant/40 hover:bg-surface-container-low transition-colors cursor-pointer">
            <div className="flex items-center gap-stack-md">
              <span className="material-symbols-outlined text-outline">language</span>
              <div className="flex flex-col">
                <span className="font-body-md text-body-md text-on-surface">{t('profile.learning')}</span>
                <span className="font-body-sm text-body-sm text-on-surface-variant">{targetLang}</span>
              </div>
            </div>
            <span className="material-symbols-outlined text-outline-variant">chevron_right</span>
          </div>
          <div className="flex items-center justify-between px-stack-md py-stack-md hover:bg-surface-container-low transition-colors cursor-pointer">
            <div className="flex items-center gap-stack-md">
              <span className="material-symbols-outlined text-outline">chat</span>
              <div className="flex flex-col">
                <span className="font-body-md text-body-md text-on-surface">{t('profile.native')}</span>
                <span className="font-body-sm text-body-sm text-on-surface-variant">{nativeLang}</span>
              </div>
            </div>
            <span className="material-symbols-outlined text-outline-variant">chevron_right</span>
          </div>
        </section>

        {/* AI Features Section */}
        <section className="bg-surface-container-lowest rounded-xl shadow-sm border border-outline-variant/40 overflow-hidden relative">
          <div className="absolute top-0 left-0 w-1 h-full bg-secondary" />
          <h3 className="font-label-sm text-label-sm text-secondary uppercase tracking-wider px-stack-md py-stack-sm bg-surface-container-low border-b border-outline-variant/40 pl-[20px]">{t('profile.aiFeatures')}</h3>
          <div className="flex items-center justify-between px-stack-md py-stack-md border-b border-outline-variant/40 hover:bg-surface-container-low transition-colors pl-[20px]">
            <div className="flex items-center gap-stack-md">
              <span className="material-symbols-outlined text-secondary">auto_awesome</span>
              <div className="flex flex-col">
                <span className="font-body-md text-body-md text-on-surface">{t('profile.autoTranslation')}</span>
                <span className="font-body-sm text-body-sm text-on-surface-variant">{t('profile.autoTranslationDesc')}</span>
              </div>
            </div>
            <span className="relative inline-flex items-center h-6 w-11 bg-secondary rounded-full">
              <span className="inline-block h-5 w-5 bg-white rounded-full shadow transition-transform translate-x-[22px]" />
            </span>
          </div>
          <div className="flex items-center justify-between px-stack-md py-stack-md hover:bg-surface-container-low transition-colors cursor-pointer pl-[20px]">
            <div className="flex items-center gap-stack-md">
              <span className="material-symbols-outlined text-secondary">analytics</span>
              <div className="flex flex-col">
                <span className="font-body-md text-body-md text-on-surface">{t('profile.grammarAnalysis')}</span>
                <span className="font-body-sm text-body-sm text-on-surface-variant">Moderate</span>
              </div>
            </div>
            <span className="material-symbols-outlined text-outline-variant">chevron_right</span>
          </div>
        </section>

        {/* Privacy Section */}
        <section className="bg-surface-container-lowest rounded-xl shadow-sm border border-outline-variant/40 overflow-hidden p-4">
          <PrivacySettings />
        </section>

        <section className="bg-surface-container-lowest rounded-xl shadow-sm border border-outline-variant/40 overflow-hidden">
          <a href="/become-teacher" className="flex items-center justify-between px-stack-md py-stack-md hover:bg-surface-container-low transition-colors cursor-pointer">
            <div className="flex items-center gap-stack-md">
              <span className="material-symbols-outlined text-outline">school</span>
              <span className="font-body-md text-body-md text-on-surface">Become a teacher</span>
            </div>
            <span className="material-symbols-outlined text-outline-variant">chevron_right</span>
          </a>
        </section>

        {/* Preferences Section */}
        <section className="bg-surface-container-lowest rounded-xl shadow-sm border border-outline-variant/40 overflow-hidden">
          <h3 className="font-label-sm text-label-sm text-primary uppercase tracking-wider px-stack-md py-stack-sm bg-surface-container-low border-b border-outline-variant/40">{t('profile.preferences')}</h3>
          <div className="flex items-center justify-between px-stack-md py-stack-md border-b border-outline-variant/40 hover:bg-surface-container-low transition-colors cursor-pointer">
            <div className="flex items-center gap-stack-md">
              <span className="material-symbols-outlined text-outline">notifications</span>
              <span className="font-body-md text-body-md text-on-surface">{t('profile.notifications')}</span>
            </div>
            <span className="material-symbols-outlined text-outline-variant">chevron_right</span>
          </div>
          <div className="flex items-center justify-between px-stack-md py-stack-md hover:bg-surface-container-low transition-colors cursor-pointer">
            <div className="flex items-center gap-stack-md">
              <span className="material-symbols-outlined text-outline">dark_mode</span>
              <span className="font-body-md text-body-md text-on-surface">{t('profile.theme')}</span>
            </div>
            <div className="flex items-center gap-unit">
              <span className="font-body-sm text-body-sm text-on-surface-variant">{t('profile.light')}</span>
              <span className="material-symbols-outlined text-outline-variant">chevron_right</span>
            </div>
          </div>
        </section>

        <div className="flex justify-center pt-stack-md">
          <button
            onClick={onLogout}
            className="font-label-md text-label-md text-error px-stack-lg py-stack-sm hover:bg-error-container rounded-lg transition-colors"
          >
            {t('profile.logOut')}
          </button>
        </div>
      </main>
      <BottomNav />
    </div>
  )
}