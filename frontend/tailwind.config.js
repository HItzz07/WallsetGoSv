// import type { Config } from "tailwindcss";

// export default {
//   content: ["./src/**/*.{html,js,svelte,ts}"],

//   theme: {
//     extend: {}
//   },

//   plugins: [require("@tailwindcss/typography"),
//     require("daisyui")]
// } as Config;

/** @type {import('tailwindcss').Config} */
export default {
  content: ["./src/**/*.{html,js,svelte,ts}"],

  theme: {
    extend: {}
  },

  plugins: [
    require("@tailwindcss/typography"),
    require("daisyui")
  ]
};
