import type { Components, Theme } from '@mui/material/styles'
import { monoFont } from './tokens'

/**
 * Component overrides. Styling belongs here rather than in `sx` props on route
 * files, which should only reach for semantic colors.
 *
 * Note that MUI's `shape.borderRadius` does not reach every component. Chip,
 * Alert, Tooltip, Skeleton and LinearProgress hardcode their own radius and
 * each need squaring off by hand below.
 */

/**
 * `vars` is optional on the base Theme type, since a theme can be built without
 * the CSS-variables API. This one always has it, set by `cssVariables` in
 * ./index.ts. Reading `theme.palette` instead would bake in the default
 * scheme's literal values and break the light/dark toggle.
 */
const v = (theme: Omit<Theme, 'components'>) => theme.vars!

/** Uppercase, letterspaced, small. The label voice used throughout. */
const LABEL = {
  textTransform: 'uppercase',
  letterSpacing: '.08em',
  fontWeight: 600,
} as const

/**
 * The chevron marker from the palette, drawn as geometry rather than text.
 *
 * Screen readers announce pseudo-element text content, so `content: '❯'` would
 * have every card header read out as "greater-than sign".
 */
const CHEVRON = {
  content: '""',
  display: 'inline-block',
  width: '0.42em',
  height: '0.66em',
  marginRight: '0.55em',
  backgroundColor: 'var(--mui-palette-primary-main)',
  clipPath: 'polygon(0 0, 100% 50%, 0 100%, 32% 50%)',
} as const

export const components: Components<Omit<Theme, 'components'>> = {
  MuiCssBaseline: {
    styleOverrides: (theme) => ({
      /**
       * The page ground: scanline texture plus the two radial washes.
       *
       * These are background layers, not an overlay. Cards and panels paint
       * opaque over them, so the texture shows in the ground and the gutters
       * and never touches text or the chart. `background-attachment: fixed`
       * keeps the washes anchored to the viewport instead of sliding down a
       * long page.
       */
      body: {
        backgroundColor: v(theme).palette.background.default,
        backgroundImage: [
          `repeating-linear-gradient(0deg,
            ${v(theme).palette.estral.texture} 0,
            ${v(theme).palette.estral.texture} 1px,
            transparent 1px,
            transparent ${v(theme).palette.estral.texturePeriod})`,
          `radial-gradient(circle at 18% 12%, ${v(theme).palette.estral.glow1}, transparent 42%)`,
          `radial-gradient(circle at 82% 78%, ${v(theme).palette.estral.glow2}, transparent 46%)`,
        ].join(', '),
        backgroundAttachment: 'fixed',
        '@media print': { backgroundImage: 'none' },
        '@media (prefers-contrast: more)': { backgroundImage: 'none' },
      },

      '::selection': {
        backgroundColor: v(theme).palette.primary.main,
        color: v(theme).palette.estral.void,
      },
      // Ghost violet, never gray.
      '*': {
        scrollbarWidth: 'thin',
        scrollbarColor: `${v(theme).palette.estral.hairlineStrong} transparent`,
      },
      // The default focus ring is a browser blue that has no business here.
      ':focus-visible': {
        outline: `1px solid ${v(theme).palette.primary.main}`,
        outlineOffset: 2,
      },
    }),
  },

  MuiPaper: {
    defaultProps: { elevation: 0, variant: 'outlined' },
    styleOverrides: {
      root: ({ theme }) => ({
        borderRadius: 0,
        borderColor: v(theme).palette.estral.hairline,
        // Paper paints `--Paper-overlay`, a white gradient, at any elevation
        // above 0. Menu, Dialog, Popover and Select all hardcode elevation 8
        // or 24, so without this the substrate washes out to gray on every
        // overlay in the app. Gray is the one thing this palette does not have.
        backgroundImage: 'none',
      }),
    },
  },

  MuiCard: {
    styleOverrides: {
      root: { position: 'relative' },
    },
  },

  MuiCardHeader: {
    styleOverrides: {
      title: {
        ...LABEL,
        fontSize: '.8125rem',
        display: 'flex',
        alignItems: 'center',
        '&::before': CHEVRON,
      },
      subheader: {
        fontSize: '.6875rem',
        letterSpacing: '.04em',
      },
    },
  },

  MuiAppBar: {
    defaultProps: { elevation: 0, color: 'transparent' },
    styleOverrides: {
      root: ({ theme }) => ({
        backgroundColor: v(theme).palette.background.paper,
        backgroundImage: 'none',
      }),
    },
  },

  MuiTabs: {
    // Uppercase plus letterspacing on a monospace runs wide. Five tabs, a
    // stream chip and a theme toggle share one toolbar, and they collide well
    // before the mobile breakpoint. Scrollable keeps that from clipping.
    defaultProps: { variant: 'scrollable', scrollButtons: 'auto' },
    styleOverrides: {
      indicator: ({ theme }) => ({
        height: 2,
        backgroundColor: v(theme).palette.secondary.main,
      }),
    },
  },

  MuiTab: {
    styleOverrides: {
      root: ({ theme }) => ({
        ...LABEL,
        // Tighter than the standard label voice: this is the widest run of
        // uppercase text in the app.
        letterSpacing: '.06em',
        fontSize: '.75rem',
        minHeight: 48,
        '&.Mui-selected': { color: v(theme).palette.secondary.main },
      }),
    },
  },

  MuiChip: {
    styleOverrides: {
      root: {
        borderRadius: 0,
        fontFamily: monoFont,
        letterSpacing: '.06em',
      },
      label: { textTransform: 'uppercase' },
    },
    variants: [
      {
        // Aberration. The transparent ground is deliberate: an RGB split over
        // a saturated fill reads as mud. It needs the dark behind it.
        props: { variant: 'glitch' },
        style: ({ theme }) => ({
          backgroundColor: 'transparent',
          color: v(theme).palette.error.main,
          border: `1px solid ${v(theme).palette.error.main}`,
          fontWeight: 600,
          textShadow: `-1px 0 ${v(theme).palette.estral.glitchR}, 1px 0 ${v(theme).palette.estral.glitchC}`,
        }),
      },
    ],
  },

  MuiButton: {
    defaultProps: { disableElevation: true },
    styleOverrides: {
      root: {
        borderRadius: 0,
        ...LABEL,
        letterSpacing: '.1em',
        fontSize: '.75rem',
      },
    },
  },

  MuiAlert: {
    defaultProps: { variant: 'outlined' },
    styleOverrides: {
      root: { borderRadius: 0 },
      message: { fontSize: '.8125rem' },
    },
    variants: [
      {
        // Gold is capped at one element per screen. No `severity` can express
        // that, since MUI has five and none of them is gold. Making it a
        // variant keeps the rule enforceable.
        props: { variant: 'verdict' },
        style: ({ theme }) => ({
          backgroundColor: 'transparent',
          color: v(theme).palette.estral.gold,
          border: `1px solid ${v(theme).palette.estral.gold}`,
          '& .MuiAlert-icon': { color: v(theme).palette.estral.gold },
        }),
      },
    ],
  },

  MuiAlertTitle: {
    styleOverrides: {
      root: { ...LABEL, fontSize: '.75rem' },
    },
  },

  MuiTableCell: {
    styleOverrides: {
      root: ({ theme }) => ({
        borderColor: v(theme).palette.estral.hairline,
        fontFamily: monoFont,
        fontVariantNumeric: 'tabular-nums',
        fontSize: '.8125rem',
      }),
      head: ({ theme }) => ({
        ...LABEL,
        fontSize: '.6875rem',
        color: v(theme).palette.text.secondary,
        backgroundColor: v(theme).palette.background.default,
      }),
    },
  },

  MuiTableRow: {
    styleOverrides: {
      root: ({ theme }) => ({
        // Not `surface`. Hover lifts in both schemes. See the note on
        // `hoverRow` in tokens.ts for why pressing down fails in light mode.
        '&:hover': { backgroundColor: v(theme).palette.estral.hoverRow },
      }),
    },
  },

  MuiTablePagination: {
    styleOverrides: {
      selectLabel: { ...LABEL, fontSize: '.6875rem' },
      displayedRows: { fontFamily: monoFont, fontSize: '.75rem' },
    },
  },

  MuiOutlinedInput: {
    styleOverrides: {
      root: { borderRadius: 0 },
      notchedOutline: ({ theme }) => ({
        borderColor: v(theme).palette.estral.hairlineStrong,
      }),
      input: { fontFamily: monoFont },
    },
  },

  MuiInputLabel: {
    styleOverrides: {
      root: { ...LABEL, fontSize: '.75rem' },
    },
  },

  MuiTooltip: {
    styleOverrides: {
      tooltip: ({ theme }) => ({
        // The default is grey[700] at 92%. Warm-free, but unmistakably gray,
        // and it would be the only gray surface in the app.
        backgroundColor: v(theme).palette.estral.substrate,
        border: `1px solid ${v(theme).palette.primary.main}`,
        borderRadius: 0,
        fontFamily: monoFont,
        fontSize: '.6875rem',
        color: v(theme).palette.text.primary,
      }),
      arrow: ({ theme }) => ({ color: v(theme).palette.primary.main }),
    },
  },

  MuiDivider: {
    styleOverrides: {
      root: ({ theme }) => ({ borderColor: v(theme).palette.estral.hairline }),
    },
  },

  MuiSkeleton: {
    styleOverrides: {
      root: { borderRadius: 0 },
    },
  },

  MuiLinearProgress: {
    styleOverrides: {
      root: { borderRadius: 0 },
      bar: { borderRadius: 0 },
    },
  },

  MuiDialog: {
    styleOverrides: {
      paper: ({ theme }) => ({
        borderRadius: 0,
        border: `1px solid ${v(theme).palette.estral.hairlineStrong}`,
        backgroundImage: 'none',
      }),
    },
  },

  MuiDialogTitle: {
    styleOverrides: {
      root: { ...LABEL, fontSize: '.875rem' },
    },
  },

  MuiLink: {
    defaultProps: { underline: 'hover' },
    styleOverrides: {
      root: ({ theme }) => ({ color: v(theme).palette.primary.main }),
    },
  },
}
