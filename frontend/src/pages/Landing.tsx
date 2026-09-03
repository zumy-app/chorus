import { Link } from 'react-router-dom'

export default function Landing() {
  return (
    <div className="min-h-screen bg-surface text-on-surface">
      {/* TopNavBar - wireframe code.html:115-132 */}
      <header className="sticky top-0 w-full z-50 bg-surface/80 backdrop-blur-md shadow-sm">
        <div className="flex justify-between items-center px-6 py-4 max-w-7xl mx-auto">
          <div className="flex items-center gap-2">
            <img
              alt="Chorus Logo"
              className="h-8"
              src="https://lh3.googleusercontent.com/aida-public/AB6AXuDa6GYGIgVtywOaPb01RW695oait4jcaavx16r_nKo33XbyRf-0zAaymauW4w884StH5gn8sDcki7Zh0IoCPttuAck8NYnvSGCjj7JCXGVuxenqclT71o6xDo7KjLR_uzsdmcZfgviMidS0oJshP79Ss3UoB_Nkh0zH76bKG_esCs2PbivF_pwTdIoaEtCFA4ZJGbkL34pFiRw0v2wNIFWRNo4x_k1f4Dyk3oP0JVWWdVXde1-DK2G6"
            />
          </div>
          <nav className="hidden md:flex items-center gap-8">
            <a className="text-on-surface-variant hover:text-primary transition-colors duration-200" href="#features">
              Features
            </a>
            <a className="text-on-surface-variant hover:text-primary transition-colors duration-200" href="#pricing">
              Pricing
            </a>
            <a className="text-on-surface-variant hover:text-primary transition-colors duration-200" href="#about">
              About Us
            </a>
          </nav>
          <div className="flex items-center gap-4">
            <Link to="/register">
              <button className="bg-primary text-on-primary font-label-md px-6 py-2.5 rounded-full hover:opacity-90 transition-opacity duration-200 shadow-sm">
                Get Started
              </button>
            </Link>
          </div>
        </div>
      </header>

      <main>
        {/* Hero Section - code.html:134-163 */}
        <section className="relative pt-24 pb-32 overflow-hidden bg-surface-container-low">
          <div className="max-w-7xl mx-auto px-6 grid md:grid-cols-2 gap-12 items-center relative z-10">
            <div className="space-y-8">
              <h1 className="font-headline-lg text-headline-lg md:text-5xl lg:text-6xl text-on-surface leading-tight tracking-tight">
                Communication is Learning. <br /> <span className="text-primary">Redefining how we acquire language.</span>
              </h1>
              <p className="font-body-lg text-body-lg text-on-surface-variant max-w-xl">
                Bridging the gap between messaging apps and learning platforms. We turn your daily conversations into a personalized learning journey, making communication and learning the exact same function.
              </p>
              <div className="flex flex-col sm:flex-row gap-4 pt-4">
                <Link to="/register">
                  <button className="bg-primary text-on-primary font-label-md px-8 py-4 rounded-full hover:bg-primary/90 transition-colors shadow-md text-center">
                    Start Your Journey
                  </button>
                </Link>
                <button className="bg-transparent border-2 border-outline-variant text-primary font-label-md px-8 py-4 rounded-full hover:bg-surface-variant/50 transition-colors flex items-center justify-center gap-2">
                  <span className="material-symbols-outlined text-[20px]">play_circle</span>
                  Watch Demo
                </button>
              </div>
            </div>
            <div className="relative">
              <div className="absolute inset-0 bg-primary/5 rounded-3xl blur-3xl transform rotate-3"></div>
              <div className="relative rounded-2xl shadow-[0_8px_30px_rgb(0,0,0,0.08)] overflow-hidden">
                <img
                  alt="Brain Neural Pathways"
                  className="w-full h-auto object-cover"
                  src="https://lh3.googleusercontent.com/aida-public/AB6AXuDgQhw3sVSThrLEgije7dkEJr4B-KhL_jlmgoT7OCR-tu9Wg0-ZO2dCEUdRRXZtNDF1dZwNu2b_FAx2GdcCxm6CoPp34KNd6PLqadPWRBPd4j59XdYzmvDrD0ZwSt5MdqajfdJTvPtv7l5cJy0RUMrRtxQaYC4KwOAcAgo60N9p5sY_K985F67YZHqu-axUbl3PaATcc56Db3G9uFiF01Mlr7_6otiEFrdiqNS1TChuz0OZhchB31FT"
                />
              </div>
            </div>
          </div>
          <div className="absolute top-1/2 right-0 -translate-y-1/2 translate-x-1/3 w-96 h-96 bg-secondary/10 rounded-full blur-3xl pointer-events-none"></div>
        </section>

        {/* Problem / Solution - Bridging - code.html:164-173 */}
        <section className="py-24 bg-surface">
          <div className="max-w-4xl mx-auto px-6 text-center space-y-6">
            <h2 className="font-headline-md text-headline-md text-on-surface">
              Bridging Messaging and Learning. <br className="hidden sm:block" /> <span className="text-primary">The Best of Both Worlds.</span>
            </h2>
            <p className="font-body-lg text-body-lg text-on-surface-variant">
              Why choose between a messenger like WhatsApp and a learning tool like Duolingo? Chorus combines them. We analyze your actual, real-world conversations to build vocabulary and grammar lessons based exclusively on the language <em>you</em> need, making the act of communicating and learning one seamless experience.
            </p>
          </div>
        </section>

        {/* Features Grid - Ecosystem - code.html:174-220 */}
        <section className="py-24 bg-surface-container-lowest" id="features">
          <div className="max-w-7xl mx-auto px-6">
            <div className="text-center mb-16">
              <h2 className="font-headline-lg text-headline-lg text-on-surface">A Complete Language Ecosystem</h2>
              <p className="font-body-md text-on-surface-variant mt-4 mb-12">Everything you need to go from basic phrases to true fluency.</p>
              <div className="max-w-4xl mx-auto mb-16 rounded-2xl overflow-hidden shadow-lg border border-outline-variant/30">
                <img
                  alt="Chorus App Mockup"
                  className="w-full h-auto object-cover"
                  src="https://lh3.googleusercontent.com/aida-public/AB6AXuBiIW1Pswg-d9O3dPkH_pRY26cjhtNRzdzPlybhdzB1bWm3ps3BcDeReivUrvxFJJb4cMfNDyX0at7osxAqWO_kXG0pEDgNWdOf2bFRW1RevouA_h6KZB1Zsi8Vs2Rug8O_vFqO_XG0pEDgNWdOf2bFRW1RevouA_h6KZB1Zsi8Vs2Rug8O_vjxj-gCnyzbMaEiyc-C97oVFoNG8qbRqPArY4brforVqA2VZXHQTsTxaSpeQVGJFioF1OmlsYkD44Nn3ONhcNUHnp0FLlg9uU8L_Y8GSjYxxzcuD_XLd"
                />
              </div>
            </div>
            <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
              {/* AI Deep Dive */}
              <div className="bg-surface-container-low p-8 rounded-2xl border border-outline-variant/30 hover:shadow-md transition-shadow">
                <div className="w-12 h-12 bg-primary/10 rounded-xl flex items-center justify-center mb-6">
                  <span className="material-symbols-outlined text-primary text-[24px]">analytics</span>
                </div>
                <h3 className="font-headline-sm text-headline-sm text-on-surface mb-3">AI Deep Dive</h3>
                <p className="font-body-sm text-on-surface-variant">Instant grammar analysis and CEFR-aligned drills generated from your chat history.</p>
              </div>
              {/* Real Talk */}
              <div className="bg-surface-container-low p-8 rounded-2xl border border-outline-variant/30 hover:shadow-md transition-shadow">
                <div className="w-12 h-12 bg-secondary/10 rounded-xl flex items-center justify-center mb-6">
                  <span className="material-symbols-outlined text-secondary text-[24px]">forum</span>
                </div>
                <h3 className="font-headline-sm text-headline-sm text-on-surface mb-3">Real Talk</h3>
                <p className="font-body-sm text-on-surface-variant">AI-guided roleplays for real-world scenarios. Practice before you have to perform.</p>
              </div>
              {/* Teacher Marketplace */}
              <div className="bg-surface-container-low p-8 rounded-2xl border border-outline-variant/30 hover:shadow-md transition-shadow">
                <div className="w-12 h-12 bg-tertiary/10 rounded-xl flex items-center justify-center mb-6">
                  <span className="material-symbols-outlined text-tertiary text-[24px]">school</span>
                </div>
                <h3 className="font-headline-sm text-headline-sm text-on-surface mb-3">Teacher Marketplace</h3>
                <p className="font-body-sm text-on-surface-variant">Book 1:1 sessions with professional tutors who can see your progress data and tailor lessons.</p>
              </div>
              {/* Phase 2 Ready */}
              <div className="bg-surface-container-low p-8 rounded-2xl border border-outline-variant/30 hover:shadow-md transition-shadow relative overflow-hidden">
                <div className="absolute top-0 right-0 bg-primary text-on-primary font-label-sm px-3 py-1 rounded-bl-lg">Coming Soon</div>
                <div className="w-12 h-12 bg-surface-variant rounded-xl flex items-center justify-center mb-6">
                  <span className="material-symbols-outlined text-on-surface-variant text-[24px]">video_call</span>
                </div>
                <h3 className="font-headline-sm text-headline-sm text-on-surface mb-3">Phase 2 Ready</h3>
                <p className="font-body-sm text-on-surface-variant">High-fidelity voice &amp; video calls with live translated captions and pronunciation feedback.</p>
              </div>
            </div>
          </div>
        </section>

        {/* Pricing Section - code.html:221-294 */}
        <section className="py-24 bg-surface" id="pricing">
          <div className="max-w-5xl mx-auto px-6">
            <div className="text-center mb-16">
              <h2 className="font-headline-lg text-headline-lg text-on-surface">Simple, Transparent Pricing</h2>
              <p className="font-body-md text-on-surface-variant mt-4">Start for free, upgrade when you&apos;re ready to accelerate.</p>
            </div>
            <div className="grid md:grid-cols-2 gap-8 max-w-3xl mx-auto">
              {/* Free Tier */}
              <div className="bg-surface-container-lowest p-8 rounded-3xl border border-outline-variant/50 shadow-sm flex flex-col">
                <div className="mb-6">
                  <h3 className="font-headline-md text-headline-md text-on-surface">Free</h3>
                  <div className="mt-4 flex items-baseline text-on-surface">
                    <span className="text-4xl font-bold tracking-tight">$0</span>
                    <span className="text-on-surface-variant ml-1 font-body-sm">/month</span>
                  </div>
                  <p className="font-body-sm text-on-surface-variant mt-2">Essential features to start your journey.</p>
                </div>
                <ul className="space-y-4 mb-8 flex-1">
                  <li className="flex items-start gap-3">
                    <span className="material-symbols-outlined text-tertiary text-[20px] mt-0.5">check_circle</span>
                    <span className="font-body-sm text-on-surface">280-character messages</span>
                  </li>
                  <li className="flex items-start gap-3">
                    <span className="material-symbols-outlined text-tertiary text-[20px] mt-0.5">check_circle</span>
                    <span className="font-body-sm text-on-surface">Basic AI translations</span>
                  </li>
                  <li className="flex items-start gap-3">
                    <span className="material-symbols-outlined text-tertiary text-[20px] mt-0.5">check_circle</span>
                    <span className="font-body-sm text-on-surface">Limited daily AI insights</span>
                  </li>
                </ul>
                <button className="w-full bg-surface-container-high text-on-surface font-label-md py-3 rounded-full hover:bg-surface-variant transition-colors">
                  Get Started Free
                </button>
              </div>
              {/* Premium Tier */}
              <div className="bg-primary/5 p-8 rounded-3xl border-2 border-primary relative flex flex-col shadow-md">
                <div className="absolute -top-4 left-1/2 -translate-x-1/2 bg-primary text-on-primary font-label-sm px-4 py-1 rounded-full shadow-sm">
                  Most Popular
                </div>
                <div className="mb-6">
                  <h3 className="font-headline-md text-headline-md text-primary">Premium</h3>
                  <div className="mt-4 flex items-baseline text-on-surface">
                    <span className="text-4xl font-bold tracking-tight">$7.99</span>
                    <span className="text-on-surface-variant ml-1 font-body-sm">/month</span>
                  </div>
                  <p className="font-body-sm text-on-surface-variant mt-2">Unleash the full power of the AI tutor.</p>
                </div>
                <ul className="space-y-4 mb-8 flex-1">
                  <li className="flex items-start gap-3">
                    <span className="material-symbols-outlined text-primary text-[20px] mt-0.5">check_circle</span>
                    <span className="font-body-sm text-on-surface font-medium">1000-character messages</span>
                  </li>
                  <li className="flex items-start gap-3">
                    <span className="material-symbols-outlined text-primary text-[20px] mt-0.5">check_circle</span>
                    <span className="font-body-sm text-on-surface font-medium">Unlimited AI Deep Dives</span>
                  </li>
                  <li className="flex items-start gap-3">
                    <span className="material-symbols-outlined text-primary text-[20px] mt-0.5">check_circle</span>
                    <span className="font-body-sm text-on-surface font-medium">Monthly trial credits for live tutors</span>
                  </li>
                  <li className="flex items-start gap-3">
                    <span className="material-symbols-outlined text-primary text-[20px] mt-0.5">check_circle</span>
                    <span className="font-body-sm text-on-surface font-medium">Reduced marketplace fees</span>
                  </li>
                </ul>
                <button className="w-full bg-primary text-on-primary font-label-md py-3 rounded-full hover:bg-primary/90 transition-colors shadow-sm">
                  Upgrade to Premium
                </button>
              </div>
            </div>
          </div>
        </section>

        {/* About Us - Our Mission - code.html:295-303 */}
        <section className="py-24 bg-surface-container-lowest border-t border-outline-variant/20" id="about">
          <div className="max-w-4xl mx-auto px-6 text-center">
            <h2 className="font-headline-md text-headline-md text-on-surface mb-6">Our Mission</h2>
            <p className="font-body-lg text-body-lg text-on-surface-variant max-w-2xl mx-auto">
              We believe language shouldn&apos;t be a barrier, but a bridge. Chorus was built by a team of linguists and engineers dedicated to bridging global communication gaps through science-based acquisition, not rote memorization.
            </p>
          </div>
        </section>

        {/* Final CTA - code.html:304-315 */}
        <section className="py-32 bg-primary text-on-primary relative overflow-hidden">
          <div className="absolute inset-0 opacity-10" style={{ backgroundImage: 'radial-gradient(circle at 2px 2px, white 1px, transparent 0)', backgroundSize: '24px 24px' }}></div>
          <div className="max-w-3xl mx-auto px-6 text-center relative z-10">
            <h2 className="font-headline-lg text-headline-lg mb-8">Ready to reach fluency?</h2>
            <p className="font-body-lg mb-10 text-on-primary/90">Join thousands of learners who have transformed their daily chats into a masterclass.</p>
            <Link to="/register">
              <button className="bg-surface-container-lowest text-primary font-label-md px-10 py-4 rounded-full hover:bg-surface transition-colors shadow-lg text-lg">
                Get Started Now
              </button>
            </Link>
          </div>
        </section>
      </main>

      {/* Footer - code.html:317-346 */}
      <footer className="w-full bg-surface-container-low border-t border-outline-variant">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-8 px-6 py-12 max-w-7xl mx-auto">
          {/* Brand Column */}
          <div className="col-span-1 md:col-span-1 flex flex-col gap-4">
            <a href="/" className="font-headline-sm text-headline-sm text-primary">
              Chorus
            </a>
            <p className="font-body-sm text-body-sm text-on-surface-variant">© 2024 Chorus AI. Language learning reimagined.</p>
          </div>
          {/* Links Columns */}
          <div className="col-span-1 md:col-span-3 grid grid-cols-2 sm:grid-cols-3 gap-8">
            <div className="flex flex-col gap-3">
              <span className="font-label-md text-on-surface font-bold uppercase tracking-wider mb-2">Product</span>
              <a className="font-body-sm text-body-sm text-on-surface-variant hover:text-primary hover:underline transition-all opacity-90 hover:opacity-100" href="#features">
                Features
              </a>
              <a className="font-body-sm text-body-sm text-on-surface-variant hover:text-primary hover:underline transition-all opacity-90 hover:opacity-100" href="#pricing">
                Pricing
              </a>
            </div>
            <div className="flex flex-col gap-3">
              <span className="font-label-md text-on-surface font-bold uppercase tracking-wider mb-2">Company</span>
              <a className="font-body-sm text-body-sm text-on-surface-variant hover:text-primary hover:underline transition-all opacity-90 hover:opacity-100" href="#about">
                About Us
              </a>
              <a className="font-body-sm text-body-sm text-on-surface-variant hover:text-primary hover:underline transition-all opacity-90 hover:opacity-100" href="#">
                Privacy Policy
              </a>
              <a className="font-body-sm text-body-sm text-on-surface-variant hover:text-primary hover:underline transition-all opacity-90 hover:opacity-100" href="#">
                Terms of Service
              </a>
            </div>
            <div className="flex flex-col gap-3">
              <span className="font-label-md text-on-surface font-bold uppercase tracking-wider mb-2">Support</span>
              <a className="font-body-sm text-body-sm text-on-surface-variant hover:text-primary hover:underline transition-all opacity-90 hover:opacity-100" href="#">
                Help Center
              </a>
            </div>
          </div>
        </div>
      </footer>
    </div>
  )
}
