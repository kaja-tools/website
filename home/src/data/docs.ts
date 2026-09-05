/* The docs page is one column of sections with a nav to its left and a
   screenshot to its right, and all three read from this list — the nav's
   links, the rail's figures, and the scroll spy that keeps them in step. A
   section is added here and its body is written in `pages/docs.astro` under
   the same `id`.

   A figure is a crop of one of the shots in `shots.ts`, and it may belong to
   one platform: the desktop's New app dialog is not a picture of anything the
   container does, so Docker gets the tree its configuration produces instead.
   A section with no figure for the platform being read leaves the rail empty.
   That is the honest state and not a gap to fill with a picture of something
   else — the rail is a decorative column.

   A crop is read at the rail's width, which is about a seventh of the shot's,
   so a region wider than a thousand pixels of the shot is a picture nobody can
   read the words in. Crop tight. */
import { shots, type Crop, type Shot } from "./shots";

export type Platform = "desktop" | "docker";

export interface Figure {
  shot: Shot;
  /* The whole shot when absent. */
  crop?: Crop;
  caption: string;
  /* Both platforms when absent. */
  platform?: Platform;
}

export interface Section {
  /* The anchor, and what the nav and the rail address the section by. */
  id: string;
  label: string;
  figures?: Figure[];
}

export const sections: Section[] = [
  {
    id: "installation",
    label: "Installation",
    figures: [{ shot: shots.run, caption: "Kaja, with a script open and the run it made below it." }],
  },
  {
    id: "apps",
    label: "Apps",
    figures: [
      {
        shot: shots.newApp,
        crop: { x: 0.34, y: 0.325, w: 0.32, h: 0.35 },
        caption: "The + beside Apps. Each type asks for one thing.",
        platform: "desktop",
      },
      {
        shot: shots.apps,
        crop: { x: 0.0, y: 0.255, w: 0.168, h: 0.405 },
        caption: "What the configuration file produces: every app, service and method.",
        platform: "docker",
      },
    ],
  },
  {
    id: "scripts",
    label: "Scripts",
    figures: [
      {
        shot: shots.run,
        crop: { x: 0.168, y: 0.045, w: 0.3, h: 0.36 },
        caption: "The drafted call, and the run it made: status, duration, and the response as it came back.",
      },
    ],
  },
  {
    id: "variables",
    label: "Variables",
    figures: [
      {
        shot: shots.variables,
        crop: { x: 0.175, y: 0.055, w: 0.33, h: 0.27 },
        caption: "Each row says where its value lives.",
        platform: "desktop",
      },
    ],
  },
  { id: "deeplinks", label: "Deeplinks" },
  {
    id: "agents",
    label: "Agents",
    figures: [
      {
        shot: shots.agent,
        crop: { x: 0.178, y: 0.155, w: 0.33, h: 0.29 },
        caption: "Pick the agent and copy what it takes.",
        platform: "desktop",
      },
    ],
  },
];
