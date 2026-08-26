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
          bg: "#000000",
          card: "rgba(28, 28, 30, 0.65)",
          glass: "rgba(255, 255, 255, 0.08)",
          glassBorder: "rgba(255, 255, 255, 0.12)",
          glassHover: "rgba(255, 255, 255, 0.16)",
          subtext: "#86868B",
          text: "#F5F5F7",
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
        'glass': '0 8px 32px 0 rgba(0, 0, 0, 0.37)',
        'apple': '0 20px 40px rgba(0, 0, 0, 0.4)',
        'apple-focus': '0 0 0 4px rgba(0, 113, 227, 0.4)',
      }
    },
  },
  plugins: [],
}
