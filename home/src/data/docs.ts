/* The docs page is one column of sections with a nav to its left, and both
   read from this list — the nav's links and the scroll spy that keeps them in
   step. A section is added here and its body is written in `pages/docs.astro`
   under the same `id`.

   Screenshots are not here: a figure sits in the copy under the thing it
   shows, and often inside one platform's block, so it is markup in the page
   rather than a list a column reads. */
export interface Section {
  /* The anchor, and what the nav addresses the section by. */
  id: string;
  label: string;
}

export const sections: Section[] = [
  { id: "installation", label: "Installation" },
  { id: "apps", label: "Apps" },
  { id: "first-call", label: "Run your first call" },
  { id: "scripts", label: "Writing scripts" },
  { id: "helpers", label: "Kaja helpers" },
  { id: "files", label: "Files and drafts" },
  { id: "variables", label: "Variables" },
  { id: "deeplinks", label: "Deeplinks" },
  { id: "agents", label: "Agents" },
];

/* The two builds the docs are written for. Each is its own page —
   `/docs/desktop/` and `/docs/docker/` — so the URL is the whole of the
   state: a link says which build it was read for, and there is nothing to
   restore from a browser that a reader elsewhere does not have.

   Desktop first, and the one `/docs` resolves to, because the app does
   everything in the window; Docker is the variant, set up from mounted
   files. */
export interface Build {
  id: Platform;
  label: string;
  /* The page's own title and description: two pages indexed separately want
     to say which build each one is about. */
  title: string;
  description: string;
}

export type Platform = "desktop" | "docker";

export const builds: Build[] = [
  {
    id: "desktop",
    label: "Desktop",
    title: "Desktop docs - Kaja",
    description:
      "Set up Kaja on macOS: connect gRPC, OpenAPI, Twirp and MCP apps from the sidebar, run scripts against them, keep secrets in the Keychain, and point an agent at it over MCP.",
  },
  {
    id: "docker",
    label: "Docker",
    title: "Docker docs - Kaja",
    description:
      "Set up Kaja in Docker: connect gRPC, OpenAPI, Twirp and MCP apps from kaja.json, run scripts against them, keep secrets out of the file, and point an agent at it over MCP.",
  },
];

/* Which build a page is for, read off its own URL. A block asks for this
   rather than being handed it, so the platform is not threaded as a prop
   through every level of the page. */
export function platformOf(url: URL): Platform {
  return url.pathname.startsWith("/docs/docker") ? "docker" : "desktop";
}
