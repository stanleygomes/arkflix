/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        apple: {
          bg: "var(--apple-bg)",
          card: "var(--apple-card)",
          glass: "var(--apple-glass)",
          glassBorder: "var(--apple-glass-border)",
          glassHover: "var(--apple-glass-hover)",
          subtext: "var(--apple-subtext)",
          text: "var(--apple-text)",
          blue: "#0071E3",
          blueHover: "#0077ED",
          accent: "#2997FF",
        }
      },
      fontFamily: {
        sans: [
          'Inter',
          '-apple-system',
          'BlinkMacSystemFont',
          '"SF Pro Display"',
          '"SF Pro Text"',
          'Helvetica Neue',
          'sans-serif'
        ],
        brand: [
          'Syne',
          'Montserrat',
          'sans-serif'
        ]
      },
      borderRadius: {
        'squircle-sm': '10px',
        'squircle': '16px',
        'squircle-lg': '22px',
        'squircle-xl': '28px',
        'squircle-2xl': '36px',
      },
      backdropBlur: {
        'glass': '20px',
        'glass-heavy': '40px',
      },
      boxShadow: {
        'glass': '0 8px 32px 0 rgba(0, 0, 0, 0.15)',
        'apple': '0 20px 40px rgba(0, 0, 0, 0.15)',
        'apple-focus': '0 0 0 4px rgba(0, 113, 227, 0.3)',
      }
    },
  },
  plugins: [],
}
