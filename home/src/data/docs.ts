/* The docs page is one column of sections with a nav to its left and a
   screenshot to its right, and all three read from this list — the nav's
   links, the rail's figures, and the scroll spy that keeps them in step. A
   section is added here and its body is written in `pages/docs.astro` under
   the same `id`.

   A section without a `shot` leaves the rail empty while it is being read.
   That is the honest state and not a gap to fill with a picture of something
   else: the rail is a decorative column, and the two sections that have no
   shot yet — Variables and Agents — are the two whose screens nothing in
   `public/assets/` has been taken of. Drop one in and name it here. */

export interface Section {
  /* The anchor, and what the nav and the rail address the section by. */
  id: string;
  label: string;
  /* The shot the rail holds while this section is being read, and the line
     under it. Its natural size is carried so the rail doesn't reflow as the
     image arrives; the shots differ wildly in shape, being crops the home
     page took for its own layout. */
  shot?: { src: string; width: number; height: number; caption: string };
}

export const sections: Section[] = [
  {
    id: "installation",
    label: "Installation",
    shot: {
      src: "/assets/app-hero.png",
      width: 3104,
      height: 2024,
      caption: "Kaja, with a script open and the run it made below it.",
    },
  },
  {
    id: "apps",
    label: "Apps",
    shot: {
      src: "/assets/shot-apps.png",
      width: 698,
      height: 961,
      caption: "One tree for every app you connected.",
    },
  },
  { id: "variables", label: "Variables" },
  {
    id: "scripts",
    label: "Scripts",
    shot: {
      src: "/assets/shot-scripts.png",
      width: 698,
      height: 516,
      caption: "Scripts and drafts share the sidebar.",
    },
  },
  { id: "agents", label: "Agents" },
  {
    id: "concepts",
    label: "Core concepts",
    shot: {
      src: "/assets/shot-response.png",
      width: 1226,
      height: 304,
      caption: "Request, response and headers, for every call in a run.",
    },
  },
];
