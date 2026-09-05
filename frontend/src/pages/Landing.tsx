import { Link } from 'react-router-dom'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { detectBrowserLanguage, getNativeLanguageName, SUPPORTED_LANGUAGES } from '../services/language'
import LanguageSelector from '../components/LanguageSelector'
import { useStore } from '../store'
import { YEARLY_LIST_PRICE } from '@chorus/shared'

// =============================================================================
// Full-page internationalization for the landing site.
// Every visible string on the page is translated for the top-10 world
// languages (matches LibreTranslate LT_LOAD_ONLY). Unsupported languages fall
// back to English. Selecting a language in the navbar changes ALL sections —
// nav, hero, stats, features, how-it-works, languages, CTA and footer.
// =============================================================================
interface LandingStrings {
  nav: { features: string; how: string; languages: string; pricing: string; launch: string; login: string }
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
  pricing: {
    title: string
    subtitle: string
    free: { name: string; price: string; per: string; cta: string; features: string[] }
    premium: { name: string; price: string; per: string; promo: string; cta: string; features: string[] }
    enterprise: { name: string; desc: string; cta: string; features: string[] }
    note: string
  }
  cta: { title: string; subtitle: string; button: string }
  footer: { tagline: string; product: string; productLinks: { label: string; href: string }[]; company: string; companyLinks: { label: string; href: string }[]; support: string; supportLinks: { label: string; href: string }[]; rights: string }
}

// Ordered top-10 list for consistent display.
const TOP10 = ['en', 'zh', 'hi', 'es', 'ar', 'fr', 'bn', 'pt', 'ru', 'ur']

const STRINGS: Record<string, LandingStrings> = {
  en: {
    nav: { features: 'Features', how: 'How It Works', languages: 'Languages', pricing: 'Pricing', launch: 'Launch App', login: 'Log In' },
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
    pricing: {
      title: 'Simple Pricing',
      subtitle: 'Start free. Upgrade when you\'re ready for more.',
      free: {
        name: 'Free', price: '$0', per: 'forever',
        cta: 'Current Plan',
        features: ['Unlimited chats & groups', 'Live translation up to 200 characters', 'On-demand grammar & vocabulary tools', 'Search across all messages'],
      },
      premium: {
        name: 'Premium', price: '$79.90', per: 'year',
        promo: '2 months free',
        cta: 'Get Premium',
        features: ['Automatic grammar analysis', 'Faster AI responses', 'Messages longer than 200 characters', 'Higher daily quotas'],
      },
      enterprise: {
        name: 'Enterprise', desc: 'Self-hosted or custom deployment for teams.',
        cta: 'Contact Us',
        features: ['Self-hosting & custom domains', 'Dedicated support', 'Volume & SLA options'],
      },
      note: 'Free forever plan — no credit card required.',
    },
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
    nav: { features: '功能', how: '工作原理', languages: '语言', pricing: '定价', launch: '启动应用', login: '登录' },
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
    pricing: {
      title: '简单定价',
      subtitle: '免费开始。准备好后随时升级。',
      free: {
        name: '免费', price: '$0', per: '永久',
        cta: '当前套餐',
        features: ['无限聊天和群组', '实时翻译，最多 200 个字符', '按需使用语法和词汇工具', '跨所有消息搜索'],
      },
      premium: {
        name: '高级版', price: '$79.90', per: '年',
        promo: '免费送 2 个月',
        cta: '获取高级版',
        features: ['自动语法分析', '更快的 AI 响应', '超过 200 个字符的消息', '更高的每日限额'],
      },
      enterprise: {
        name: '企业版', desc: '面向团队的自托管或定制部署。',
        cta: '联系我们',
        features: ['自托管和自定义域名', '专属支持', '批量与 SLA 选项'],
      },
      note: '永久免费套餐 — 无需信用卡。',
    },
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
    nav: { features: 'विशेषताएं', how: 'यह कैसे काम करता है', languages: 'भाषाएं', pricing: 'मूल्य निर्धारण', launch: 'ऐप शुरू करें', login: 'लॉग इन' },
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
    pricing: {
      title: 'सरल मूल्य निर्धारण',
      subtitle: 'मुफ्त में शुरू करें। तैयार होने पर अपग्रेड करें।',
      free: {
        name: 'फ्री', price: '$0', per: 'हमेशा के लिए',
        cta: 'वर्तमान प्लान',
        features: ['असीमित चैट व समूह', '200 अक्षरों तक लाइव अनुवाद', 'मांग पर व्याकरण व शब्दावली टूल', 'सभी संदेशों में खोज'],
      },
      premium: {
        name: 'प्रीमियम', price: '$79.90', per: 'साल',
        promo: '2 महीने मुफ़्त',
        cta: 'प्रीमियम पाएं',
        features: ['स्वचालित व्याकरण विश्लेषण', 'तेज़ AI प्रतिक्रियाएँ', '200 अक्षरों से लंबे संदेश', 'अधिक दैनिक सीमाएँ'],
      },
      enterprise: {
        name: 'उद्यम', desc: 'टीमों के लिए स्व-होस्टेड या कस्टम तैनाती।',
        cta: 'संपर्क करें',
        features: ['स्व-होस्टिंग व कस्टम डोमेन', 'समर्पित सहायता', 'वॉल्यूम व SLA विकल्प'],
      },
      note: 'हमेशा के लिए मुफ्त प्लान — कोई क्रेडिट कार्ड नहीं चाहिए।',
    },
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
    nav: { features: 'Características', how: 'Cómo Funciona', languages: 'Idiomas', pricing: 'Precios', launch: 'Abrir App', login: 'Iniciar sesión' },
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
    pricing: {
      title: 'Precios Sencillos',
      subtitle: 'Empieza gratis. Mejora cuando estés listo.',
      free: {
        name: 'Gratis', price: '$0', per: 'para siempre',
        cta: 'Plan Actual',
        features: ['Chats y grupos ilimitados', 'Traducción en vivo hasta 200 caracteres', 'Herramientas de gramática y vocabulario bajo demanda', 'Búsqueda en todos los mensajes'],
      },
      premium: {
        name: 'Premium', price: '$79.90', per: 'año',
        promo: '2 meses gratis',
        cta: 'Consigue Premium',
        features: ['Análisis gramatical automático', 'Respuestas IA más rápidas', 'Mensajes de más de 200 caracteres', 'Cuotas diarias más altas'],
      },
      enterprise: {
        name: 'Empresa', desc: 'Autoalojado o implementación personalizada para equipos.',
        cta: 'Contáctanos',
        features: ['Autoalojamiento y dominios personalizados', 'Soporte dedicado', 'Opciones de volumen y SLA'],
      },
      note: 'Plan gratuito para siempre: no se requiere tarjeta de crédito.',
    },
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
    nav: { features: 'الميزات', how: 'كيف يعمل', languages: 'اللغات', pricing: 'الأسعار', launch: 'تشغيل التطبيق', login: 'تسجيل الدخول' },
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
    pricing: {
      title: 'أسعار بسيطة',
      subtitle: 'ابدأ مجانًا. قم بالترقية عندما تكون مستعدًا.',
      free: {
        name: 'مجاني', price: '$0', per: 'للأبد',
        cta: 'الخطة الحالية',
        features: ['محادثات ومجموعات غير محدودة', 'ترجمة فورية حتى 200 حرف', 'أدوات قواعد ومفردات عند الطلب', 'بحث في جميع الرسائل'],
      },
      premium: {
        name: 'بريميوم', price: '$79.90', per: 'سنة',
        promo: 'شهران مجانًا',
        cta: 'اشترك في بريميوم',
        features: ['تحليل نحوي تلقائي', 'استجابات ذكاء اصطناعي أسرع', 'رسائل أطول من 200 حرف', 'حصص يومية أعلى'],
      },
      enterprise: {
        name: 'المؤسسات', desc: 'استضافة ذاتية أو نشر مخصص للفرق.',
        cta: 'اتصل بنا',
        features: ['استضافة ذاتية ومجالات مخصصة', 'دعم مخصص', 'خيارات الحجم واتفاقيات SLA'],
      },
      note: 'خطة مجانية للأبد — لا حاجة لبطاقة ائتمان.',
    },
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
    nav: { features: 'Fonctionnalités', how: 'Comment ça Marche', languages: 'Langues', pricing: 'Tarifs', launch: 'Lancer l\'App', login: 'Connexion' },
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
    pricing: {
      title: 'Tarifs Simples',
      subtitle: 'Commencez gratuitement. Passez à la vitesse supérieure quand vous voulez.',
      free: {
        name: 'Gratuit', price: '$0', per: 'pour toujours',
        cta: 'Forfait Actuel',
        features: ['Chats et groupes illimités', 'Traduction en direct jusqu\'à 200 caractères', 'Outils de grammaire et de vocabulaire à la demande', 'Recherche dans tous les messages'],
      },
      premium: {
        name: 'Premium', price: '$79.90', per: 'an',
        promo: '2 mois offerts',
        cta: 'Obtenir Premium',
        features: ['Analyse grammaticale automatique', 'Réponses IA plus rapides', 'Messages de plus de 200 caractères', 'Quotas journaliers plus élevés'],
      },
      enterprise: {
        name: 'Entreprise', desc: 'Auto-hébergé ou déploiement personnalisé pour les équipes.',
        cta: 'Contactez-nous',
        features: ['Auto-hébergement et domaines personnalisés', 'Assistance dédiée', 'Options de volume et de SLA'],
      },
      note: 'Forfait gratuit pour toujours — aucune carte de crédit requise.',
    },
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
    nav: { features: 'বৈশিষ্ট্য', how: 'কীভাবে কাজ করে', languages: 'ভাষা', pricing: 'দাম', launch: 'অ্যাপ চালু করুন', login: 'লগ ইন' },
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
    pricing: {
      title: 'সহজ দাম',
      subtitle: 'বিনামূল্যে শুরু করুন। প্রস্তুত হলে আপগ্রেড করুন।',
      free: {
        name: 'ফ্রি', price: '$0', per: 'সব সময়ের জন্য',
        cta: 'বর্তমান প্ল্যান',
        features: ['আনলিমিটেড চ্যাট ও গ্রুপ', '২০০ অক্ষর পর্যন্ত লাইভ অনুবাদ', 'চাহিদা অনুযায়ী ব্যাকরণ ও শব্দভান্ডার টুল', 'সব বার্তায় খোঁজ'],
      },
      premium: {
        name: 'প্রিমিয়াম', price: '$79.90', per: 'বছর',
        promo: '২ মাস ফ্রি',
        cta: 'প্রিমিয়াম নিন',
        features: ['স্বয়ংক্রিয় ব্যাকরণ বিশ্লেষণ', 'দ্রুততর AI উত্তর', '২০০ অক্ষরের বেশি বার্তা', 'উচ্চতর দৈনিক সীমা'],
      },
      enterprise: {
        name: 'এন্টারপ্রাইজ', desc: 'দলের জন্য সেলফ-হোস্টেড বা কাস্টম ডিপ্লয়মেন্ট।',
        cta: 'যোগাযোগ করুন',
        features: ['সেলফ-হোস্টিং ও কাস্টম ডোমেইন', 'ডেডিকেটেড সাপোর্ট', 'ভলিউম ও SLA বিকল্প'],
      },
      note: 'চিরকালের ফ্রি প্ল্যান — কোনো ক্রেডিট কার্ড লাগবে না।',
    },
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
    nav: { features: 'Recursos', how: 'Como Funciona', languages: 'Idiomas', pricing: 'Preços', launch: 'Abrir App', login: 'Entrar' },
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
    pricing: {
      title: 'Preços Simples',
      subtitle: 'Comece grátis. Faça upgrade quando estiver pronto.',
      free: {
        name: 'Grátis', price: '$0', per: 'para sempre',
        cta: 'Plano Atual',
        features: ['Chats e grupos ilimitados', 'Tradução ao vivo até 200 caracteres', 'Ferramentas de gramática e vocabulário sob demanda', 'Busca em todas as mensagens'],
      },
      premium: {
        name: 'Premium', price: '$79.90', per: 'ano',
        promo: '2 meses grátis',
        cta: 'Obter Premium',
        features: ['Análise gramatical automática', 'Respostas de IA mais rápidas', 'Mensagens com mais de 200 caracteres', 'Cotas diárias maiores'],
      },
      enterprise: {
        name: 'Empresa', desc: 'Auto-hospedado ou implantação personalizada para equipes.',
        cta: 'Fale conosco',
        features: ['Auto-hospedagem e domínios personalizados', 'Suporte dedicado', 'Opções de volume e SLA'],
      },
      note: 'Plano grátis para sempre — sem cartão de crédito.',
    },
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
    nav: { features: 'Возможности', how: 'Как это работает', languages: 'Языки', pricing: 'Цены', launch: 'Открыть приложение', login: 'Войти' },
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
    pricing: {
      title: 'Простые тарифы',
      subtitle: 'Начните бесплатно. Улучшайте тариф, когда будете готовы.',
      free: {
        name: 'Бесплатный', price: '$0', per: 'навсегда',
        cta: 'Текущий план',
        features: ['Безлимитные чаты и группы', 'Перевод в реальном времени до 200 символов', 'Инструменты грамматики и словаря по запросу', 'Поиск по всем сообщениям'],
      },
      premium: {
        name: 'Premium', price: '$79.90', per: 'год',
        promo: '2 месяца бесплатно',
        cta: 'Получить Premium',
        features: ['Автоматический грамматический анализ', 'Более быстрые ответы ИИ', 'Сообщения длиннее 200 символов', 'Более высокие дневные лимиты'],
      },
      enterprise: {
        name: 'Для команд', desc: 'Самодеплой или индивидуальное развёртывание для команд.',
        cta: 'Свяжитесь с нами',
        features: ['Самодеплой и собственные домены', 'Выделенная поддержка', 'Варианты объёма и SLA'],
      },
      note: 'Бесплатный план навсегда — кредитная карта не нужна.',
    },
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
    nav: { features: 'خصوصیات', how: 'یہ کیسے کام کرتا ہے', languages: 'زبانیں', pricing: 'قیمتیں', launch: 'ایپ شروع کریں', login: 'لاگ ان' },
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
    pricing: {
      title: 'سادہ قیمتیں',
      subtitle: 'مفت شروع کریں۔ جب تیار ہوں تو اپ گریڈ کریں۔',
      free: {
        name: 'مفت', price: '$0', per: 'ہمیشہ کے لیے',
        cta: 'موجودہ پلان',
        features: ['لا محدود چیٹس اور گروپس', '200 حروف تک لائیو ترجمہ', 'طلب پر گرامر اور الفاظ کے ٹولز', 'تمام پیغامات میں تلاش'],
      },
      premium: {
        name: 'پریمیم', price: '$79.90', per: 'سال',
        promo: '2 مہینے مفت',
        cta: 'پریمیم حاصل کریں',
        features: ['خودکار گرامر تجزیہ', 'تیز AI جوابات', '200 حروف سے لمبے پیغامات', 'اعلیٰ روزانہ حدود'],
      },
      enterprise: {
        name: 'انٹرپرائز', desc: 'ٹیموں کے لیے سیلف ہوسٹڈ یا کسٹم ڈپلائمنٹ۔',
        cta: 'ہم سے رابطہ کریں',
        features: ['سیلف ہوسٹنگ اور کسٹم ڈومین', 'مخصوص سپورٹ', 'حجم اور SLA اختیارات'],
      },
      note: 'ہمیشہ کے لیے مفت پلان — کوئی کریڈٹ کارڈ درکار نہیں۔',
    },
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

const FEATURE_ACCENTS = [
  { circle: 'bg-primary-container/10', icon: 'text-primary' },
  { circle: 'bg-secondary-container/10', icon: 'text-secondary' },
  { circle: 'bg-tertiary-container/10', icon: 'text-tertiary' },
  { circle: 'bg-primary-container/10', icon: 'text-primary' },
  { circle: 'bg-secondary-container/10', icon: 'text-secondary' },
  { circle: 'bg-tertiary-container/10', icon: 'text-tertiary' },
]

export default function Landing() {
  const [selectedLang, setSelectedLang] = useState(() => localStorage.getItem('preferredLanguage') || detectBrowserLanguage())
  const { t: ti18n } = useTranslation()
  const user = useStore(s => s.user)
  const lang = STRINGS[selectedLang] ? selectedLang : 'en'
  const t = STRINGS[lang]
  const nativeName = getNativeLanguageName(lang)
  const supportedCount = TOP10.length

  const handleLanguageChange = (code: string) => {
    setSelectedLang(code)
    localStorage.setItem('preferredLanguage', code)
  }

  return (
    <div className="min-h-screen bg-background text-on-surface" lang={lang}>
      {/* Navigation */}
      <nav className="fixed top-0 w-full bg-surface/90 backdrop-blur border-b border-outline-variant/40 z-50">
        <div className="max-w-6xl mx-auto px-6 py-4 flex justify-between items-center">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-primary-container/10 rounded-full flex items-center justify-center">
              <span className="material-symbols-outlined text-primary text-2xl" style={{ fontVariationSettings: "'FILL' 1" }}>graphic_eq</span>
            </div>
            <span className="font-headline-md text-headline-md font-bold text-primary">Chorus</span>
          </div>
          <div className="flex items-center gap-4">
            <ul className="hidden md:flex gap-8 items-center">
              <li><a href="#features" className="text-on-surface-variant hover:text-primary transition">{t.nav.features}</a></li>
              <li><a href="#how-it-works" className="text-on-surface-variant hover:text-primary transition">{t.nav.how}</a></li>
              <li><a href="#languages" className="text-on-surface-variant hover:text-primary transition">{t.nav.languages}</a></li>
              <li><Link to="/pricing" className="text-on-surface-variant hover:text-primary transition">{t.nav.pricing}</Link></li>
            </ul>
            <LanguageSelector
              currentLang={lang}
              onLanguageChange={handleLanguageChange}
              variant="navbar"
            />
            <a
              href="https://discord.gg/7DVwM6jsS"
              target="_blank"
              rel="noreferrer"
              aria-label={ti18n('nav.joinChorusDiscord')}
              className="hidden sm:flex h-10 w-10 items-center justify-center rounded-lg bg-[#5865F2] text-white transition hover:bg-[#4752c4]"
            >
              <svg className="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M20.317 4.37a19.79 19.79 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 0 0-.041-.106 13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.892.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z" />
              </svg>
            </a>
            {!user && (
              <Link to="/waitlist" className="px-4 py-2 bg-primary text-on-primary rounded-lg hover:bg-on-primary-fixed-variant transition whitespace-nowrap">{ti18n('nav.joinWaitlist')}</Link>
            )}
            {user ? (
              <Link to="/chat" className="px-4 py-2 border border-outline-variant text-on-surface rounded-lg hover:border-primary hover:text-primary transition whitespace-nowrap">
                {ti18n('nav.openApp')}
              </Link>
            ) : (
              <Link to="/login" className="px-4 py-2 border border-outline-variant text-on-surface rounded-lg hover:border-primary hover:text-primary transition whitespace-nowrap">
                {t.nav.login}
              </Link>
            )}
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="pt-32 pb-20 px-6">
        <div className="max-w-6xl mx-auto">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-12 items-center">
            <div>
              <div className="inline-block bg-primary-fixed text-primary text-sm font-semibold px-3 py-1 rounded-full mb-4">
                🌍 {nativeName} · {t.hero.badge.replace('{count}', String(supportedCount))}
              </div>
              <h1 className="font-headline-lg text-5xl md:text-6xl font-bold mb-6">
                {t.hero.titleA},<br />
                <span className="bg-gradient-to-r from-primary to-secondary bg-clip-text text-transparent">{t.hero.titleB}</span>
              </h1>
              <p className="text-body-lg text-on-surface-variant mb-8">
                {t.hero.subtitle}
              </p>
              <div className="flex gap-4 mb-12 flex-wrap">
                <Link to={user ? '/chat' : '/login'} className="px-8 py-4 bg-primary text-on-primary rounded-xl font-label-md text-label-md shadow-lg hover:bg-on-primary-fixed-variant transition">
                  {user ? ti18n('nav.openApp') : t.nav.login}
                </Link>
                {!user && (
                  <Link to="/waitlist" className="px-8 py-4 border-2 border-primary text-primary rounded-xl font-label-md text-label-md hover:bg-primary-fixed transition">
                    {ti18n('nav.joinWaitlist')}
                  </Link>
                )}
                <a href="#how-it-works" className="px-8 py-4 border-2 border-outline-variant text-on-surface-variant rounded-xl font-label-md text-label-md hover:bg-surface-container-low transition">
                  {t.hero.seeHow}
                </a>
              </div>
              <div className="flex gap-8">
                <div>
                  <div className="text-3xl font-bold text-primary">{supportedCount}</div>
                  <p className="text-on-surface-variant">{t.hero.stats.languages}</p>
                </div>
                <div>
                  <div className="text-3xl font-bold text-primary">{t.hero.stats.translation}</div>
                  <p className="text-on-surface-variant">{t.features.items[0].title}</p>
                </div>
                <div>
                  <div className="text-3xl font-bold text-primary">100%</div>
                  <p className="text-on-surface-variant">{t.hero.stats.free}</p>
                </div>
              </div>
            </div>
            <div className="relative">
              <div className="rounded-[2rem] overflow-hidden shadow-[0px_8px_24px_rgba(0,0,0,0.1)] bg-surface-container">
                <div className="h-72 md:h-96 bg-gradient-to-br from-primary-container/15 via-surface-container to-secondary-container/15 flex flex-col items-center justify-center gap-3">
                  <span className="material-symbols-outlined text-primary text-[72px]" style={{ fontVariationSettings: "'FILL' 1" }}>translate</span>
                </div>
                <div className="absolute bottom-6 left-6 right-6 bg-white/90 backdrop-blur-md rounded-2xl p-4 shadow-[0px_4px_12px_rgba(0,0,0,0.05)] border-l-2 border-secondary-container">
                  <div className="flex items-center gap-3 mb-2">
                    <div className="w-8 h-8 rounded-full bg-surface-container flex items-center justify-center">
                      <span className="material-symbols-outlined text-primary text-sm" style={{ fontVariationSettings: "'FILL' 1" }}>smart_toy</span>
                    </div>
                    <span className="font-label-md text-label-md text-on-surface">AI Tutor Sparky</span>
                  </div>
                  <p className="font-body-md text-body-md text-on-surface mb-1">{t.hero.chat.bubble1}</p>
                  <p className="font-translation-text text-translation-text text-secondary">{t.hero.chat.bubble1Trans}</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section id="features" className="py-20 px-6 bg-surface-container-low">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="font-headline-md text-4xl md:text-5xl font-bold mb-4">{t.features.title}</h2>
            <p className="text-body-lg text-on-surface-variant">{t.features.subtitle}</p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {t.features.items.map((feature, i) => (
              <div key={i} className="bg-surface-container-lowest p-8 rounded-2xl shadow-[0px_4px_12px_rgba(0,0,0,0.05)] hover:shadow-xl transition">
                <div className={`w-12 h-12 rounded-full ${FEATURE_ACCENTS[i % FEATURE_ACCENTS.length].circle} flex items-center justify-center mb-4`}>
                  <span className={`text-2xl ${FEATURE_ACCENTS[i % FEATURE_ACCENTS.length].icon}`}>{feature.icon}</span>
                </div>
                <h3 className="font-headline-sm text-headline-sm text-on-surface mb-2">{feature.title}</h3>
                <p className="font-body-md text-body-md text-on-surface-variant">{feature.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* How It Works */}
      <section id="how-it-works" className="py-20 px-6">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="font-headline-md text-4xl md:text-5xl font-bold mb-4">{t.how.title}</h2>
            <p className="text-body-lg text-on-surface-variant">{t.how.subtitle}</p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            {t.how.steps.map((step, i) => (
              <div key={i} className="text-center">
                <div className="w-16 h-16 bg-gradient-to-br from-primary to-secondary text-on-primary rounded-full flex items-center justify-center mx-auto mb-4 text-2xl font-bold">
                  {step.num}
                </div>
                <h3 className="font-headline-sm text-headline-sm text-on-surface mb-3">{step.title}</h3>
                <p className="font-body-md text-body-md text-on-surface-variant">{step.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Languages Section */}
      <section id="languages" className="py-20 px-6 bg-surface-container-low">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="font-headline-md text-4xl md:text-5xl font-bold mb-4">{t.languages.title}</h2>
            <p className="text-body-lg text-on-surface-variant">{t.languages.subtitle}</p>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-10 gap-4">
            {TOP10.map((code, i) => {
              const langInfo = SUPPORTED_LANGUAGES.find(l => l.code === code) || SUPPORTED_LANGUAGES[0]
              return (
                <div key={i} className="bg-surface-container-lowest p-6 rounded-xl text-center shadow-[0px_4px_12px_rgba(0,0,0,0.05)] hover:shadow-xl transition">
                  <div className="text-4xl mb-2">{langInfo.flag}</div>
                  <p className="font-semibold text-on-surface">{langInfo.name}</p>
                  <p className="text-xs text-on-surface-variant/60 mt-1">{langInfo.nativeName}</p>
                </div>
              )
            })}
          </div>
        </div>
      </section>

      {/* Pricing Section */}
      <section id="pricing" className="py-20 px-6">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="font-headline-md text-4xl md:text-5xl font-bold mb-4">{t.pricing.title}</h2>
            <p className="text-body-lg text-on-surface-variant">{t.pricing.subtitle}</p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            {/* Free */}
            <div className="bg-surface-container-lowest rounded-2xl shadow-[0px_4px_12px_rgba(0,0,0,0.05)] p-8 flex flex-col">
              <h3 className="text-2xl font-bold mb-1">{t.pricing.free.name}</h3>
              <p className="text-4xl font-bold mb-1">{t.pricing.free.price}<span className="text-base font-normal text-on-surface-variant">/{t.pricing.free.per}</span></p>
              <ul className="mt-6 mb-8 space-y-3 flex-1">
                {t.pricing.free.features.map((f, i) => (
                  <li key={i} className="flex items-start gap-2 text-on-surface-variant">
                    <span className="text-tertiary mt-0.5">✓</span>{f}
                  </li>
                ))}
              </ul>
              <button className="w-full px-4 py-3 border border-outline-variant rounded-lg text-on-surface-variant font-semibold cursor-default">{t.pricing.free.cta}</button>
            </div>

            {/* Premium */}
            <div className="bg-gradient-to-br from-primary to-secondary rounded-2xl p-8 text-on-primary shadow-2xl flex flex-col md:-my-4">
              <h3 className="text-2xl font-bold mb-1">✦ {t.pricing.premium.name}</h3>
              <p className="text-base font-semibold text-on-primary/80 mb-0.5"><s>{YEARLY_LIST_PRICE}</s></p>
              <p className="text-4xl font-bold mb-1">{t.pricing.premium.price}<span className="text-base font-normal text-on-primary/80">/{t.pricing.premium.per}</span></p>
              <p className="text-sm font-semibold text-on-primary/90 mb-2">✦ {t.pricing.premium.promo}</p>
              <ul className="mt-6 mb-8 space-y-3 flex-1">
                {t.pricing.premium.features.map((f, i) => (
                  <li key={i} className="flex items-start gap-2">
                    <span className="text-tertiary-fixed mt-0.5">✓</span>{f}
                  </li>
                ))}
              </ul>
              <Link to="/pricing" className="w-full px-4 py-3 bg-white text-primary rounded-lg font-bold text-center hover:bg-gray-100 transition block">
                {t.pricing.premium.cta}
              </Link>
            </div>

            {/* Enterprise */}
            <div className="bg-surface-container-lowest rounded-2xl shadow-[0px_4px_12px_rgba(0,0,0,0.05)] p-8 flex flex-col">
              <h3 className="text-2xl font-bold mb-1">{t.pricing.enterprise.name}</h3>
              <p className="text-on-surface-variant mb-1">{t.pricing.enterprise.desc}</p>
              <ul className="mt-6 mb-8 space-y-3 flex-1">
                {t.pricing.enterprise.features.map((f, i) => (
                  <li key={i} className="flex items-start gap-2 text-on-surface-variant">
                    <span className="text-tertiary mt-0.5">✓</span>{f}
                  </li>
                ))}
              </ul>
              <a href="mailto:hello@chorus.talk?subject=Enterprise%20Enquiry" className="w-full px-4 py-3 border border-outline-variant rounded-lg text-on-surface font-semibold text-center hover:border-primary hover:text-primary transition">
                {t.pricing.enterprise.cta}
              </a>
            </div>
          </div>

          <p className="text-center text-sm text-on-surface-variant mt-8">{t.pricing.note}</p>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20 px-6 bg-gradient-to-r from-primary to-secondary">
        <div className="max-w-4xl mx-auto text-center text-on-primary">
          <h2 className="font-headline-md text-4xl md:text-5xl font-bold mb-4">{t.cta.title}</h2>
          <p className="text-xl mb-8 opacity-90">{t.cta.subtitle}</p>
          {user ? (
            <Link to="/chat" className="px-8 py-4 bg-white text-primary rounded-lg font-bold text-lg hover:bg-gray-100 transition inline-block">
              {ti18n('nav.openApp')}
            </Link>
          ) : (
            <Link to="/waitlist" className="px-8 py-4 bg-white text-primary rounded-lg font-bold text-lg hover:bg-gray-100 transition inline-block">
              {ti18n('nav.joinWaitlist')}
            </Link>
          )}
          <a href="https://discord.gg/7DVwM6jsS" target="_blank" rel="noreferrer" className="ml-3 px-8 py-4 border-2 border-white/60 text-on-primary rounded-lg font-bold text-lg hover:bg-white hover:text-primary transition inline-block">
            {ti18n('nav.joinDiscord')}
          </a>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-inverse-surface text-white py-12 px-6">
        <div className="max-w-6xl mx-auto">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-8 mb-8">
            <div>
              <div className="flex items-center gap-2 mb-4">
                <div className="w-8 h-8 bg-primary-container/10 rounded-full flex items-center justify-center">
                  <span className="material-symbols-outlined text-primary-fixed text-base" style={{ fontVariationSettings: "'FILL' 1" }}>graphic_eq</span>
                </div>
                <span className="font-headline-md text-headline-md font-bold">Chorus</span>
              </div>
              <p className="text-gray-400">{t.footer.tagline}</p>
            </div>
            <div>
              <h4 className="font-bold mb-4">{t.footer.product}</h4>
              <ul className="space-y-2 text-gray-400">
                {t.footer.productLinks.filter(l => l.href !== '/login').map((l, i) => (
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
                <li>
                  <a href="https://discord.gg/7DVwM6jsS" target="_blank" rel="noreferrer" className="hover:text-white">
                    {ti18n('nav.discord')}
                  </a>
                </li>
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