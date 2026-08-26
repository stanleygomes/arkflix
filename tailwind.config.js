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
        netflix: {
          red: "#E50914",
          black: "#141414",
          dark: "#181818",
          card: "#232323",
          gray: "#808080",
          lightGray: "#e5e5e5"
        }
      },
      fontFamily: {
        sans: [
          'Netflix Sans',
          'Helvetica Neue',
          'Segoe UI',
          'Roboto',
          'Ubuntu',
          'sans-serif'
        ]
      }
    },
  },
  plugins: [],
}
