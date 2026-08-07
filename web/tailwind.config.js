/** @type {import('tailwindcss').Config} */
// Pre-Phase F scaffold (issue #6) — deliberately theme-free. Phase F reads
// design.md and adds the design-token `theme.extend` block (colors,
// typography, spacing, radius) there — do not invent tokens here. This
// file exists only so Tailwind's build pipeline (content scanning,
// PostCSS integration) is wired and working before that happens.
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {},
  },
  plugins: [],
};
