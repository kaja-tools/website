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
  { id: "scripts", label: "Scripts" },
  { id: "variables", label: "Variables" },
  { id: "deeplinks", label: "Deeplinks" },
  { id: "agents", label: "Agents" },
];
