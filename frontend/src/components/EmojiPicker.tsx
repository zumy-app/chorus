import { useEffect, useRef, useState } from 'react'

export interface EmojiCategory {
  tab: string
  name: string
  emojis: string[]
}

// A curated set of emoji so the picker stays fast and mobile-friendly. Each
// entry is a short, stable sequence that passes through translation unchanged
// (emoji are never stripped or rewritten by the translation pipeline).
export const EMOJI_CATEGORIES: EmojiCategory[] = [
  {
    tab: '😀',
    name: 'Smileys',
    emojis: [
      '😀', '😁', '😂', '🤣', '😃', '😄', '😅', '😆', '😉', '😊',
      '😋', '😎', '😍', '😘', '🥰', '😗', '😙', '😚', '🙂', '🤗',
      '🤩', '🤔', '🤨', '😐', '😑', '😶', '🙄', '😏', '😣', '😥',
      '😮', '🤐', '😯', '😪', '😫', '😴', '😌', '😛', '😜', '😝',
      '🤤', '😒', '😓', '😔', '😕', '🙃', '🤑', '😲', '☹️', '🙁',
      '😖', '😞', '😟', '😤', '😢', '😭', '😦', '😧', '😨', '😩',
      '🤯', '😬', '😰', '😱', '🥵', '🥶', '😳', '🤪', '😵', '😡',
      '😠', '🤬', '😷', '🤒', '🤕', '🤢', '🤮', '🤧', '😇', '🤠',
      '🤡', '🤥', '🤫', '🤭', '🧐', '🤓', '😈', '👿', '👹', '👺',
      '💩', '👻', '💀', '👽', '👾', '🤖', '🎃', '😺', '😸', '😹',
    ],
  },
  {
    tab: '👋',
    name: 'Gestures',
    emojis: [
      '👋', '🤚', '✋', '🖖', '👌', '✌️', '🤞', '🤟', '🤘', '🤙',
      '👈', '👉', '👆', '🖕', '👇', '☝️', '👍', '👎', '✊', '👊',
      '🤛', '🤜', '👏', '🙌', '👐', '🤲', '🤝', '🙏', '💪', '🦾',
      '🖤', '👀', '🧠', '🦷', '👅', '👄', '🦶', '👣', '👂', '👃',
      '🗣️', '👤', '🕴️',
    ],
  },
  {
    tab: '❤️',
    name: 'Hearts',
    emojis: [
      '❤️', '🧡', '💛', '💚', '💙', '💜', '🖤', '🤍', '🤎', '💔',
      '❣️', '💕', '💞', '💓', '💗', '💖', '💘', '💝', '💟', '♥️',
      '💌', '💋', '💯', '💢', '💥', '💫', '💦', '💨', '💣', '💬',
      '👁️', '💤', '🗨️', '💭',
    ],
  },
  {
    tab: '🐶',
    name: 'Animals',
    emojis: [
      '🐶', '🐱', '🐭', '🐹', '🐰', '🦊', '🐻', '🐼', '🐨', '🐯',
      '🦁', '🐮', '🐷', '🐸', '🐵', '🙈', '🙉', '🙊', '🐒', '🐔',
      '🐧', '🐦', '🐤', '🦆', '🦅', '🦉', '🦇', '🐺', '🐗', '🐴',
      '🦄', '🐝', '🐛', '🦋', '🐌', '🐞', '🐢', '🐍', '🦎', '🐙',
      '🦑', '🦐', '🦀', '🐡', '🐠', '🐟', '🐬', '🐳', '🐋', '🦈',
      '🐊', '🐅', '🦓', '🦍', '🐘', '🦒', '🦘', '🐏', '🐑', '🦙',
      '🐐', '🦌', '🐕', '🐩', '🐈', '🦃', '🦚', '🦜', '🦢', '🦩',
      '🕊️', '🐇', '🦔', '🐁', '🌵', '🎄', '🌲', '🌳', '🌴',
      '🌱', '🌿', '☘️', '🍀', '🍃', '🍂', '🍁', '🌷', '🌹', '🌺',
      '🌸', '🌼', '🌻', '🌞', '🌝', '🌛', '🌜', '🌚', '🌍', '🌎',
      '🌏', '⭐', '🌟', '✨', '⚡', '🔥', '🌈', '☀️', '❄️', '⛄',
      '🌊',
    ],
  },
  {
    tab: '🍕',
    name: 'Food',
    emojis: [
      '🍏', '🍎', '🍐', '🍊', '🍋', '🍌', '🍉', '🍇', '🍓', '🍈',
      '🍒', '🍑', '🥭', '🍍', '🥥', '🥝', '🍅', '🥑', '🥦', '🥬',
      '🥒', '🌶️', '🌽', '🥕', '🧄', '🧅', '🥔', '🍠', '🥐', '🍞',
      '🥖', '🥨', '🧀', '🥚', '🍳', '🥞', '🧇', '🥓', '🥩', '🍗',
      '🍖', '🌭', '🍔', '🍟', '🍕', '🥪', '🥙', '🌮', '🌯', '🥗',
      '🥘', '🍝', '🍜', '🍲', '🍛', '🍣', '🍱', '🥟', '🍤', '🍙',
      '🍚', '🍧', '🍨', '🍦', '🥧', '🧁', '🍰', '🎂', '🍮', '🍭',
      '🍬', '🍫', '🍿', '🍩', '🍪', '🌰', '🥜', '🍯', '🥛', '🍼',
      '☕', '🍵', '🧃', '🥤', '🧋', '🍺', '🍻', '🥂', '🍷', '🥃',
      '🍸', '🍹', '🍾',
    ],
  },
  {
    tab: '⚽',
    name: 'Activities',
    emojis: [
      '⚽', '🏀', '🏈', '⚾', '🎾', '🏐', '🏉', '🏓', '🏸', '🏒',
      '🏑', '🥍', '🏏', '⛳', '🏹', '🎣', '🥊', '🥋', '🛹', '🛼',
      '⛸️', '🥌', '🎿', '⛷️', '🏂', '🏋️', '🤼', '🤸', '⛹️', '🤺',
      '🏌️', '🏇', '🧘', '🏄', '🏊', '🤽', '🚣', '🧗', '🚵', '🚴',
      '🏆', '🥇', '🥈', '🥉', '🏅', '🎖️', '🎫', '🎪', '🤹', '🎭',
      '🎨', '🎬', '🎤', '🎧', '🎼', '🎹', '🥁', '🎷', '🎺', '🎸',
      '🎻', '🎲', '🎯', '🎳', '🎮', '🎰',
    ],
  },
  {
    tab: '✈️',
    name: 'Travel',
    emojis: [
      '🚗', '🚕', '🚙', '🚌', '🚎', '🏎️', '🚓', '🚑', '🚒', '🚐',
      '🚚', '🚛', '🚜', '🚲', '🛴', '🛵', '🏍️', '🛺', '🚨', '🚔',
      '✈️', '🛫', '🛬', '🛩️', '💺', '🛰️', '🚀', '🛸', '🚁', '🛶',
      '⛵', '🚤', '🛥️', '🛳️', '⛴️', '🚢', '⚓', '⛽', '🚧', '🚦',
      '🗺️', '🗿', '🗽', '🗼', '🏰', '🏯', '🏟️', '🎡', '🎢', '🎠',
      '⛲', '⛱️', '🏖️', '🏝️', '🏜️', '🌋', '⛰️', '🏔️', '🗻', '🏕️',
      '⛺', '🏠', '🏡', '🏘️', '🏭', '🏢', '🏬', '🏥', '🏦', '🏨',
      '🏪', '🏫', '⛪', '🕌', '🕍', '🏙️', '🌃', '🌆', '🌇', '🌉',
      '🌅', '🌄', '🎇', '🎆',
    ],
  },
  {
    tab: '💡',
    name: 'Objects',
    emojis: [
      '⌚', '📱', '📲', '💻', '⌨️', '🖥️', '🖨️', '🖱️', '🕹️', '💾',
      '💿', '📀', '📼', '📷', '📸', '📹', '🎥', '📞', '☎️', '📟',
      '📺', '📻', '🎙️', '⏰', '⏳', '📡', '🔋', '🔌', '💡', '🔦',
      '🕯️', '💸', '💵', '💰', '💳', '💎', '⚖️', '🔧', '🔨', '⚙️',
      '🔫', '💣', '🔪', '🗡️', '⚔️', '🛡️', '🔮', '📿', '💈', '🔭',
      '🔬', '🩹', '🩺', '💊', '💉', '🩸', '🧬', '🦠', '🧫', '🧪',
      '🌡️', '🧹', '🧺', '🧻', '🚽', '🚿', '🛁', '🛀', '🧼', '🪒',
      '🧴', '🛎️', '🔑', '🗝️', '🚪', '🪑', '🛋️', '🛏️', '🛌', '🧸',
      '🖼️', '🛍️', '🛒', '🎁', '🎈', '🎏', '🎀', '🎊', '🎉', '🎎',
      '🏮', '📧', '📦', '📩', '📨', '📥', '📤', '📪', '📫', '📬',
      '📭', '📮', '📜', '📃', '📄', '📑', '📊', '📈', '📉', '🗒️',
      '📆', '📅', '🗑️', '📋', '📁', '📂', '🗞️', '📰', '📓', '📔',
      '📕', '📗', '📘', '📙', '📚', '📖', '🔖', '🔗', '📎', '📐',
      '📏', '🧮', '📌', '📍', '✂️', '🖊️', '🖋️', '✒️', '🖌️', '🖍️',
      '📝', '✏️', '🔍', '🔎', '🔏', '🔐', '🔒', '🔓',
    ],
  },
  {
    tab: '✅',
    name: 'Symbols',
    emojis: [
      '❗', '❕', '❓', '❔', '‼️', '⁉️', '🔅', '🔆', '💲', '♾️',
      '💱', '✅', '☑️', '✔️', '✖️', '❌', '❎', '➗', '➖', '➕',
      '©️', '®️', '™️', 'ℹ️', '🔚', '🔙', '🔛', '🔝', '🔜', '🔴',
      '🟠', '🟡', '🟢', '🔵', '🟣', '⚫', '⚪', '🟤', '🔺', '🔻',
      '🔸', '🔹', '🔶', '🔷', '🔳', '🔲', '▪️', '▫️', '◾', '◽',
      '◼️', '◻️', '🟥', '🟧', '🟨', '🟩', '🟦', '🟪', '⬛', '⬜',
      '🔈', '🔇', '🔉', '🔊', '🔔', '🔕', '📣', '📢', '💬', '🗯️',
    ],
  },
]

interface EmojiPickerProps {
  onSelect: (emoji: string) => void
  onClose: () => void
}

export default function EmojiPicker({ onSelect, onClose }: EmojiPickerProps) {
  const [active, setActive] = useState(0)
  const panelRef = useRef<HTMLDivElement>(null)

  // Close when the user clicks/taps anywhere outside the panel.
  useEffect(() => {
    const handlePointerDown = (e: MouseEvent | TouchEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        onClose()
      }
    }
    document.addEventListener('mousedown', handlePointerDown)
    document.addEventListener('touchstart', handlePointerDown)
    return () => {
      document.removeEventListener('mousedown', handlePointerDown)
      document.removeEventListener('touchstart', handlePointerDown)
    }
  }, [onClose])

  // Escape closes the picker.
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [onClose])

  const category = EMOJI_CATEGORIES[active]

  return (
    <div
      ref={panelRef}
      role="dialog"
      aria-label="Emoji picker"
      className="absolute bottom-full mb-2 left-0 z-50 rounded-2xl border border-outline-variant/30 bg-surface-container-lowest shadow-[0px_8px_24px_rgba(0,0,0,0.16)] overflow-hidden"
    >
      {/* Category tabs */}
      <div className="flex items-center gap-1 px-2 py-2 border-b border-outline-variant/20 bg-surface-container-high overflow-x-auto">
        {EMOJI_CATEGORIES.map((c, i) => (
          <button
            key={c.name}
            type="button"
            onClick={() => setActive(i)}
            aria-label={c.name}
            aria-pressed={i === active}
            className={`w-9 h-9 flex items-center justify-center rounded-lg text-[18px] transition shrink-0 ${
              i === active
                ? 'bg-primary-container text-on-primary-container'
                : 'hover:bg-surface-variant/30 text-on-surface-variant'
            }`}
          >
            {c.tab}
          </button>
        ))}
      </div>

      {/* Emoji grid */}
      <div className="grid grid-cols-8 gap-0.5 p-2 max-h-56 overflow-y-auto">
        {category.emojis.map((emoji) => (
          <button
            key={emoji}
            type="button"
            onClick={() => onSelect(emoji)}
            className="w-8 h-8 flex items-center justify-center text-[20px] rounded-lg hover:bg-surface-variant/30 transition active:scale-90"
          >
            {emoji}
          </button>
        ))}
      </div>
    </div>
  )
}

export const EMOJI_LIST = EMOJI_CATEGORIES
