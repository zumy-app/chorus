import { Link } from 'react-router-dom'

export default function Landing() {
  return (
    <div className="min-h-screen bg-gradient-to-b from-white to-gray-50">
      {/* Navigation */}
      <nav className="fixed top-0 w-full bg-white/95 backdrop-blur border-b border-gray-200 z-50">
        <div className="max-w-6xl mx-auto px-6 py-4 flex justify-between items-center">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-gradient-to-br from-indigo-600 to-purple-600 rounded-full flex items-center justify-center">
              <svg className="w-6 h-6 text-white" fill="currentColor" viewBox="0 0 20 20">
                <path d="M2 5a2 2 0 012-2h12a2 2 0 012 2v10a2 2 0 01-2 2H4a2 2 0 01-2-2V5z"></path>
                <path d="M7.5 7.5a1.5 1.5 0 113 0 1.5 1.5 0 01-3 0z"></path>
              </svg>
            </div>
            <span className="text-2xl font-bold bg-gradient-to-r from-indigo-600 to-purple-600 bg-clip-text text-transparent">Chorus</span>
          </div>
          <ul className="hidden md:flex gap-8 items-center">
            <li><a href="#features" className="text-gray-700 hover:text-indigo-600 transition">Features</a></li>
            <li><a href="#how-it-works" className="text-gray-700 hover:text-indigo-600 transition">How It Works</a></li>
            <li><a href="#languages" className="text-gray-700 hover:text-indigo-600 transition">Languages</a></li>
            <li><Link to="/login" className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition">Launch App</Link></li>
          </ul>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="pt-32 pb-20 px-6">
        <div className="max-w-6xl mx-auto">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-12 items-center">
            <div>
              <h1 className="text-5xl md:text-6xl font-bold mb-6">
                Break Language <span className="bg-gradient-to-r from-indigo-600 to-purple-600 bg-clip-text text-transparent">Barriers</span>, Connect Globally
              </h1>
              <p className="text-xl text-gray-600 mb-8">
                Real-time messaging with instant translation in 9 languages. Chat naturally in your language while others read in theirs.
              </p>
              <div className="flex gap-4 mb-12 flex-wrap">
                <Link to="/register" className="px-8 py-4 bg-indigo-600 text-white rounded-lg font-semibold hover:bg-indigo-700 transition text-lg">
                  Get Started Free
                </Link>
                <a href="#how-it-works" className="px-8 py-4 border-2 border-indigo-600 text-indigo-600 rounded-lg font-semibold hover:bg-indigo-50 transition text-lg">
                  See How It Works
                </a>
              </div>
              <div className="flex gap-8">
                <div>
                  <div className="text-3xl font-bold text-indigo-600">9</div>
                  <p className="text-gray-600">Languages</p>
                </div>
                <div>
                  <div className="text-3xl font-bold text-indigo-600">Real-time</div>
                  <p className="text-gray-600">Translation</p>
                </div>
                <div>
                  <div className="text-3xl font-bold text-indigo-600">100%</div>
                  <p className="text-gray-600">Free to Use</p>
                </div>
              </div>
            </div>
            <div className="relative">
              <div className="bg-gradient-to-br from-indigo-600 to-purple-600 rounded-3xl p-1 shadow-2xl">
                <div className="bg-white rounded-3xl p-6">
                  <div className="space-y-4">
                    <div className="bg-gray-100 rounded-lg p-4">
                      <p className="text-sm text-gray-600 mb-1">Hola! ¿Cómo estás?</p>
                      <p className="text-gray-400 text-xs">Hello! How are you?</p>
                    </div>
                    <div className="bg-indigo-600 rounded-lg p-4 ml-8">
                      <p className="text-sm text-white mb-1">I'm great! Learning Spanish</p>
                      <p className="text-indigo-200 text-xs">¡Estoy genial! Aprendiendo español</p>
                    </div>
                    <div className="bg-gray-100 rounded-lg p-4">
                      <p className="text-sm text-gray-600">Fantástico! 🎉</p>
                      <p className="text-gray-400 text-xs">Fantastic! 🎉</p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section id="features" className="py-20 px-6 bg-gray-50">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="text-4xl md:text-5xl font-bold mb-4">Powerful Features for Global Communication</h2>
            <p className="text-xl text-gray-600">Everything you need to connect with people worldwide</p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
            {[
              { icon: '🌐', title: 'Instant Translation', desc: 'Messages are automatically translated to your preferred language in real-time. No delays, no manual selection.' },
              { icon: '✏️', title: 'Grammar Analysis', desc: 'AI-powered grammar checking with CEFR difficulty assessment helps you learn while you chat.' },
              { icon: '📚', title: 'Vocabulary Builder', desc: 'Smart spaced repetition system helps you remember new words and phrases from your conversations.' },
              { icon: '👥', title: 'Group Chats', desc: 'Create multilingual group conversations with up to 100 participants, each reading in their own language.' },
              { icon: '🔍', title: 'Smart Search', desc: 'Find messages across all your chats with full-text search that works in multiple languages.' },
              { icon: '🔒', title: 'Privacy First', desc: 'Your conversations are encrypted and secure. We don\'t store your messages permanently.' },
            ].map((feature, i) => (
              <div key={i} className="bg-white p-8 rounded-2xl shadow hover:shadow-lg transition">
                <div className="text-4xl mb-4">{feature.icon}</div>
                <h3 className="text-xl font-bold mb-2">{feature.title}</h3>
                <p className="text-gray-600">{feature.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* How It Works */}
      <section id="how-it-works" className="py-20 px-6">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="text-4xl md:text-5xl font-bold mb-4">How Chorus Works</h2>
            <p className="text-xl text-gray-600">Start chatting in minutes, no language barriers</p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            {[
              { num: '1', title: 'Sign Up Free', desc: 'Create your account and select your native language and the languages you want to learn.' },
              { num: '2', title: 'Start Chatting', desc: 'Find friends or join groups. Type messages in your language—they\'ll be translated automatically.' },
              { num: '3', title: 'Learn & Grow', desc: 'Save vocabulary, review grammar suggestions, and improve your language skills naturally.' },
            ].map((step, i) => (
              <div key={i} className="text-center">
                <div className="w-16 h-16 bg-gradient-to-br from-indigo-600 to-purple-600 text-white rounded-full flex items-center justify-center mx-auto mb-4 text-2xl font-bold">
                  {step.num}
                </div>
                <h3 className="text-2xl font-bold mb-3">{step.title}</h3>
                <p className="text-gray-600">{step.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Languages Section */}
      <section id="languages" className="py-20 px-6 bg-gray-50">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="text-4xl md:text-5xl font-bold mb-4">Supported Languages</h2>
            <p className="text-xl text-gray-600">Connect with people across 9 major languages</p>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-9 gap-4">
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
              <div key={i} className="bg-white p-6 rounded-xl text-center shadow hover:shadow-lg transition">
                <div className="text-4xl mb-2">{lang.flag}</div>
                <p className="font-semibold text-gray-800">{lang.name}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20 px-6 bg-gradient-to-r from-indigo-600 to-purple-600">
        <div className="max-w-4xl mx-auto text-center text-white">
          <h2 className="text-4xl md:text-5xl font-bold mb-4">Ready to Break Language Barriers?</h2>
          <p className="text-xl mb-8 opacity-90">Join Chorus today and start connecting with people worldwide</p>
          <Link to="/register" className="px-8 py-4 bg-white text-indigo-600 rounded-lg font-bold text-lg hover:bg-gray-100 transition inline-block">
            Get Started Now
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-gray-900 text-white py-12 px-6">
        <div className="max-w-6xl mx-auto">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-8 mb-8">
            <div>
              <div className="flex items-center gap-2 mb-4">
                <div className="w-8 h-8 bg-gradient-to-br from-indigo-600 to-purple-600 rounded-full"></div>
                <span className="font-bold text-lg">Chorus</span>
              </div>
              <p className="text-gray-400">Break language barriers and connect with people worldwide through real-time translation.</p>
            </div>
            <div>
              <h4 className="font-bold mb-4">Product</h4>
              <ul className="space-y-2 text-gray-400">
                <li><a href="#features" className="hover:text-white">Features</a></li>
                <li><Link to="/login" className="hover:text-white">Web App</Link></li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold mb-4">Company</h4>
              <ul className="space-y-2 text-gray-400">
                <li><a href="#how-it-works" className="hover:text-white">How It Works</a></li>
                <li><a href="#languages" className="hover:text-white">Languages</a></li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold mb-4">Support</h4>
              <ul className="space-y-2 text-gray-400">
                <li><a href="http://localhost:8080/health" className="hover:text-white">API Status</a></li>
              </ul>
            </div>
          </div>
          <div className="border-t border-gray-800 pt-8 text-center text-gray-400">
            <p>&copy; 2026 Chorus. All rights reserved.</p>
          </div>
        </div>
      </footer>
    </div>
  )
}
