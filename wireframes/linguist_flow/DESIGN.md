---
name: Linguist Flow
colors:
  surface: '#f8f9ff'
  surface-dim: '#cbdbf5'
  surface-bright: '#f8f9ff'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#eff4ff'
  surface-container: '#e5eeff'
  surface-container-high: '#dce9ff'
  surface-container-highest: '#d3e4fe'
  on-surface: '#0b1c30'
  on-surface-variant: '#434655'
  inverse-surface: '#213145'
  inverse-on-surface: '#eaf1ff'
  outline: '#737686'
  outline-variant: '#c3c6d7'
  surface-tint: '#0053db'
  primary: '#004ac6'
  on-primary: '#ffffff'
  primary-container: '#2563eb'
  on-primary-container: '#eeefff'
  inverse-primary: '#b4c5ff'
  secondary: '#6b38d4'
  on-secondary: '#ffffff'
  secondary-container: '#8455ef'
  on-secondary-container: '#fffbff'
  tertiary: '#006242'
  on-tertiary: '#ffffff'
  tertiary-container: '#007d55'
  on-tertiary-container: '#bdffdb'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#dbe1ff'
  primary-fixed-dim: '#b4c5ff'
  on-primary-fixed: '#00174b'
  on-primary-fixed-variant: '#003ea8'
  secondary-fixed: '#e9ddff'
  secondary-fixed-dim: '#d0bcff'
  on-secondary-fixed: '#23005c'
  on-secondary-fixed-variant: '#5516be'
  tertiary-fixed: '#6ffbbe'
  tertiary-fixed-dim: '#4edea3'
  on-tertiary-fixed: '#002113'
  on-tertiary-fixed-variant: '#005236'
  background: '#f8f9ff'
  on-background: '#0b1c30'
  surface-variant: '#d3e4fe'
typography:
  headline-lg:
    fontFamily: Plus Jakarta Sans
    fontSize: 30px
    fontWeight: '700'
    lineHeight: 38px
    letterSpacing: -0.02em
  headline-md:
    fontFamily: Plus Jakarta Sans
    fontSize: 24px
    fontWeight: '700'
    lineHeight: 32px
    letterSpacing: -0.01em
  headline-sm:
    fontFamily: Plus Jakarta Sans
    fontSize: 20px
    fontWeight: '600'
    lineHeight: 28px
  body-lg:
    fontFamily: Be Vietnam Pro
    fontSize: 18px
    fontWeight: '400'
    lineHeight: 28px
  body-md:
    fontFamily: Be Vietnam Pro
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 24px
  body-sm:
    fontFamily: Be Vietnam Pro
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
  label-md:
    fontFamily: Inter
    fontSize: 13px
    fontWeight: '600'
    lineHeight: 18px
    letterSpacing: 0.01em
  label-sm:
    fontFamily: Inter
    fontSize: 11px
    fontWeight: '500'
    lineHeight: 16px
    letterSpacing: 0.03em
  translation-text:
    fontFamily: Be Vietnam Pro
    fontSize: 14px
    fontWeight: '500'
    lineHeight: 20px
rounded:
  sm: 0.25rem
  DEFAULT: 0.5rem
  md: 0.75rem
  lg: 1rem
  xl: 1.5rem
  full: 9999px
spacing:
  unit: 4px
  margin-mobile: 16px
  gutter-mobile: 12px
  stack-sm: 8px
  stack-md: 16px
  stack-lg: 24px
---

## Brand & Style

The design system is built on the intersection of **Modern Social Messaging** and **Educational Technology**. It prioritizes high legibility and an approachable atmosphere to reduce the cognitive load associated with language learning. 

The style is **Modern / Humanist**, utilizing soft geometry and generous whitespace to create a "safe space" for making mistakes. It avoids the clinical feel of traditional education tools in favor of the dynamic, fluid motion found in top-tier social platforms. The emotional goal is to evoke confidence, curiosity, and clarity.

## Colors

The palette is anchored by **Fluency Blue** (#2563EB), representing trust and communication. **Insight Purple** (#8B5CF6) is reserved specifically for AI-powered interactions, such as instant translations, grammar corrections, and learning hints.

- **Primary (Fluency Blue):** Core UI actions, active states, and user message bubbles.
- **Secondary (Insight Purple):** AI features, "Magic" buttons, and translation overlays.
- **Success (Emerald):** Correct answers, completed milestones, and progress bars.
- **Warning/Error (Rose):** Error states and areas needing review.
- **Neutrals:** Slate-based grays provide a high-contrast foundation for text without the harshness of pure black.

## Typography

This design system uses a multi-font strategy to balance personality with extreme legibility. 

- **Plus Jakarta Sans** is used for headings to provide a friendly, modern tech aesthetic.
- **Be Vietnam Pro** is used for body text and message bubbles; its clean terminals and generous x-height make it exceptionally readable for non-native speakers and various character sets.
- **Inter** is used for functional labels and micro-copy due to its systematic and neutral nature.

**Hierarchy Note:** Translated text should always be rendered 2pt smaller than the original text and styled with `translation-text` tokens (Medium weight, italicized) to distinguish it from primary communication.

## Layout & Spacing

The layout follows a **Fluid Grid** model optimized for one-handed mobile use. 

- **Horizontal Margins:** A standard 16px margin is maintained on mobile devices.
- **Message Bubbles:** Use a 12px gutter between the bubble and the screen edge.
- **Rhythm:** An 8px (2-unit) base grid drives all vertical spacing.
- **AI Tiers:** Elements containing AI insights (translations/grammar) should use an additional 4px of internal padding to accommodate the specific Purple accent border.

## Elevation & Depth

Visual hierarchy is established through **Tonal Layering** and **Ambient Shadows**.

- **Level 0 (Background):** Surface color is Slate-50 (#F8FAFC).
- **Level 1 (Cards/Bubbles):** White (#FFFFFF) with a very soft, diffused shadow (0px 4px 12px, 5% opacity black).
- **Level 2 (Modals/Popovers):** Higher contrast shadow (0px 8px 24px, 10% opacity black) to pull the element toward the user.
- **AI Features:** Use a subtle "Insight Purple" glow (2px blur, 10% opacity) rather than a standard gray shadow to signify the "smart" nature of the content.

## Shapes

The shape language is **Rounded**, conveying a soft and approachable personality.

- **Buttons & Inputs:** Use the standard `rounded` (0.5rem) token.
- **Message Bubbles:** Use `rounded-lg` (1rem) for the outer corners, while the tail corner remains slightly sharper (0.25rem) to indicate directionality.
- **Progress Bars:** Fully pill-shaped (rounded-xl) to feel continuous and smooth.

## Components

### Buttons
- **Primary:** Fluency Blue background, White text. High-visibility for "Send" or "Continue."
- **Ghost/Tertiary:** No background, Blue text. Used for secondary actions like "View Original."

### Message Bubbles
- **Incoming:** White background with a light Slate-200 border.
- **Outgoing:** Fluency Blue background with White text.
- **AI-Enhanced:** Subtle Purple-50 background with a 1px solid Insight Purple left-border. This houses the original text and the translated counterpart.

### Learning Progress
- **Progress Bar:** A track in Slate-100 with a fill in Emerald Green.
- **Streak Counter:** A small chip containing a flame icon and Label-MD text, using a subtle Orange-50 background.

### Input Fields
- **Chat Input:** A single-line field with 1rem roundedness, containing icons for "Voice Record" and "AI Hint."
- **Focus State:** 2px solid Fluency Blue border with no inner shadow.

### Chips & Badges
- **Correction Chip:** Small, soft-red background with white text, used to highlight specific grammar mistakes within a sentence.
- **Level Badge:** A small circular badge with Label-SM text (e.g., "A2" or "B1") placed next to the user's avatar.