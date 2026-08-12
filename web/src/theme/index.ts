import { createTheme } from '@mui/material/styles'
import { components } from './components'
import {
  ESTRAL_DARK,
  ESTRAL_LIGHT,
  IP_RAMP_DARK,
  IP_RAMP_LIGHT,
  monoFont,
  rampToPalette,
  type EstralRamp,
} from './tokens'

export { IP_RAMP_SIZE } from './tokens'

declare module '@mui/material/styles' {
  interface Palette {
    estral: EstralRamp
    ipRamp: Record<string, string>
  }
  interface PaletteOptions {
    estral?: EstralRamp
    ipRamp?: Record<string, string>
  }
}

declare module '@mui/material/Chip' {
  interface ChipPropsVariantOverrides {
    glitch: true
  }
}

declare module '@mui/material/Alert' {
  interface AlertPropsVariantOverrides {
    verdict: true
  }
}

/**
 * Semantic mapping. Read this before changing a color, because two of these
 * are not what MUI's names imply:
 *
 *   success   Readout Cyan, not green. There is no green in this palette. An
 *             ALLOWED request reports nominal state, which is what cyan is
 *             reserved for. Anyone reading `color="success"` and expecting
 *             green will still be surprised.
 *   warning   Hot Trace. Covers the xff-trust-all chip and per-step errors and
 *             nothing else. A rate-limit block is a fault, so it uses `error`.
 *   info      Daemon Violet. The replay banner is the app describing itself,
 *             which is chrome rather than an alert.
 *
 * Gold is deliberately absent. It is not a semantic color. It appears exactly
 * once per screen, through `<Alert variant="verdict">`.
 *
 * `light`, `dark` and `contrastText` are pinned on every entry instead of
 * being left to `augmentColor`. MUI derives Alert backgrounds from
 * `palette.*.light`, and the sRGB interpolation it uses at `tonalOffset: 0.2`
 * drifts the hue off the ramp.
 */
function palette(
  ramp: EstralRamp,
  shades: Record<string, [string, string]>,
  contrast: string,
  action: Record<string, string>,
) {
  const on = (main: string, key: string) => ({
    main,
    light: shades[key][0],
    dark: shades[key][1],
    contrastText: contrast,
  })

  return {
    primary: on(ramp.violet, 'violet'),
    secondary: on(ramp.magenta, 'magenta'),
    success: on(ramp.cyan, 'cyan'),
    error: on(ramp.error, 'error'),
    warning: on(ramp.hot, 'hot'),
    info: on(ramp.violet, 'violet'),

    background: { default: ramp.void, paper: ramp.substrate },
    text: { primary: ramp.bone, secondary: ramp.ghost, disabled: ramp.dim },
    divider: ramp.hairline,

    /**
     * MUI's defaults here are black and white washes, `rgba(0,0,0,0.04)` and
     * similar. Left alone they put a warm neutral gray on every menu item,
     * list row, icon button and disabled control, which the palette rules out.
     * Re-derived from violet.
     */
    action,

    estral: ramp,
    ipRamp: rampToPalette(ramp === ESTRAL_DARK ? IP_RAMP_DARK : IP_RAMP_LIGHT),
  }
}

const DARK_ACTION = {
  active: ESTRAL_DARK.ghost,
  hover: 'rgba(183, 140, 255, 0.10)',
  selected: 'rgba(183, 140, 255, 0.16)',
  focus: 'rgba(183, 140, 255, 0.14)',
  disabled: ESTRAL_DARK.dim,
  disabledBackground: 'rgba(183, 140, 255, 0.12)',
}

const LIGHT_ACTION = {
  active: ESTRAL_LIGHT.ghost,
  hover: 'rgba(91, 47, 191, 0.06)',
  selected: 'rgba(91, 47, 191, 0.11)',
  focus: 'rgba(91, 47, 191, 0.10)',
  disabled: ESTRAL_LIGHT.dim,
  disabledBackground: 'rgba(91, 47, 191, 0.10)',
}

const DARK_SHADES: Record<string, [string, string]> = {
  violet: ['#D4BCFF', '#8B5CF0'],
  magenta: ['#F2A3FF', '#C42BE0'],
  cyan: ['#A8EEFF', '#33C4E8'],
  error: ['#FF7D9C', '#D91847'],
  hot: ['#FF9BE2', '#E02FAF'],
}

const LIGHT_SHADES: Record<string, [string, string]> = {
  violet: ['#8C68E2', '#4E28A8'],
  magenta: ['#B84FD6', '#781594'],
  cyan: ['#2A8CAB', '#054E66'],
  error: ['#E03A64', '#96032D'],
  hot: ['#D338AC', '#8C0669'],
}

export const theme = createTheme({
  /**
   * The CSS-variables color scheme API switches the theme without a re-render
   * and without flashing the wrong palette on first paint.
   */
  cssVariables: { colorSchemeSelector: 'class' },

  /**
   * Without this, MUI resolves the default to `light` whenever a light scheme
   * exists. That would put light values on `:root` and flash a white viewport
   * on every cold load of a near-black app. The pre-hydration script in
   * index.html covers the case where the user has already chosen a scheme.
   */
  defaultColorScheme: 'dark',

  colorSchemes: {
    // contrastText sits on saturated fills. `#0A0711` rather than MUI's default
    // `rgba(0,0,0,0.87)`: a true black is the one neutral this palette rules
    // out, and void reads 5.8:1 or better on every accent above.
    dark: { palette: palette(ESTRAL_DARK, DARK_SHADES, ESTRAL_DARK.void, DARK_ACTION) },
    light: { palette: palette(ESTRAL_LIGHT, LIGHT_SHADES, ESTRAL_LIGHT.void, LIGHT_ACTION) },
  },

  // Nothing in this system is rounded.
  shape: { borderRadius: 0 },

  typography: {
    fontFamily: monoFont,
    // Only 400 and 600 are loaded. Left at the default 700, CssBaseline's
    // `strong, b` rule synthesizes a faux-bold, which on a monospace breaks
    // the fixed advance width and smears.
    fontWeightBold: 600,
    fontWeightMedium: 600,

    h1: { fontSize: '1.5rem', fontWeight: 600, letterSpacing: '.12em', textTransform: 'uppercase' },
    h2: { fontSize: '1.2rem', fontWeight: 600, letterSpacing: '.08em' },
    h3: { fontSize: '1rem', fontWeight: 600, letterSpacing: '.1em', textTransform: 'uppercase' },
    overline: { letterSpacing: '.14em', fontWeight: 600 },

    // Letterspacing belongs to the uppercase labels only. On a monospace it
    // destroys the column rhythm that makes the face worth using.
    body1: { lineHeight: 1.65, letterSpacing: 0 },
    body2: { lineHeight: 1.6, letterSpacing: 0 },
  },

  components,
})
