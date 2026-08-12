/**
 * ESTRAL raw color ramp.
 *
 * No MUI imports, so the SVG chart and anything else outside the component
 * tree can read these directly.
 *
 * Rules the palette depends on. None of them are visible in the hex values,
 * and the system stops holding together if they drift:
 *
 *   Ratio       about 70% near-black, 20% violet, 10% everything else.
 *   Violet      all system chrome: frames, rules, headers.
 *   Cyan        readouts only. If it does not report state, it is not cyan.
 *   Gold        once per screen, maximum.
 *   Aberration  text-shadow or edge offset only. Never a fill or background.
 *   Warmth      none. Bone-white instead of white, ghost violet instead of gray.
 */

export interface EstralRamp {
  /** Base background. */
  void: string
  /** Panels, window fill. */
  substrate: string

  /** Primary. All chrome: frames, rules, headers. */
  violet: string
  /** Emphasis, active state. */
  magenta: string
  /** Alerts, anything spiking. */
  hot: string
  /** Readouts only. */
  cyan: string
  /** The one warm hue. Once per screen. */
  gold: string
  /** Faults. */
  error: string

  /** Body text. */
  bone: string
  /** Secondary body text. The palette's stand-in for gray. */
  ghost: string
  /** Disabled and decorative only. See the note on the dark ramp below. */
  dim: string

  /**
   * RGB split. Offset copies of text or edges, never a fill. Per-mode, because
   * the cyan half of a split disappears on paper, so light deepens both halves.
   */
  glitchR: string
  glitchC: string

  /** Hairline borders and dividers. */
  hairline: string
  /** Derived. Input outlines, dialog frames, scrollbar. */
  hairlineStrong: string
  /** Derived. Row tint for a blocked request. */
  blockedRow: string

  /**
   * Table-row hover.
   *
   * Light mode surfaces descend in lightness, so pressing a row down toward
   * #DDD5EE drops readout text to 4.09:1 and gold to 3.79:1. There is no room
   * to nudge it either: readout needs a background luminance of 0.766 and the
   * paper is already at 0.796. Hover lifts in both modes instead, toward the
   * raised tone in dark and toward the page ground in light.
   */
  hoverRow: string

  /**
   * Page-ground texture ink and its repeat period.
   *
   * The period carries its unit. These become CSS custom properties and get
   * interpolated into a gradient, where a bare `3` produces `transparent 1px 3`.
   * That is invalid and drops the layer with no error.
   */
  texture: string
  texturePeriod: string
  /** The two radial washes on the page background. */
  glow1: string
  glow2: string
}

/**
 * Dark mode.
 *
 * `ghost` and `dim` split one authored token in two. MUI needs secondary and
 * disabled text to differ, or a disabled control looks identical to a caption.
 * Both values come from the palette: #8478A6 is the current dim value, and
 * #7A6E9E is the one it replaced. #7A6E9E measures 4.33:1 on the background,
 * under the 4.5:1 body text needs, but fine for disabled, which WCAG 1.4.3
 * exempts.
 */
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

  // Kept as rgba rather than flattened to hex so a hairline composites over
  // whatever surface it lands on. Custom palette keys get no Channel var from
  // MUI, which rules out manipulating alpha at call sites, but not storing an
  // alpha value here.
  hairline: 'rgba(183, 140, 255, 0.18)',
  hairlineStrong: 'rgba(183, 140, 255, 0.38)',
  blockedRow: 'rgba(255, 59, 107, 0.14)',

  hoverRow: '#1E1433',

  texture: 'rgba(0, 0, 0, 0.22)',
  texturePeriod: '3px',
  glow1: 'rgba(183, 140, 255, 0.10)',
  glow2: 'rgba(232, 95, 255, 0.08)',
}

/**
 * Light mode.
 *
 * The surfaces descend in lightness here: page #F4F1FA, panels #EAE4F5. Panels
 * sit recessed into the page rather than lifted off it, which inverts the usual
 * Material stacking. That is why `hoverRow` lifts toward the page ground rather
 * than pressing further down.
 *
 * `dim` has no authored light value. It is derived by lightening `ghost`
 * toward the paper until it reads as inactive.
 */
export const ESTRAL_LIGHT: EstralRamp = {
  void: '#F4F1FA',
  substrate: '#EAE4F5',

  violet: '#5B2FBF',
  magenta: '#A81FB0',
  // Two deviations from the authored palette, both darkened at constant hue
  // (41 and 309 degrees, unchanged). The authored values clear AA against the
  // page background #F4F1FA, but most text here sits on a card, and against
  // paper #EAE4F5 the authored hot measured 4.17:1 and gold 4.32:1. Lightening
  // the paper instead would have flattened the page, panel and raised descent
  // that gives light mode its character, so darkening two accents was the
  // smaller change.
  hot: '#B31A9B', // authored #C21FA8, 4.17:1 on paper
  cyan: '#0E6E8C',
  gold: '#825D0C', // authored #8A6410, 4.32:1 on paper
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

/**
 * Per-IP identity colors.
 *
 * 12 hues from 228 to 336 degrees, blue-violet through violet and magenta to
 * hot, at two lightness tiers. Cyan's band is excluded so it stays a state
 * readout rather than becoming an identity token.
 */
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

/** Keyed form, because MUI's CSS-var parser walks objects, not arrays. */
export function rampToPalette(ramp: readonly string[]): Record<string, string> {
  return Object.fromEntries(ramp.map((hex, i) => [String(i), hex]))
}

/**
 * IBM Plex Mono carries the whole UI. Loaded in main.tsx.
 *
 * Every fallback has roughly a 0.6em advance, so a late font swap barely
 * reflows.
 */
export const monoFont =
  "'IBM Plex Mono', ui-monospace, SFMono-Regular, Menlo, Consolas, 'Liberation Mono', monospace"
