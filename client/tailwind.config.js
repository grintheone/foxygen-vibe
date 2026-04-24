/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,jsx}"],
  theme: {
    extend: {
      borderRadius: {
        "2xl": "0.5rem",
        "3xl": "0.5rem",
      },
    },
  },
  plugins: [],
};
