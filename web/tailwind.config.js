/** @type {import('tailwindcss').Config} */
// Phase F (issue #6) — every value below is a direct translation of
// design.md's locked tokens (Typography / Color Palette / Spacing & Shape)
// into the Tailwind theme. Colors reference the CSS custom properties
// defined in src/index.css (single source of truth); nothing here is a
// hardcoded hex. No token exists here that design.md does not define.
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        bg: 'var(--color-bg)',
        surface: 'var(--color-surface)',
        'surface-2': 'var(--color-surface-2)',
        raised: 'var(--color-raised)',
        text: 'var(--color-text)',
        'text-dim': 'var(--color-text-dim)',
        'text-bright': 'var(--color-text-bright)',
        primary: 'var(--color-primary)',
        'primary-dim': 'var(--color-primary-dim)',
        accent: 'var(--color-accent)',
        border: 'var(--color-border)',
        'border-hi': 'var(--color-border-hi)',
        success: 'var(--color-success)',
        warning: 'var(--color-warning)',
        error: 'var(--color-error)',
        'status-inbox': 'var(--color-status-inbox)',
        'status-reading': 'var(--color-status-reading)',
        'status-done': 'var(--color-status-done)',
      },
      fontFamily: {
        display: ['Fraunces', 'serif'],
        sans: ['Hanken Grotesk', 'system-ui', 'sans-serif'],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', '"Cascadia Mono"', 'monospace'],
      },
      // Type scale (base 16px) — design.md "Typography".
      fontSize: {
        xs: '12px',
        sm: '14px',
        base: '16px',
        lg: '18px',
        xl: '22px',
        '2xl': '28px',
        '3xl': '36px',
      },
      // Border radius — design.md "Spacing & Shape". Named to match
      // design.md's own scale (sm/md/lg/xl/full), not Tailwind's defaults.
      borderRadius: {
        sm: '4px',
        md: '8px',
        lg: '12px',
        xl: '16px',
        full: '9999px',
      },
      boxShadow: {
        drag: 'var(--shadow-drag)',
        modal: 'var(--shadow-modal)',
      },
      maxWidth: {
        board: '1120px',
      },
      // design.md "Layout Rules": stack breakpoint ~720px, above which the
      // board shows three equal columns side by side.
      screens: {
        board: '721px',
      },
    },
  },
  plugins: [],
};
