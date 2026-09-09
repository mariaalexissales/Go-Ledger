export interface EstralRamp {
  void: string
  substrate: string

  violet: string
  magenta: string
  hot: string
  cyan: string
  gold: string
  error: string

  bone: string
  ghost: string
  dim: string

  glitchR: string
  glitchC: string

  hairline: string
  hairlineStrong: string
  blockedRow: string

  hoverRow: string

  texture: string
  texturePeriod: string
  glow1: string
  glow2: string
}

export const ESTRAL_DARK: EstralRamp = {
  void: '#0A0711',
  substrate: '#150E24',

  violet: '#B78CFF',
  magenta: '#E85FFF',
  hot: '#FF5FD2',
  cyan: '#6FE3FF',
  gold: '#E8C46A',
  error: '#FF3B6B',

  bone: '#E9E3F5',
  ghost: '#8478A6',
  dim: '#7A6E9E',

  glitchR: '#FF0040',
  glitchC: '#00FFE1',

  hairline: 'rgba(183, 140, 255, 0.18)',
  hairlineStrong: 'rgba(183, 140, 255, 0.38)',
  blockedRow: 'rgba(255, 59, 107, 0.14)',

  hoverRow: '#1E1433',

  texture: 'rgba(0, 0, 0, 0.22)',
  texturePeriod: '3px',
  glow1: 'rgba(183, 140, 255, 0.10)',
  glow2: 'rgba(232, 95, 255, 0.08)',
}

export const ESTRAL_LIGHT: EstralRamp = {
  void: '#F4F1FA',
  substrate: '#EAE4F5',

  violet: '#5B2FBF',
  magenta: '#A81FB0',
  hot: '#B31A9B',
  cyan: '#0E6E8C',
  gold: '#825D0C',
  error: '#C41442',

  bone: '#1A1226',
  ghost: '#655A85',
  dim: '#8F86A8',

  glitchR: '#E0004A',
  glitchC: '#0090C0',

  hairline: 'rgba(91, 47, 191, 0.24)',
  hairlineStrong: 'rgba(91, 47, 191, 0.45)',
  blockedRow: 'rgba(196, 20, 66, 0.10)',

  hoverRow: '#F4F1FA',

  texture: 'rgba(91, 47, 191, 0.05)',
  texturePeriod: '4px',
  glow1: 'rgba(91, 47, 191, 0.07)',
  glow2: 'rgba(168, 31, 176, 0.05)',
}

export const IP_RAMP_SIZE = 24

export const IP_RAMP_DARK: readonly string[] = [
  '#8199F8',
  '#8185F8',
  '#9181F8',
  '#A381F8',
  '#B781F8',
  '#CB81F8',
  '#DF81F8',
  '#F281F8',
  '#F881EA',
  '#F881D9',
  '#F881C5',
  '#F881B1',
  '#AFBFFD',
  '#AFB2FD',
  '#B9AFFD',
  '#C5AFFD',
  '#D2AFFD',
  '#DFAFFD',
  '#ECAFFD',
  '#F9AFFD',
  '#FDAFF4',
  '#FDAFE8',
  '#FDAFDB',
  '#FDAFCE',
]

export const IP_RAMP_LIGHT: readonly string[] = [
  '#2A43A7',
  '#2A2EA7',
  '#3B2AA7',
  '#4D2AA7',
  '#622AA7',
  '#772AA7',
  '#8C2AA7',
  '#A12AA7',
  '#A72A99',
  '#A72A86',
  '#A72A71',
  '#A72A5C',
  '#182F8B',
  '#181C8B',
  '#28188B',
  '#39188B',
  '#4C188B',
  '#5F188B',
  '#72188B',
  '#85188B',
  '#8B187D',
  '#8B186C',
  '#8B1859',
  '#8B1846',
]

export function rampToPalette(ramp: readonly string[]): Record<string, string> {
  return Object.fromEntries(ramp.map((hex, i) => [String(i), hex]))
}

export const monoFont =
  "'IBM Plex Mono', ui-monospace, SFMono-Regular, Menlo, Consolas, 'Liberation Mono', monospace"
