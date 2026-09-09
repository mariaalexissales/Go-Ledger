import type { Components, Theme } from '@mui/material/styles'

const v = (theme: Omit<Theme, 'components'>) => theme.vars!

const LABEL = {
  textTransform: 'uppercase',
  letterSpacing: '.08em',
  fontWeight: 600,
} as const

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
      '*': {
        scrollbarWidth: 'thin',
        scrollbarColor: `${v(theme).palette.estral.hairlineStrong} transparent`,
      },
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
        borderColor: v(theme).palette.estral.hairline,
        backgroundImage: 'none',
      }),
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
    styleOverrides: {
      root: ({ theme }) => ({
        backgroundColor: v(theme).palette.background.paper,
      }),
    },
  },

  MuiTabs: {
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
        letterSpacing: '.06em',
      },
      label: { textTransform: 'uppercase' },
    },
    variants: [
      {
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
        ...LABEL,
        letterSpacing: '.1em',
        fontSize: '.75rem',
      },
    },
  },

  MuiAlert: {
    defaultProps: { variant: 'outlined' },
    styleOverrides: {
      message: { fontSize: '.8125rem' },
    },
    variants: [
      {
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
        '&:hover': { backgroundColor: v(theme).palette.estral.hoverRow },
      }),
    },
  },

  MuiTablePagination: {
    styleOverrides: {
      selectLabel: { ...LABEL, fontSize: '.6875rem' },
      displayedRows: { fontSize: '.75rem' },
    },
  },

  MuiOutlinedInput: {
    styleOverrides: {
      notchedOutline: ({ theme }) => ({
        borderColor: v(theme).palette.estral.hairlineStrong,
      }),
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
        backgroundColor: v(theme).palette.estral.substrate,
        border: `1px solid ${v(theme).palette.primary.main}`,
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

  MuiDialog: {
    styleOverrides: {
      paper: ({ theme }) => ({
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
