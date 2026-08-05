import { Link } from 'react-router-dom'
import { useState } from 'react'
import { detectBrowserLanguage, getNativeLanguageName, SUPPORTED_LANGUAGES } from '../services/language'
import LanguageSelector from '../components/LanguageSelector'

// =============================================================================
// Full-page internationalization for the landing site.
// Every visible string on the page is translated for the top-10 world
// languages (matches LibreTranslate LT_LOAD_ONLY). Unsupported languages fall
// back to English. Selecting a language in the navbar changes ALL sections —
// nav, hero, stats, features, how-it-works, languages, CTA and footer.
// =============================================================================
interface LandingStrings {
  nav: { features: string; how: string; languages: string; launch: string }
  hero: {
    badge: string // "Updated: {count} languages"
    titleA: string
    titleB: string
    subtitle: string
    cta: string
    seeHow: string
    stats: { languages: string; translation: string; free: string }
    chat: { bubble1: string; bubble1Trans: string; bubble2: string; bubble2Trans: string; bubble3: string; bubble3Trans: string }
  }
  features: { title: string; subtitle: string; items: { icon: string; title: string; desc: string }[] }
  how: { title: string; subtitle: string; steps: { num: string; title: string; desc: string }[] }
  languages: { title: string; subtitle: string }
  cta: { title: string; subtitle: string; button: string }
  footer: { tagline: string; product: string; productLinks: { label: string; href: string }[]; company: string; companyLinks: { label: string; href: string }[]; support: string; supportLinks: { label: string; href: string }[]; rights: string }
}

// Ordered top-10 list for consistent display.
const TOP10 = ['en', 'zh', 'hi', 'es', 'ar', 'fr', 'bn', 'pt', 'ru', 'ur']

const STRINGS: Record<string, LandingStrings> = {
  en: {
    nav: { features: 'Features', how: 'How It Works', languages: 'Languages', launch: 'Launch App' },
    hero: {
      badge: 'Available in {count} languages',
      titleA: 'Break Language Barriers',
      titleB: 'Connect Globally',
      subtitle: 'Real-time messaging with instant translation in 10 major languages. Chat naturally in your language while others read in theirs.',
      cta: 'Get Started Free',
      seeHow: 'See How It Works',
      stats: { languages: 'Languages', translation: 'Real-time', free: 'Free to Use' },
      chat: {
        bubble1: '¡Hola! ¿Cómo estás?', bubble1Trans: 'Hello! How are you?',
        bubble2: "I'm great! Learning Spanish", bubble2Trans: '¡Estoy genial! Aprendiendo español',
        bubble3: 'Fantástico! 🎉', bubble3Trans: 'Fantastic! 🎉',
      },
    },
    features: {
      title: 'Powerful Features for Global Communication',
      subtitle: 'Everything you need to connect with people worldwide',
      items: [
        { icon: '🌐', title: 'Instant Translation', desc: 'Messages are automatically translated to your preferred language in real-time. No delays, no manual selection.' },
        { icon: '✏️', title: 'Grammar Analysis', desc: 'AI-powered grammar checking with CEFR difficulty assessment helps you learn while you chat.' },
        { icon: '📚', title: 'Vocabulary Builder', desc: 'Smart spaced repetition system helps you remember new words and phrases from your conversations.' },
        { icon: '👥', title: 'Group Chats', desc: 'Create multilingual group conversations with up to 100 participants, each reading in their own language.' },
        { icon: '🔍', title: 'Smart Search', desc: 'Find messages across all your chats with full-text search that works in multiple languages.' },
        { icon: '🔒', title: 'Privacy First', desc: 'Your conversations are encrypted and secure. We don\'t store your messages permanently.' },
      ],
    },
    how: {
      title: 'How Chorus Works',
      subtitle: 'Start chatting in minutes, no language barriers',
      steps: [
        { num: '1', title: 'Sign Up Free', desc: 'Create your account and select your native language and the languages you want to learn.' },
        { num: '2', title: 'Start Chatting', desc: 'Find friends or join groups. Type messages in your language—they\'ll be translated automatically.' },
        { num: '3', title: 'Learn & Grow', desc: 'Save vocabulary, review grammar suggestions, and improve your language skills naturally.' },
      ],
    },
    languages: { title: 'Supported Languages', subtitle: 'Connect with people across 10 major languages' },
    cta: { title: 'Ready to Break Language Barriers?', subtitle: 'Join Chorus today and start connecting with people worldwide', button: 'Get Started Now' },
    footer: {
      tagline: 'Break language barriers and connect with people worldwide through real-time translation.',
      product: 'Product',
      productLinks: [{ label: 'Features', href: '#features' }, { label: 'Web App', href: '/login' }],
      company: 'Company',
      companyLinks: [{ label: 'How It Works', href: '#how-it-works' }, { label: 'Languages', href: '#languages' }],
      support: 'Support',
      supportLinks: [{ label: 'API Status', href: 'http://localhost:8080/health' }],
      rights: '© 2026 Chorus. All rights reserved.',
    },
  },

  zh: {
    nav: { features: '功能', how: '工作原理', languages: '语言', launch: '启动应用' },
    hero: {
      badge: '支持 {count} 种语言',
      titleA: '打破语言障碍',
      titleB: '连接全球',
      subtitle: '支持10种主要语言的实时消息即时翻译。用你的语言自然聊天，对方用他们的语言阅读。',
      cta: '免费开始',
      seeHow: '了解工作原理',
      stats: { languages: '语言', translation: '实时', free: '永久免费' },
      chat: {
        bubble1: '¡Hola! ¿Cómo estás?', bubble1Trans: '你好！你好吗？',
        bubble2: '我很好！正在学西班牙语', bubble2Trans: '¡Estoy genial! Aprendiendo español',
        bubble3: '太棒了！🎉', bubble3Trans: 'Fantástico! 🎉',
      },
    },
    features: {
      title: '强大的全球沟通功能',
      subtitle: '连接世界各地的人所需的一切',
      items: [
        { icon: '🌐', title: '即时翻译', desc: '消息实时自动翻译成您的首选语言。无延迟，无需手动选择。' },
        { icon: '✏️', title: '语法分析', desc: 'AI语法检查与CEFR难度评估，帮助您在聊天中学习。' },
        { icon: '📚', title: '词汇构建器', desc: '智能间隔重复系统帮助您记住对话中的新单词和短语。' },
        { icon: '👥', title: '群组聊天', desc: '创建最多100名参与者的多语言群组对话，每个人用自己的语言阅读。' },
        { icon: '🔍', title: '智能搜索', desc: '支持多种语言的全文搜索，在所有聊天中查找消息。' },
        { icon: '🔒', title: '隐私优先', desc: '您的对话经过加密且安全。我们不永久存储您的消息。' },
      ],
    },
    how: {
      title: 'Chorus 工作原理',
      subtitle: '几分钟内开始聊天，没有语言障碍',
      steps: [
        { num: '1', title: '免费注册', desc: '创建账户并选择您的母语和想学习的语言。' },
        { num: '2', title: '开始聊天', desc: '找朋友或加入群组。用您的语言输入消息——它们会自动翻译。' },
        { num: '3', title: '学习成长', desc: '保存词汇、查看语法建议，自然地提升语言技能。' },
      ],
    },
    languages: { title: '支持的语言', subtitle: '连接10种主要语言的人们' },
    cta: { title: '准备好打破语言障碍了吗？', subtitle: '立即加入 Chorus，开始与世界各地的朋友联系', button: '立即开始' },
    footer: {
      tagline: '通过实时翻译打破语言障碍，与世界各地的朋友联系。',
      product: '产品',
      productLinks: [{ label: '功能', href: '#features' }, { label: '网页应用', href: '/login' }],
      company: '公司',
      companyLinks: [{ label: '工作原理', href: '#how-it-works' }, { label: '语言', href: '#languages' }],
      support: '支持',
      supportLinks: [{ label: 'API 状态', href: 'http://localhost:8080/health' }],
      rights: '© 2026 Chorus. 保留所有权利。',
    },
  },

  hi: {
    nav: { features: 'विशेषताएं', how: 'यह कैसे काम करता है', languages: 'भाषाएं', launch: 'ऐप शुरू करें' },
    hero: {
      badge: '{count} भाषाओं में उपलब्ध',
      titleA: 'भाषा की बाधाएं तोड़ें',
      titleB: 'वैश्विक रूप से जुड़ें',
      subtitle: '10 प्रमुख भाषाओं में त्वरित अनुवाद के साथ रीयल-टाइम मैसेजिंग। अपनी भाषा में स्वाभाविक रूप से चैट करें जबकि अन्य अपनी भाषा में पढ़ें।',
      cta: 'मुफ्त में शुरू करें',
      seeHow: 'यह कैसे काम करता है देखें',
      stats: { languages: 'भाषाएं', translation: 'रीयल-टाइम', free: 'उपयोग मुफ्त' },
      chat: {
        bubble1: '¡Hola! ¿Cómo estás?', bubble1Trans: 'नमस्ते! आप कैसे हैं?',
        bubble2: 'मैं बढ़िया हूँ! स्पेनिश सीख रहा हूँ', bubble2Trans: '¡Estoy genial! Aprendiendo español',
        bubble3: 'बढ़िया! 🎉', bubble3Trans: 'Fantástico! 🎉',
      },
    },
    features: {
      title: 'वैश्विक संचार के लिए शक्तिशाली विशेषताएं',
      subtitle: 'दुनिया भर के लोगों से जुड़ने के लिए सब कुछ',
      items: [
        { icon: '🌐', title: 'तुरंत अनुवाद', desc: 'संदेश स्वचालित रूप से आपकी पसंदीदा भाषा में रीयल-टाइम अनुवादित होते हैं। कोई देरी नहीं।' },
        { icon: '✏️', title: 'व्याकरण विश्लेषण', desc: 'AI-संचालित व्याकरण जांच और CEFR कठिनाई मूल्यांकन आपको चैट करते समय सीखने में मदद करता है।' },
        { icon: '📚', title: 'शब्दावली निर्माता', desc: 'स्मार्ट स्पेस्ड रिपीटिशन सिस्टम आपको बातचीत से नए शब्द और वाक्यांश याद रखने में मदद करता है।' },
        { icon: '👥', title: 'समूह चैट', desc: '100 प्रतिभागियों तक के बहुभाषी समूह वार्तालाप बनाएं, हर कोई अपनी भाषा में पढ़े।' },
        { icon: '🔍', title: 'स्मार्ट खोज', desc: 'कई भाषाओं में काम करने वाले फुल-टेक्स्ट खोज के साथ सभी चैट में संदेश खोजें।' },
        { icon: '🔒', title: 'गोपनीयता पहले', desc: 'आपकी बातचीत एन्क्रिप्टेड और सुरक्षित है। हम आपके संदेश स्थायी रूप से संग्रहीत नहीं करते।' },
      ],
    },
    how: {
      title: 'Chorus कैसे काम करता है',
      subtitle: 'मिनटों में चैट शुरू करें, कोई भाषा बाधा नहीं',
      steps: [
        { num: '1', title: 'मुफ्त साइन अप', desc: 'अपना खाता बनाएं और अपनी मातृभाषा और सीखने वाली भाषाएं चुनें।' },
        { num: '2', title: 'चैट शुरू करें', desc: 'दोस्त खोजें या समूहों में शामिल हों। अपनी भाषा में संदेश लिखें—वे स्वचालित रूप से अनुवादित होंगे।' },
        { num: '3', title: 'सीखें और बढ़ें', desc: 'शब्दावली सहेजें, व्याकरण सुझाव देखें, और स्वाभाविक रूप से अपनी भाषा कौशल सुधारें।' },
      ],
    },
    languages: { title: 'समर्थित भाषाएं', subtitle: '10 प्रमुख भाषाओं के लोगों से जुड़ें' },
    cta: { title: 'भाषा की बाधाएं तोड़ने के लिए तैयार?', subtitle: 'आज ही Chorus से जुड़ें और दुनिया भर के लोगों से जुड़ना शुरू करें', button: 'अभी शुरू करें' },
    footer: {
      tagline: 'रीयल-टाइम अनुवाद के जरिए भाषा की बाधाएं तोड़ें और दुनिया भर के लोगों से जुड़ें।',
      product: 'उत्पाद',
      productLinks: [{ label: 'विशेषताएं', href: '#features' }, { label: 'वेब ऐप', href: '/login' }],
      company: 'कंपनी',
      companyLinks: [{ label: 'यह कैसे काम करता है', href: '#how-it-works' }, { label: 'भाषाएं', href: '#languages' }],
      support: 'सहायता',
      supportLinks: [{ label: 'API स्थिति', href: 'http://localhost:8080/health' }],
      rights: '© 2026 Chorus. सर्वाधिकार सुरक्षित।',
    },
  },

  es: {
    nav: { features: 'Características', how: 'Cómo Funciona', languages: 'Idiomas', launch: 'Abrir App' },
    hero: {
      badge: 'Disponible en {count} idiomas',
      titleA: 'Rompe Barreras Lingüísticas',
      titleB: 'Conecta Globalmente',
      subtitle: 'Mensajería en tiempo real con traducción instantánea en 10 idiomas principales. Chatea naturalmente en tu idioma mientras otros leen en el suyo.',
      cta: 'Comienza Gratis',
      seeHow: 'Ver Cómo Funciona',
      stats: { languages: 'Idiomas', translation: 'Tiempo Real', free: 'Gratis' },
      chat: {
        bubble1: '¡Hola! ¿Cómo estás?', bubble1Trans: 'Hello! How are you?',
        bubble2: '¡Estoy genial! Aprendiendo español', bubble2Trans: "I'm great! Learning Spanish",
        bubble3: 'Fantástico! 🎉', bubble3Trans: 'Fantastic! 🎉',
      },
    },
    features: {
      title: 'Funciones Potentes para la Comunicación Global',
      subtitle: 'Todo lo que necesitas para conectar con personas en todo el mundo',
      items: [
        { icon: '🌐', title: 'Traducción Instantánea', desc: 'Los mensajes se traducen automáticamente a tu idioma en tiempo real. Sin demoras, sin selección manual.' },
        { icon: '✏️', title: 'Análisis de Gramática', desc: 'Revisión gramatical con IA y evaluación de dificultad CEFR para que aprendas mientras chateas.' },
        { icon: '📚', title: 'Constructor de Vocabulario', desc: 'Sistema inteligente de repetición espaciada para recordar palabras y frases de tus conversaciones.' },
        { icon: '👥', title: 'Chats de Grupo', desc: 'Crea conversaciones multilingües con hasta 100 participantes, cada uno lee en su idioma.' },
        { icon: '🔍', title: 'Búsqueda Inteligente', desc: 'Encuentra mensajes en todos tus chats con búsqueda de texto completo en varios idiomas.' },
        { icon: '🔒', title: 'Privacidad Primero', desc: 'Tus conversaciones están encriptadas y seguras. No guardamos tus mensajes permanentemente.' },
      ],
    },
    how: {
      title: 'Cómo Funciona Chorus',
      subtitle: 'Empieza a chatear en minutos, sin barreras de idioma',
      steps: [
        { num: '1', title: 'Regístrate Gratis', desc: 'Crea tu cuenta y selecciona tu idioma nativo y los idiomas que quieres aprender.' },
        { num: '2', title: 'Empieza a Chatear', desc: 'Encuentra amigos o únete a grupos. Escribe en tu idioma: se traducirán automáticamente.' },
        { num: '3', title: 'Aprende y Crece', desc: 'Guarda vocabulario, revisa sugerencias de gramática y mejora tus habilidades naturalmente.' },
      ],
    },
    languages: { title: 'Idiomas Soportados', subtitle: 'Conecta con personas en 10 idiomas principales' },
    cta: { title: '¿Listo para Romper Barreras Lingüísticas?', subtitle: 'Únete a Chorus hoy y empieza a conectar con personas en todo el mundo', button: 'Comienza Ahora' },
    footer: {
      tagline: 'Rompe las barreras del idioma y conecta con personas de todo el mundo mediante traducción en tiempo real.',
      product: 'Producto',
      productLinks: [{ label: 'Características', href: '#features' }, { label: 'Web App', href: '/login' }],
      company: 'Empresa',
      companyLinks: [{ label: 'Cómo Funciona', href: '#how-it-works' }, { label: 'Idiomas', href: '#languages' }],
      support: 'Soporte',
      supportLinks: [{ label: 'Estado de API', href: 'http://localhost:8080/health' }],
      rights: '© 2026 Chorus. Todos los derechos reservados.',
    },
  },

  ar: {
    nav: { features: 'الميزات', how: 'كيف يعمل', languages: 'اللغات', launch: 'تشغيل التطبيق' },
    hero: {
      badge: 'متوفر بـ {count} لغات',
      titleA: 'اكسر حواجز اللغة',
      titleB: 'تواصل عالميًا',
      subtitle: 'رسائل فورية مع ترجمة فورية بـ10 لغات رئيسية. تحدث بلغتك بشكل طبيعي بينما يقرأ الآخرون بلغتهم.',
      cta: 'ابدأ مجانًا',
      seeHow: 'شاهد كيف يعمل',
      stats: { languages: 'لغات', translation: 'فوري', free: 'مجاني' },
      chat: {
        bubble1: '¡Hola! ¿Cómo estás?', bubble1Trans: 'مرحبًا! كيف حالك؟',
        bubble2: 'أنا بخير! أتعلم الإسبانية', bubble2Trans: '¡Estoy genial! Aprendiendo español',
        bubble3: 'رائع! 🎉', bubble3Trans: 'Fantástico! 🎉',
      },
    },
    features: {
      title: 'ميزات قوية للتواصل العالمي',
      subtitle: 'كل ما تحتاجه للتواصل مع الناس حول العالم',
      items: [
        { icon: '🌐', title: 'ترجمة فورية', desc: 'تتم ترجمة الرسائل تلقائيًا إلى لغتك المفضلة في الوقت الفعلي. لا تأخير، لا اختيار يدوي.' },
        { icon: '✏️', title: 'تحليل القواعد', desc: 'فحص نحوي بالذكاء الاصطناعي وتقييم صعوبة CEFR يساعدك على التعلم أثناء المحادثة.' },
        { icon: '📚', title: 'بناء المفردات', desc: 'نظام تكرار ذكي يساعدك على تذكر الكلمات والعبارات الجديدة من محادثاتك.' },
        { icon: '👥', title: 'محادثات جماعية', desc: 'أنشئ محادثات جماعية متعددة اللغات حتى 100 مشارك، كل منهم يقرأ بلغته.' },
        { icon: '🔍', title: 'بحث ذكي', desc: 'ابحث عن الرسائل عبر جميع محادثاتك ببحث نصي كامل يعمل بعدة لغات.' },
        { icon: '🔒', title: 'الخصوصية أولاً', desc: 'محادثاتك مشفرة وآمنة. لا نخزن رسائلك بشكل دائم.' },
      ],
    },
    how: {
      title: 'كيف يعمل Chorus',
      subtitle: 'ابدأ المحادثة في دقائق، دون حواجز لغوية',
      steps: [
        { num: '1', title: 'سجل مجانًا', desc: 'أنشئ حسابك واختر لغتك الأم واللغات التي تريد تعلمها.' },
        { num: '2', title: 'ابدأ المحادثة', desc: 'ابحث عن أصدقاء أو انضم إلى مجموعات. اكتب بلغتك وستُترجم تلقائيًا.' },
        { num: '3', title: 'تعلم ونمُ', desc: 'احفظ المفردات، راجع اقتراحات القواعد، وحسّن مهاراتك اللغوية بشكل طبيعي.' },
      ],
    },
    languages: { title: 'اللغات المدعومة', subtitle: 'تواصل مع أشخاص بـ10 لغات رئيسية' },
    cta: { title: 'مستعد لكسر حواجز اللغة؟', subtitle: 'انضم إلى Chorus اليوم وابدأ التواصل مع الناس حول العالم', button: 'ابدأ الآن' },
    footer: {
      tagline: 'اكسر حواجز اللغة وتواصل مع الناس حول العالم عبر الترجمة الفورية.',
      product: 'المنتج',
      productLinks: [{ label: 'الميزات', href: '#features' }, { label: 'تطبيق الويب', href: '/login' }],
      company: 'الشركة',
      companyLinks: [{ label: 'كيف يعمل', href: '#how-it-works' }, { label: 'اللغات', href: '#languages' }],
      support: 'الدعم',
      supportLinks: [{ label: 'حالة API', href: 'http://localhost:8080/health' }],
      rights: '© 2026 Chorus. جميع الحقوق محفوظة.',
    },
  },

  fr: {
    nav: { features: 'Fonctionnalités', how: 'Comment ça Marche', languages: 'Langues', launch: 'Lancer l\'App' },
    hero: {
      badge: 'Disponible en {count} langues',
      titleA: 'Brisez les Barrières Linguistiques',
      titleB: 'Connectez-vous Globalement',
      subtitle: 'Messagerie en temps réel avec traduction instantanée dans 10 langues majeures. Discutez naturellement dans votre langue pendant que les autres lisent dans la leur.',
      cta: 'Commencer Gratuitement',
      seeHow: 'Voir Comment ça Marche',
      stats: { languages: 'Langues', translation: 'Temps Réel', free: 'Gratuit' },
      chat: {
        bubble1: '¡Hola! ¿Cómo estás?', bubble1Trans: 'Bonjour ! Comment vas-tu ?',
        bubble2: "Je vais très bien ! J'apprends l'espagnol", bubble2Trans: '¡Estoy genial! Aprendiendo español',
        bubble3: 'Fantastique ! 🎉', bubble3Trans: 'Fantástico! 🎉',
      },
    },
    features: {
      title: 'Des Fonctions Puissantes pour une Communication Globale',
      subtitle: 'Tout ce dont vous avez besoin pour vous connecter avec le monde entier',
      items: [
        { icon: '🌐', title: 'Traduction Instantanée', desc: 'Messages traduits automatiquement en temps réel. Aucun délai, aucune sélection manuelle.' },
        { icon: '✏️', title: 'Analyse Grammaticale', desc: 'Vérification grammaticale par IA et évaluation de difficulté CEFR pour apprendre en discutant.' },
        { icon: '📚', title: 'Constructeur de Vocabulaire', desc: 'Système intelligent de répétition espacée pour mémoriser les mots de vos conversations.' },
        { icon: '👥', title: 'Discussions de Groupe', desc: 'Créez des conversations multilingues jusqu\'à 100 participants, chacun lit dans sa langue.' },
        { icon: '🔍', title: 'Recherche Intelligente', desc: 'Trouvez des messages grâce à une recherche plein texte multilingue.' },
        { icon: '🔒', title: 'Confidentialité d\'abord', desc: 'Vos conversations sont chiffrées et sécurisées. Nous ne stockons pas vos messages.' },
      ],
    },
    how: {
      title: 'Comment Chorus Fonctionne',
      subtitle: 'Commencez à discuter en quelques minutes, sans barrières linguistiques',
      steps: [
        { num: '1', title: 'Inscrivez-vous Gratuitement', desc: 'Créez votre compte et choisissez votre langue maternelle et celles à apprendre.' },
        { num: '2', title: 'Commencez à Discuter', desc: 'Trouvez des amis ou rejoignez des groupes. Écrivez dans votre langue : tout est traduit.' },
        { num: '3', title: 'Apprenez et Progressez', desc: 'Enregistrez du vocabulaire, révisez les suggestions et améliorez vos compétences.' },
      ],
    },
    languages: { title: 'Langues Prises en Charge', subtitle: 'Connectez-vous avec des gens dans 10 langues majeures' },
    cta: { title: 'Prêt à Briser les Barrières Linguistiques ?', subtitle: 'Rejoignez Chorus aujourd\'hui et connectez-vous avec le monde entier', button: 'Commencer Maintenant' },
    footer: {
      tagline: 'Brisez les barrières linguistiques et connectez-vous grâce à la traduction en temps réel.',
      product: 'Produit',
      productLinks: [{ label: 'Fonctionnalités', href: '#features' }, { label: 'Application Web', href: '/login' }],
      company: 'Entreprise',
      companyLinks: [{ label: 'Comment ça Marche', href: '#how-it-works' }, { label: 'Langues', href: '#languages' }],
      support: 'Support',
      supportLinks: [{ label: 'Statut API', href: 'http://localhost:8080/health' }],
      rights: '© 2026 Chorus. Tous droits réservés.',
    },
  },

  bn: {
    nav: { features: 'বৈশিষ্ট্য', how: 'কীভাবে কাজ করে', languages: 'ভাষা', launch: 'অ্যাপ চালু করুন' },
    hero: {
      badge: '{count}টি ভাষায় উপলব্ধ',
      titleA: 'ভাষার বাধা ভাঙুন',
      titleB: 'বিশ্বব্যাপী সংযুক্ত হোন',
      subtitle: '১০টি প্রধান ভাষায় তাৎক্ষণিক অনুবাদসহ রিয়েল-টাইম মেসেজিং। আপনার ভাষায় স্বাভাবিকভাবে চ্যাট করুন, অন্যজন নিজের ভাষায় পড়বেন।',
      cta: 'বিনামূল্যে শুরু করুন',
      seeHow: 'কীভাবে কাজ করে দেখুন',
      stats: { languages: 'ভাষা', translation: 'রিয়েল-টাইম', free: 'বিনামূল্যে' },
      chat: {
        bubble1: '¡Hola! ¿Cómo estás?', bubble1Trans: 'হ্যালো! কেমন আছেন?',
        bubble2: 'আমি দারুণ! স্প্যানিশ শিখছি', bubble2Trans: '¡Estoy genial! Aprendiendo español',
        bubble3: 'দারুণ! 🎉', bubble3Trans: 'Fantástico! 🎉',
      },
    },
    features: {
      title: 'বিশ্বব্যাপী যোগাযোগের জন্য শক্তিশালী বৈশিষ্ট্য',
      subtitle: 'বিশ্বজুড়ে মানুষের সাথে সংযোগের জন্য যা যা প্রয়োজন',
      items: [
        { icon: '🌐', title: 'তাৎক্ষণিক অনুবাদ', desc: 'বার্তা স্বয়ংক্রিয়ভাবে আপনার পছন্দের ভাষায় রিয়েল-টাইমে অনুবাদিত হয়। কোনো বিলম্ব নেই।' },
        { icon: '✏️', title: 'ব্যাকরণ বিশ্লেষণ', desc: 'AI-চালিত ব্যাকরণ পরীক্ষা ও CEFR স্তর নির্ধারণ আপনাকে চ্যাটের সময় শিখতে সাহায্য করে।' },
        { icon: '📚', title: 'শব্দভান্ডার নির্মাতা', desc: 'স্মার্ট স্পেসড রিপিটিশন সিস্টেম কথা থেকে নতুন শব্দ মনে রাখতে সাহায্য করে।' },
        { icon: '👥', title: 'গ্রুপ চ্যাট', desc: '১০০ জন পর্যন্ত বহুভাষিক গ্রুপ তৈরি করুন, প্রত্যেকে নিজের ভাষায় পড়বে।' },
        { icon: '🔍', title: 'স্মার্ট অনুসন্ধান', desc: 'বহু ভাষায় কাজ করা ফুল-টেক্সট অনুসন্ধানে সব চ্যাটে বার্তা খুঁজুন।' },
        { icon: '🔒', title: 'গোপনীয়তা প্রথম', desc: 'আপনার কথোপকথন এনক্রিপ্টেড ও নিরাপদ। আমরা বার্তা স্থায়ীভাবে সংরক্ষণ করি না।' },
      ],
    },
    how: {
      title: 'Chorus কীভাবে কাজ করে',
      subtitle: 'মিনিটেই চ্যাট শুরু করুন, কোনো ভাষার বাধা নেই',
      steps: [
        { num: '১', title: 'বিনামূল্যে সাইন আপ', desc: 'অ্যাকাউন্ট খুলুন এবং মাতৃভাষা ও শেখার ভাষা নির্বাচন করুন।' },
        { num: '২', title: 'চ্যাট শুরু করুন', desc: 'বন্ধু খুঁজুন বা গ্রুপে যোগ দিন। আপনার ভাষায় লিখুন—স্বয়ংক্রিয় অনুবাদ হবে।' },
        { num: '৩', title: 'শিখুন ও বাড়ুন', desc: 'শব্দভান্ডার সংরক্ষণ, ব্যাকরণ পরামর্শ ও ভাষা দক্ষতা উন্নত করুন।' },
      ],
    },
    languages: { title: 'সমর্থিত ভাষা', subtitle: '১০টি প্রধান ভাষার মানুষের সাথে সংযোগ করুন' },
    cta: { title: 'ভাষার বাধা ভাঙতে প্রস্তুত?', subtitle: 'আজই Chorus-এ যোগ দিন এবং বিশ্বজুড়ে মানুষের সাথে যুক্ত হোন', button: 'এখনই শুরু করুন' },
    footer: {
      tagline: 'রিয়েল-টাইম অনুবাদের মাধ্যমে ভাষার বাধা ভেঙে বিশ্বজুড়ে সংযোগ করুন।',
      product: 'পণ্য',
      productLinks: [{ label: 'বৈশিষ্ট্য', href: '#features' }, { label: 'ওয়েব অ্যাপ', href: '/login' }],
      company: 'কোম্পানি',
      companyLinks: [{ label: 'কীভাবে কাজ করে', href: '#how-it-works' }, { label: 'ভাষা', href: '#languages' }],
      support: 'সহায়তা',
      supportLinks: [{ label: 'API অবস্থা', href: 'http://localhost:8080/health' }],
      rights: '© ২০২৬ Chorus. সর্বস্বত্ব সংরক্ষিত।',
    },
  },

  pt: {
    nav: { features: 'Recursos', how: 'Como Funciona', languages: 'Idiomas', launch: 'Abrir App' },
    hero: {
      badge: 'Disponível em {count} idiomas',
      titleA: 'Quebre Barreiras Linguísticas',
      titleB: 'Conecte-se Globalmente',
      subtitle: 'Mensagens em tempo real com tradução instantânea em 10 idiomas principais. Converse naturalmente no seu idioma enquanto outros leem no deles.',
      cta: 'Comece Grátis',
      seeHow: 'Veja Como Funciona',
      stats: { languages: 'Idiomas', translation: 'Tempo Real', free: 'Gratuito' },
      chat: {
        bubble1: '¡Hola! ¿Cómo estás?', bubble1Trans: 'Olá! Como você está?',
        bubble2: 'Estou ótimo! Aprendendo espanhol', bubble2Trans: '¡Estoy genial! Aprendiendo español',
        bubble3: 'Fantástico! 🎉', bubble3Trans: 'Fantástico! 🎉',
      },
    },
    features: {
      title: 'Recursos Poderosos para Comunicação Global',
      subtitle: 'Tudo que você precisa para se conectar com pessoas no mundo todo',
      items: [
        { icon: '🌐', title: 'Tradução Instantânea', desc: 'Mensagens traduzidas automaticamente em tempo real. Sem atrasos, sem seleção manual.' },
        { icon: '✏️', title: 'Análise Gramatical', desc: 'Verificação gramatical com IA e avaliação de dificuldade CEFR enquanto você conversa.' },
        { icon: '📚', title: 'Construtor de Vocabulário', desc: 'Sistema inteligente de repetição espaçada para lembrar palavras das suas conversas.' },
        { icon: '👥', title: 'Bate-papos em Grupo', desc: 'Crie conversas multilíngues com até 100 participantes, cada um lê no seu idioma.' },
        { icon: '🔍', title: 'Busca Inteligente', desc: 'Encontre mensagens com busca de texto completo que funciona em vários idiomas.' },
        { icon: '🔒', title: 'Privacidade Primeiro', desc: 'Suas conversas são criptografadas e seguras. Não armazenamos suas mensagens permanentemente.' },
      ],
    },
    how: {
      title: 'Como o Chorus Funciona',
      subtitle: 'Comece a conversar em minutos, sem barreiras de idioma',
      steps: [
        { num: '1', title: 'Cadastre-se Grátis', desc: 'Crie sua conta e selecione seu idioma nativo e os idiomas que deseja aprender.' },
        { num: '2', title: 'Comece a Conversar', desc: 'Encontre amigos ou entre em grupos. Digite no seu idioma—eles serão traduzidos.' },
        { num: '3', title: 'Aprenda e Cresça', desc: 'Salve vocabulário, revise sugestões de gramática e melhore suas habilidades.' },
      ],
    },
    languages: { title: 'Idiomas Suportados', subtitle: 'Conecte-se com pessoas em 10 idiomas principais' },
    cta: { title: 'Pronto para Quebrar Barreiras Linguísticas?', subtitle: 'Entre no Chorus hoje e comece a se conectar com pessoas no mundo todo', button: 'Comece Agora' },
    footer: {
      tagline: 'Quebre barreiras de idioma e conecte-se com pessoas do mundo todo através da tradução em tempo real.',
      product: 'Produto',
      productLinks: [{ label: 'Recursos', href: '#features' }, { label: 'Web App', href: '/login' }],
      company: 'Empresa',
      companyLinks: [{ label: 'Como Funciona', href: '#how-it-works' }, { label: 'Idiomas', href: '#languages' }],
      support: 'Suporte',
      supportLinks: [{ label: 'Status da API', href: 'http://localhost:8080/health' }],
      rights: '© 2026 Chorus. Todos os direitos reservados.',
    },
  },

  ru: {
    nav: { features: 'Возможности', how: 'Как это работает', languages: 'Языки', launch: 'Открыть приложение' },
    hero: {
      badge: 'Доступно на {count} языках',
      titleA: 'Сломайте языковые барьеры',
      titleB: 'Общайтесь глобально',
      subtitle: 'Обмен сообщениями в реальном времени с мгновенным переводом на 10 основных языков. Общайтесь на своём языке, пока другие читают на своём.',
      cta: 'Начать бесплатно',
      seeHow: 'Как это работает',
      stats: { languages: 'Языков', translation: 'В реальном времени', free: 'Бесплатно' },
      chat: {
        bubble1: '¡Hola! ¿Cómo estás?', bubble1Trans: 'Привет! Как дела?',
        bubble2: 'Отлично! Учу испанский', bubble2Trans: '¡Estoy genial! Aprendiendo español',
        bubble3: 'Отлично! 🎉', bubble3Trans: 'Fantástico! 🎉',
      },
    },
    features: {
      title: 'Мощные функции для глобального общения',
      subtitle: 'Всё, что нужно для связи с людьми по всему миру',
      items: [
        { icon: '🌐', title: 'Мгновенный перевод', desc: 'Сообщения автоматически переводятся на ваш язык в реальном времени. Без задержек.' },
        { icon: '✏️', title: 'Грамматический анализ', desc: 'Проверка грамматики с ИИ и оценка сложности CEFR помогает учиться в чате.' },
        { icon: '📚', title: 'Словарь', desc: 'Умная система интервального повторения помогает запоминать слова из разговоров.' },
        { icon: '👥', title: 'Групповые чаты', desc: 'Создавайте многоязычные группы до 100 участников, каждый читает на своём языке.' },
        { icon: '🔍', title: 'Умный поиск', desc: 'Находите сообщения полнотекстовым поиском, работающим на нескольких языках.' },
        { icon: '🔒', title: 'Конфиденциальность', desc: 'Ваши разговоры зашифрованы. Мы не храним сообщения постоянно.' },
      ],
    },
    how: {
      title: 'Как работает Chorus',
      subtitle: 'Начните общаться за минуты, без языковых барьеров',
      steps: [
        { num: '1', title: 'Регистрация бесплатно', desc: 'Создайте аккаунт и выберите родной язык и языки для изучения.' },
        { num: '2', title: 'Начните общаться', desc: 'Находите друзей или присоединяйтесь к группам. Пишите на своём языке — перевод автоматический.' },
        { num: '3', title: 'Учитесь и развивайтесь', desc: 'Сохраняйте слова, просматривайте советы по грамматике и улучшайте навыки.' },
      ],
    },
    languages: { title: 'Поддерживаемые языки', subtitle: 'Общайтесь с людьми на 10 основных языках' },
    cta: { title: 'Готовы сломать языковые барьеры?', subtitle: 'Присоединяйтесь к Chorus сегодня и общайтесь с людьми по всему миру', button: 'Начать сейчас' },
    footer: {
      tagline: 'Сломайте языковые барьеры и общайтесь с людьми по всему миру благодаря переводу в реальном времени.',
      product: 'Продукт',
      productLinks: [{ label: 'Возможности', href: '#features' }, { label: 'Веб-приложение', href: '/login' }],
      company: 'Компания',
      companyLinks: [{ label: 'Как это работает', href: '#how-it-works' }, { label: 'Языки', href: '#languages' }],
      support: 'Поддержка',
      supportLinks: [{ label: 'Статус API', href: 'http://localhost:8080/health' }],
      rights: '© 2026 Chorus. Все права защищены.',
    },
  },

  ur: {
    nav: { features: 'خصوصیات', how: 'یہ کیسے کام کرتا ہے', languages: 'زبانیں', launch: 'ایپ شروع کریں' },
    hero: {
      badge: '{count} زبانوں میں دستیاب',
      titleA: 'زبان کی رکاوٹیں توڑیں',
      titleB: 'عالمی سطح پر جڑیں',
      subtitle: '10 اہم زبانوں میں فوری ترجمے کے ساتھ حقیقی وقت کی پیغام رسانی۔ اپنی زبان میں قدرتی طور پر گفتگو کریں جبکہ دوسرے اپنی زبان میں پڑھیں۔',
      cta: 'مفت شروع کریں',
      seeHow: 'دیکھیں یہ کیسے کام کرتا ہے',
      stats: { languages: 'زبانیں', translation: 'حقیقی وقت', free: 'مفت' },
      chat: {
        bubble1: '¡Hola! ¿Cómo estás?', bubble1Trans: 'ہیلو! آپ کیسے ہیں؟',
        bubble2: 'میں بہترین ہوں! ہسپانوی سیکھ رہا ہوں', bubble2Trans: '¡Estoy genial! Aprendiendo español',
        bubble3: 'زبردست! 🎉', bubble3Trans: 'Fantástico! 🎉',
      },
    },
    features: {
      title: 'عالمی رابطے کے لیے طاقتور خصوصیات',
      subtitle: 'دنیا بھر کے لوگوں سے جڑنے کے لیے ہر وہ چیز جو آپ کو چاہیے',
      items: [
        { icon: '🌐', title: 'فوری ترجمہ', desc: 'پیغامات خود بخود آپ کی پسندیدہ زبان میں حقیقی وقت میں ترجمہ ہوتے ہیں۔ کوئی تاخیر نہیں۔' },
        { icon: '✏️', title: 'گرامر تجزیہ', desc: 'AI سے چلنے والی گرامر جانچ اور CEFR مشکل کی تشخیص گفتگو کے دوران سیکھنے میں مدد کرتی ہے۔' },
        { icon: '📚', title: 'ذخیرہ الفاظ', desc: 'سمارٹ اسپیسڈ ریپیٹیشن سسٹم گفتگو سے نئے الفاظ یاد رکھنے میں مدد کرتا ہے۔' },
        { icon: '👥', title: 'گروپ چیٹس', desc: '100 شرکاء تک کثیر لسانی گروپ بنائیں، ہر کوئی اپنی زبان میں پڑھے۔' },
        { icon: '🔍', title: 'سمارٹ تلاش', desc: 'کئی زبانوں میں کام کرنے والی مکمل متن تلاش سے تمام چیٹس میں پیغامات تلاش کریں۔' },
        { icon: '🔒', title: 'پرائیویسی پہلے', desc: 'آپ کی گفتگو خفیہ اور محفوظ ہے۔ ہم پیغامات مستقل طور پر محفوظ نہیں کرتے۔' },
      ],
    },
    how: {
      title: 'Chorus کیسے کام کرتا ہے',
      subtitle: 'منٹوں میں چیٹ شروع کریں، کوئی زبان کی رکاوٹ نہیں',
      steps: [
        { num: '۱', title: 'مفت سائن اپ', desc: 'اپنا اکاؤنٹ بنائیں اور اپنی مادری زبان اور سیکھنے کی زبانیں منتخب کریں۔' },
        { num: '۲', title: 'چیٹ شروع کریں', desc: 'دوست تلاش کریں یا گروپس میں شامل ہوں۔ اپنی زبان میں لکھیں — خودکار ترجمہ ہوگا۔' },
        { num: '۳', title: 'سیکھیں اور بڑھیں', desc: 'الفاظ محفوظ کریں، گرامر تجاویز دیکھیں، اور زبان کی مہارت بہتر بنائیں۔' },
      ],
    },
    languages: { title: 'معاون زبانیں', subtitle: '10 اہم زبانوں کے لوگوں سے جڑیں' },
    cta: { title: 'زبان کی رکاوٹیں توڑنے کے لیے تیار؟', subtitle: 'آج ہی Chorus میں شامل ہوں اور دنیا بھر کے لوگوں سے جڑنا شروع کریں', button: 'ابھی شروع کریں' },
    footer: {
      tagline: 'حقیقی وقت کے ترجمے کے ذریعے زبان کی رکاوٹیں توڑیں اور دنیا بھر کے لوگوں سے جڑیں۔',
      product: 'پروڈکٹ',
      productLinks: [{ label: 'خصوصیات', href: '#features' }, { label: 'ویب ایپ', href: '/login' }],
      company: 'کمپنی',
      companyLinks: [{ label: 'یہ کیسے کام کرتا ہے', href: '#how-it-works' }, { label: 'زبانیں', href: '#languages' }],
      support: 'سپورٹ',
      supportLinks: [{ label: 'API حیثیت', href: 'http://localhost:8080/health' }],
      rights: '© 2026 Chorus. جملہ حقوق محفوظ ہیں۔',
    },
  },
}

// Keep legacy HERO_TRANSLATIONS export for any other consumers.
export const HERO_TRANSLATIONS: Record<string, { title: string; subtitle: string; cta: string }> = Object.fromEntries(
  TOP10.map(code => [code, {
    title: `${STRINGS[code].hero.titleA}, ${STRINGS[code].hero.titleB}`,
    subtitle: STRINGS[code].hero.subtitle,
    cta: STRINGS[code].hero.cta,
  }])
)

export default function Landing() {
  const [selectedLang, setSelectedLang] = useState(() => localStorage.getItem('preferredLanguage') || detectBrowserLanguage())
  const lang = STRINGS[selectedLang] ? selectedLang : 'en'
  const t = STRINGS[lang]
  const nativeName = getNativeLanguageName(lang)
  const supportedCount = TOP10.length

  const handleLanguageChange = (code: string) => {
    setSelectedLang(code)
    localStorage.setItem('preferredLanguage', code)
  }

  return (
    <div className="min-h-screen bg-gradient-to-b from-white to-gray-50" lang={lang}>
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
          <div className="flex items-center gap-4">
            <ul className="hidden md:flex gap-8 items-center">
              <li><a href="#features" className="text-gray-700 hover:text-indigo-600 transition">{t.nav.features}</a></li>
              <li><a href="#how-it-works" className="text-gray-700 hover:text-indigo-600 transition">{t.nav.how}</a></li>
              <li><a href="#languages" className="text-gray-700 hover:text-indigo-600 transition">{t.nav.languages}</a></li>
            </ul>
            <LanguageSelector
              currentLang={lang}
              onLanguageChange={handleLanguageChange}
              variant="navbar"
            />
            <Link to="/login" className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition whitespace-nowrap">{t.nav.launch}</Link>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="pt-32 pb-20 px-6">
        <div className="max-w-6xl mx-auto">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-12 items-center">
            <div>
              <div className="inline-block bg-indigo-100 text-indigo-700 text-sm font-semibold px-3 py-1 rounded-full mb-4">
                🌍 {nativeName} · {t.hero.badge.replace('{count}', String(supportedCount))}
              </div>
              <h1 className="text-5xl md:text-6xl font-bold mb-6">
                {t.hero.titleA},<br />
                <span className="bg-gradient-to-r from-indigo-600 to-purple-600 bg-clip-text text-transparent">{t.hero.titleB}</span>
              </h1>
              <p className="text-xl text-gray-600 mb-8">
                {t.hero.subtitle}
              </p>
              <div className="flex gap-4 mb-12 flex-wrap">
                <Link to="/waitlist" className="px-8 py-4 bg-indigo-600 text-white rounded-lg font-semibold hover:bg-indigo-700 transition text-lg">
                  Join the waitlist
                </Link>
                <a href="#how-it-works" className="px-8 py-4 border-2 border-indigo-600 text-indigo-600 rounded-lg font-semibold hover:bg-indigo-50 transition text-lg">
                  {t.hero.seeHow}
                </a>
              </div>
              <div className="flex gap-8">
                <div>
                  <div className="text-3xl font-bold text-indigo-600">{supportedCount}</div>
                  <p className="text-gray-600">{t.hero.stats.languages}</p>
                </div>
                <div>
                  <div className="text-3xl font-bold text-indigo-600">{t.hero.stats.translation}</div>
                  <p className="text-gray-600">{t.features.items[0].title}</p>
                </div>
                <div>
                  <div className="text-3xl font-bold text-indigo-600">100%</div>
                  <p className="text-gray-600">{t.hero.stats.free}</p>
                </div>
              </div>
            </div>
            <div className="relative">
              <div className="bg-gradient-to-br from-indigo-600 to-purple-600 rounded-3xl p-1 shadow-2xl">
                <div className="bg-white rounded-3xl p-6">
                  <div className="space-y-4">
                    <div className="bg-gray-100 rounded-lg p-4">
                      <p className="text-sm text-gray-600 mb-1">{t.hero.chat.bubble1}</p>
                      <p className="text-gray-400 text-xs">{t.hero.chat.bubble1Trans}</p>
                    </div>
                    <div className="bg-indigo-600 rounded-lg p-4 ml-8">
                      <p className="text-sm text-white mb-1">{t.hero.chat.bubble2}</p>
                      <p className="text-indigo-200 text-xs">{t.hero.chat.bubble2Trans}</p>
                    </div>
                    <div className="bg-gray-100 rounded-lg p-4">
                      <p className="text-sm text-gray-600">{t.hero.chat.bubble3}</p>
                      <p className="text-gray-400 text-xs">{t.hero.chat.bubble3Trans}</p>
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
            <h2 className="text-4xl md:text-5xl font-bold mb-4">{t.features.title}</h2>
            <p className="text-xl text-gray-600">{t.features.subtitle}</p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
            {t.features.items.map((feature, i) => (
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
            <h2 className="text-4xl md:text-5xl font-bold mb-4">{t.how.title}</h2>
            <p className="text-xl text-gray-600">{t.how.subtitle}</p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            {t.how.steps.map((step, i) => (
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
            <h2 className="text-4xl md:text-5xl font-bold mb-4">{t.languages.title}</h2>
            <p className="text-xl text-gray-600">{t.languages.subtitle}</p>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-10 gap-4">
            {TOP10.map((code, i) => {
              const langInfo = SUPPORTED_LANGUAGES.find(l => l.code === code) || SUPPORTED_LANGUAGES[0]
              return (
                <div key={i} className="bg-white p-6 rounded-xl text-center shadow hover:shadow-lg transition">
                  <div className="text-4xl mb-2">{langInfo.flag}</div>
                  <p className="font-semibold text-gray-800">{langInfo.name}</p>
                  <p className="text-xs text-gray-400 mt-1">{langInfo.nativeName}</p>
                </div>
              )
            })}
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20 px-6 bg-gradient-to-r from-indigo-600 to-purple-600">
        <div className="max-w-4xl mx-auto text-center text-white">
          <h2 className="text-4xl md:text-5xl font-bold mb-4">{t.cta.title}</h2>
          <p className="text-xl mb-8 opacity-90">{t.cta.subtitle}</p>
          <Link to="/waitlist" className="px-8 py-4 bg-white text-indigo-600 rounded-lg font-bold text-lg hover:bg-gray-100 transition inline-block">
            Join the waitlist
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
              <p className="text-gray-400">{t.footer.tagline}</p>
            </div>
            <div>
              <h4 className="font-bold mb-4">{t.footer.product}</h4>
              <ul className="space-y-2 text-gray-400">
                {t.footer.productLinks.map((l, i) => (
                  <li key={i}><a href={l.href} className="hover:text-white">{l.label}</a></li>
                ))}
              </ul>
            </div>
            <div>
              <h4 className="font-bold mb-4">{t.footer.company}</h4>
              <ul className="space-y-2 text-gray-400">
                {t.footer.companyLinks.map((l, i) => (
                  <li key={i}><a href={l.href} className="hover:text-white">{l.label}</a></li>
                ))}
              </ul>
            </div>
            <div>
              <h4 className="font-bold mb-4">{t.footer.support}</h4>
              <ul className="space-y-2 text-gray-400">
                {t.footer.supportLinks.map((l, i) => (
                  <li key={i}><a href={l.href} className="hover:text-white">{l.label}</a></li>
                ))}
              </ul>
            </div>
          </div>
          <div className="border-t border-gray-800 pt-8 text-center text-gray-400">
            <p>{t.footer.rights}</p>
          </div>
        </div>
      </footer>
    </div>
  )
}