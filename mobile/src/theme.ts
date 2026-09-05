// Chorus design system (Linguist Flow)
// Anchored on Fluency Blue (#2563EB) + Insight Purple (#6B38D4 / #8455EF).

export const COLORS = {
  // Legacy aliases (kept for existing screens; new design below)
  primary: '#004AC6',
  primaryDark: '#003EA8',
  purple: '#6B38D4',
  white: '#FFFFFF',
  bgLight: '#F8F9FF',
  textDark: '#0B1C30',
  textGray: '#434655',
  borderGray: '#C3C6D7',
  footerBg: '#213145',
};

export const COLOR = {
  // Brand
  primary: '#004AC6',
  onPrimary: '#FFFFFF',
  primaryContainer: '#2563EB',
  onPrimaryContainer: '#EEEFFF',
  primaryFixed: '#DBE1FF',
  primaryFixedDim: '#B4C5FF',
  onPrimaryFixed: '#00174B',
  onPrimaryFixedVariant: '#003EA8',
  inversePrimary: '#B4C5FF',

  // AI / Secondary (Insight Purple)
  secondary: '#6B38D4',
  onSecondary: '#FFFFFF',
  secondaryContainer: '#8455EF',
  onSecondaryContainer: '#FFFBFF',
  secondaryFixed: '#E9DDFF',
  secondaryFixedDim: '#D0BCFF',
  onSecondaryFixed: '#23005C',
  onSecondaryFixedVariant: '#5516BE',

  // Success / Tertiary (Emerald)
  tertiary: '#006242',
  onTertiary: '#FFFFFF',
  tertiaryContainer: '#007D55',
  onTertiaryContainer: '#BDFFDB',
  tertiaryFixed: '#6FFBBE',
  tertiaryFixedDim: '#4EDEA3',
  onTertiaryFixed: '#002113',
  onTertiaryFixedVariant: '#005236',

  // Surfaces
  background: '#F8F9FF',
  onBackground: '#0B1C30',
  surface: '#F8F9FF',
  onSurface: '#0B1C30',
  surfaceBright: '#F8F9FF',
  surfaceDim: '#CBDBF5',
  surfaceContainerLowest: '#FFFFFF',
  surfaceContainerLow: '#EFF4FF',
  surfaceContainer: '#E5EEFF',
  surfaceContainerHigh: '#DCE9FF',
  surfaceContainerHighest: '#D3E4FE',
  surfaceVariant: '#D3E4FE',
  onSurfaceVariant: '#434655',
  inverseSurface: '#213145',
  inverseOnSurface: '#EAF1FF',
  surfaceTint: '#0053DB',

  // Outline
  outline: '#737686',
  outlineVariant: '#C3C6D7',

  // Error
  error: '#BA1A1A',
  onError: '#FFFFFF',
  errorContainer: '#FFDAD6',
  onErrorContainer: '#93000A',
};

export const FONTS = {
  headline: 'Plus Jakarta Sans',
  body: 'Be Vietnam Pro',
  label: 'Inter',
};

export const TYPOGRAPHY = {
  headlineLg: { fontSize: 30, lineHeight: 38, fontWeight: '700', letterSpacing: -0.6 },
  headlineMd: { fontSize: 24, lineHeight: 32, fontWeight: '700', letterSpacing: -0.24 },
  headlineSm: { fontSize: 20, lineHeight: 28, fontWeight: '600', letterSpacing: 0 },
  bodyLg: { fontSize: 18, lineHeight: 28, fontWeight: '400' },
  bodyMd: { fontSize: 16, lineHeight: 24, fontWeight: '400' },
  bodySm: { fontSize: 14, lineHeight: 20, fontWeight: '400' },
  labelMd: { fontSize: 13, lineHeight: 18, fontWeight: '600', letterSpacing: 0.13 },
  labelSm: { fontSize: 11, lineHeight: 16, fontWeight: '500', letterSpacing: 0.33 },
  translationText: { fontSize: 14, lineHeight: 20, fontWeight: '500' },
} as const;

export const SPACING = {
  unit: 4,
  stackSm: 8,
  stackMd: 16,
  stackLg: 24,
  marginMobile: 16,
  gutterMobile: 12,
};

export const RADIUS = {
  sm: 4,
  md: 8,
  lg: 12,
  xl: 16,
  full: 9999,
};

export const SHADOWS = {
  elevation1: {
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.05,
    shadowRadius: 12,
    elevation: 2,
  },
  elevation2: {
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 8 },
    shadowOpacity: 0.1,
    shadowRadius: 24,
    elevation: 4,
  },
  insightGlow: {
    shadowColor: '#6B38D4',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.1,
    shadowRadius: 12,
    elevation: 2,
  },
};
