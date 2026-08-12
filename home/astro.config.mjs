// @ts-check
import { defineConfig } from "astro/config";
import tailwindcss from "@tailwindcss/vite";

// The website is a handful of pages with nothing to render per request, so it
// builds to plain files and a file server ships them. No adapter, no Node in
// the runtime image — see Dockerfile.
export default defineConfig({
  site: "https://kaja.tools",
  output: "static",

  // Emit `privacy/index.html` rather than `privacy.html`, so `/privacy`
  // resolves as a directory index on any static server. This is what replaces
  // the extension-guessing the old Go server did by hand.
  build: {
    format: "directory",
  },

  vite: {
    plugins: [tailwindcss()],
  },
});
