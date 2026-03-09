/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ["Inter", "system-ui", "-apple-system", "sans-serif"],
      },
      colors: {
        accent: {
          DEFAULT: '#7c6af7',
          hover: '#6c5ae0',
          light: 'rgba(124, 106, 247, 0.12)',
        },
      },
    },
  },
  plugins: [],
}