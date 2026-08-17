import { useTranslation } from 'react-i18next'

export default function About() {
  const { t } = useTranslation()
  return (
    <div className="min-h-screen bg-background text-on-surface">
      <div className="max-w-3xl mx-auto px-6 py-16">
        <div className="text-center mb-12">
          <div className="w-20 h-20 bg-primary-container/10 rounded-full flex items-center justify-center mx-auto mb-6">
            <span className="material-symbols-outlined text-primary text-5xl" style={{ fontVariationSettings: "'FILL' 1" }}>graphic_eq</span>
          </div>
          <h1 className="text-4xl font-bold mb-4">{t('about.title')}</h1>
          <p className="text-body-lg text-on-surface-variant">{t('about.subtitle')}</p>
        </div>

        <div className="bg-surface-container-lowest rounded-2xl shadow-[0px_4px_12px_rgba(0,0,0,0.05)] border border-outline-variant/40 p-8 mb-8">
          <h2 className="text-2xl font-bold mb-4">{t('about.mission')}</h2>
          <p className="text-on-surface-variant leading-relaxed mb-6">
            {t('about.missionP1')}
          </p>
          <p className="text-on-surface-variant leading-relaxed">
            {t('about.missionP2')}
          </p>
        </div>

        <div className="bg-surface-container-lowest rounded-2xl shadow-[0px_4px_12px_rgba(0,0,0,0.05)] border border-outline-variant/40 p-8 mb-8">
          <h2 className="text-2xl font-bold mb-4">{t('about.features')}</h2>
          <ul className="space-y-4">
            {[
              { icon: '🌐', title: t('about.featuresItems.translation.title'), desc: t('about.featuresItems.translation.desc') },
              { icon: '✏️', title: t('about.featuresItems.grammar.title'), desc: t('about.featuresItems.grammar.desc') },
              { icon: '📚', title: t('about.featuresItems.vocabulary.title'), desc: t('about.featuresItems.vocabulary.desc') },
              { icon: '👥', title: t('about.featuresItems.groups.title'), desc: t('about.featuresItems.groups.desc') },
              { icon: '🔒', title: t('about.featuresItems.privacy.title'), desc: t('about.featuresItems.privacy.desc') },
            ].map((feature, i) => (
              <li key={i} className="flex items-start space-x-4">
                <div className={`w-11 h-11 rounded-full ${i % 2 ? 'bg-secondary-container/10' : 'bg-primary-container/10'} flex items-center justify-center shrink-0`}>
                  <span className="text-xl">{feature.icon}</span>
                </div>
                <div>
                  <h3 className="font-semibold">{feature.title}</h3>
                  <p className="text-on-surface-variant text-sm">{feature.desc}</p>
                </div>
              </li>
            ))}
          </ul>
        </div>

        <div className="bg-surface-container-lowest rounded-2xl shadow-[0px_4px_12px_rgba(0,0,0,0.05)] border border-outline-variant/40 p-8 mb-8">
          <h2 className="text-2xl font-bold mb-4">{t('about.supportedLanguages')}</h2>
          <div className="grid grid-cols-3 gap-4">
            {[
              { flag: '🇬🇧', name: 'English' },
              { flag: '🇪🇸', name: 'Spanish' },
              { flag: '🇫🇷', name: 'French' },
              { flag: '🇩🇪', name: 'German' },
              { flag: '🇮🇹', name: 'Italian' },
              { flag: '🇵🇹', name: 'Portuguese' },
              { flag: '🇯🇵', name: 'Japanese' },
              { flag: '🇰🇷', name: 'Korean' },
              { flag: '🇨🇳', name: 'Chinese' },
            ].map((lang, i) => (
              <div key={i} className="text-center p-3 bg-surface-container-low rounded-lg">
                <div className="text-2xl mb-1">{lang.flag}</div>
                <div className="text-sm font-medium text-on-surface">{lang.name}</div>
              </div>
            ))}
          </div>
        </div>

        <div className="text-center text-on-surface-variant text-sm">
          <p>{t('about.version')}</p>
          <p className="mt-1">{t('about.copyright')}</p>
        </div>
      </div>
    </div>
  )
}
