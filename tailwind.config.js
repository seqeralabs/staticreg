/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["pkg/templates/tmpl/*.html", "pkg/templates/tmpl/partials/*.html", "pkg/templates/tmpl/partials/icons/*.html"],
  darkMode: 'class',
  theme: {
    extend: {
      screens: {
        '1030': '1030px',
      },
    },
  },
  plugins: [],
}

