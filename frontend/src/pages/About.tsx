export default function About() {
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
          <h1 className="text-4xl font-bold text-gray-900 mb-4">About Chorus</h1>
          <p className="text-xl text-gray-600">Breaking language barriers through real-time communication</p>
        </div>

        <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-8 mb-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">Our Mission</h2>
          <p className="text-gray-600 leading-relaxed mb-6">
            Chorus is a multilingual messenger designed to connect people across language barriers. 
            We believe that language should never be a barrier to meaningful communication. 
            Our platform provides real-time translation in 9 languages, making it easy for anyone 
            to chat with people from around the world.
          </p>
          <p className="text-gray-600 leading-relaxed">
            Whether you're learning a new language, connecting with international friends, 
            or doing business across borders, Chorus makes communication seamless and natural.
          </p>
        </div>

        <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-8 mb-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">Key Features</h2>
          <ul className="space-y-4">
            {[
              { icon: '🌐', title: 'Instant Translation', desc: 'Messages are automatically translated to your preferred language in real-time.' },
              { icon: '✏️', title: 'Grammar Analysis', desc: 'AI-powered grammar checking with CEFR difficulty assessment helps you learn while you chat.' },
              { icon: '📚', title: 'Vocabulary Builder', desc: 'Smart spaced repetition system helps you remember new words from your conversations.' },
              { icon: '👥', title: 'Group Chats', desc: 'Create multilingual group conversations with up to 100 participants.' },
              { icon: '🔒', title: 'Privacy First', desc: 'Your conversations are encrypted and secure.' },
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
          <h2 className="text-2xl font-bold text-gray-900 mb-4">Supported Languages</h2>
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
          <p>Version 2.0.0</p>
          <p className="mt-1">© 2026 Chorus. All rights reserved.</p>
        </div>
      </div>
    </div>
  )
}
