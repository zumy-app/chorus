import { useTranslation } from 'react-i18next'

export default function About() {
  const { t } = useTranslation()
  return (
    <div className="min-h-screen bg-gradient-to-b from-white to-gray-50">
      <div className="max-w-3xl mx-auto px-6 py-16">
        <div className="text-center mb-12">
          <div className="w-20 h-20 bg-gradient-to-br from-indigo-600 to-purple-600 rounded-full flex items-center justify-center mx-auto mb-6">
            <svg className="w-10 h-10 text-white" fill="currentColor" viewBox="0 0 20 20">
              <path d="M2 5a2 2 0 012-2h12a2 2 0 012 2v10a2 2 0 01-2 2H4a2 2 0 01-2-2V5z"></path>
              <path d="M7.5 7.5a1.5 1.5 0 113 0 1.5 1.5 0 01-3 0z"></path>
            </svg>
          </div>
          <h1 className="text-4xl font-bold text-gray-900 mb-4">{t('about.title')}</h1>
          <p className="text-xl text-gray-600">{t('about.subtitle')}</p>
        </div>

        <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-8 mb-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">{t('about.mission')}</h2>
          <p className="text-gray-600 leading-relaxed mb-6">
            {t('about.missionP1')}
          </p>
          <p className="text-gray-600 leading-relaxed">
            {t('about.missionP2')}
          </p>
        </div>

        <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-8 mb-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">{t('about.features')}</h2>
          <ul className="space-y-4">
            {[
              { icon: '🌐', title: t('about.featuresItems.translation.title'), desc: t('about.featuresItems.translation.desc') },
              { icon: '✏️', title: t('about.featuresItems.grammar.title'), desc: t('about.featuresItems.grammar.desc') },
              { icon: '📚', title: t('about.featuresItems.vocabulary.title'), desc: t('about.featuresItems.vocabulary.desc') },
              { icon: '👥', title: t('about.featuresItems.groups.title'), desc: t('about.featuresItems.groups.desc') },
              { icon: '🔒', title: t('about.featuresItems.privacy.title'), desc: t('about.featuresItems.privacy.desc') },
            ].map((feature, i) => (
              <li key={i} className="flex items-start space-x-4">
                <span className="text-2xl">{feature.icon}</span>
                <div>
                  <h3 className="font-semibold text-gray-900">{feature.title}</h3>
                  <p className="text-gray-600 text-sm">{feature.desc}</p>
                </div>
              </li>
            ))}
          </ul>
        </div>

        <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-8 mb-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">{t('about.supportedLanguages')}</h2>
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
              <div key={i} className="text-center p-3 bg-gray-50 rounded-lg">
                <div className="text-2xl mb-1">{lang.flag}</div>
                <div className="text-sm font-medium text-gray-700">{lang.name}</div>
              </div>
            ))}
          </div>
        </div>

        <div className="text-center text-gray-500 text-sm">
          <p>{t('about.version')}</p>
          <p className="mt-1">{t('about.copyright')}</p>
        </div>
      </div>
    </div>
  )
}
