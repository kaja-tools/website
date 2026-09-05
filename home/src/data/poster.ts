/* The twelve statements the home page makes below the hero, each beside the
   piece of the app it is about.

   A crop is a region of one of the screenshots below, in fractions of its
   width and height, drawn as a background at `100 / w` of the frame's width,
   which is what makes the region fill the frame. `Poster.astro` does that
   arithmetic, so a crop is four numbers rather than an image file of its own
   and a fresh screenshot at the same size costs nothing but the numbers.

   Every shot is the window at 2880x1800, which is what `scripts/demo`
   photographs in kaja's own repository. `app-hero.png` is the one the page
   already shows whole under the drawing, so the two crops taken from it add
   no weight.

   `highlight` is the red box drawn on the crop, in fractions of the crop
   rather than of the shot, because it is placed against what the crop shows.
   Motion.astro draws it, and the line from the statement to it, the first time
   the crop is halfway up the viewport.

   `drift` is the share of the scroll past the item that each half moves by.
   The two signs are always opposite, so the statement and its crop pull apart
   as the item goes by. */
export interface Shot {
  src: string;
  width: number;
  height: number;
}

export const shots = {
  run: { src: "/assets/app-hero.png", width: 2880, height: 1800 },
  newApp: { src: "/assets/poster-new-app.png", width: 2880, height: 1800 },
  apps: { src: "/assets/poster-apps.png", width: 2880, height: 1800 },
  draft: { src: "/assets/poster-draft.png", width: 2880, height: 1800 },
  canvas: { src: "/assets/poster-canvas.png", width: 2880, height: 1800 },
  stats: { src: "/assets/poster-stats.png", width: 2880, height: 1800 },
  variables: { src: "/assets/poster-variables.png", width: 2880, height: 1800 },
  agent: { src: "/assets/poster-agent.png", width: 2880, height: 1800 },
} satisfies Record<string, Shot>;

export interface Item {
  caption: string;
  /* The screenshot the crop is taken from. */
  shot: Shot;
  /* The crop, as fractions of the shot: left, top, width, height. */
  crop: { x: number; y: number; w: number; h: number };
  /* The highlight box, as fractions of the crop: left, top, width, height. */
  highlight: { x: number; y: number; w: number; h: number };
  /* Where the two halves sit in the twelve-column grid, from `md` up. Below
     that the item is a plain stack and neither class applies. */
  cropCell: string;
  statementCell: string;
  /* Which end of the statement the connector leaves from, and how the
     statement is set. Right only where the crop is under it and to the right. */
  align: "left" | "right";
  drift: { crop: number; statement: number };
}

export const items: Item[] = [
  {
    caption: "An app is proto files, an OpenAPI document, or server reflection.",
    shot: shots.newApp,
    crop: { x: 0.33, y: 0.32, w: 0.34, h: 0.37 },
    highlight: { x: 0.081, y: 0.406, w: 0.845, h: 0.123 },
    cropCell: "md:col-start-1 md:col-end-6 md:row-start-1",
    statementCell: "md:col-start-7 md:col-end-13 md:row-start-1",
    align: "left",
    drift: { crop: 0.35, statement: -0.3 },
  },
  {
    caption: "Every app, service and method in one tree.",
    shot: shots.apps,
    crop: { x: 0.005, y: 0.28, w: 0.163, h: 0.368 },
    highlight: { x: 0.006, y: 0.337, w: 0.745, h: 0.337 },
    cropCell: "md:col-start-8 md:col-end-13 md:row-start-1",
    statementCell: "md:col-start-1 md:col-end-7 md:row-start-1",
    align: "left",
    drift: { crop: -0.4, statement: 0.25 },
  },
  {
    caption: "Pick a method and Kaja writes the call, typed.",
    shot: shots.draft,
    crop: { x: 0.17, y: 0.05, w: 0.45, h: 0.1 },
    highlight: { x: 0.083, y: 0.492, w: 0.739, h: 0.232 },
    cropCell: "md:col-start-1 md:col-end-13 md:row-start-2",
    statementCell: "md:col-start-1 md:col-end-9 md:row-start-1",
    align: "left",
    drift: { crop: 0.2, statement: -0.2 },
  },
  {
    caption: "A script you haven’t named is a draft. Saving it gives it a file.",
    shot: shots.draft,
    crop: { x: 0.005, y: 0.045, w: 0.163, h: 0.255 },
    highlight: { x: 0.03, y: 0.112, w: 0.9, h: 0.188 },
    cropCell: "md:col-start-1 md:col-end-6 md:row-start-1",
    statementCell: "md:col-start-7 md:col-end-13 md:row-start-1",
    align: "left",
    drift: { crop: 0.3, statement: -0.3 },
  },
  {
    caption: "Status, duration and size on every call.",
    shot: shots.run,
    crop: { x: 0.17, y: 0.18, w: 0.82, h: 0.1 },
    highlight: { x: 0.802, y: 0.688, w: 0.122, h: 0.224 },
    cropCell: "md:col-start-1 md:col-end-13 md:row-start-1",
    statementCell: "md:col-start-4 md:col-end-13 md:row-start-2",
    align: "right",
    drift: { crop: -0.2, statement: 0.3 },
  },
  {
    caption: "The response, as it came back.",
    shot: shots.run,
    crop: { x: 0.17, y: 0.28, w: 0.34, h: 0.24 },
    highlight: { x: 0.118, y: 0.273, w: 0.559, h: 0.68 },
    cropCell: "md:col-start-7 md:col-end-13 md:row-start-1",
    statementCell: "md:col-start-1 md:col-end-6 md:row-start-1",
    align: "left",
    drift: { crop: -0.35, statement: 0.2 },
  },
  {
    caption: "A script can draw a table instead. It pages the rows itself.",
    shot: shots.canvas,
    crop: { x: 0.02, y: 0.06, w: 0.96, h: 0.16 },
    highlight: { x: 0.859, y: 0.105, w: 0.135, h: 0.205 },
    cropCell: "md:col-start-1 md:col-end-13 md:row-start-2",
    statementCell: "md:col-start-5 md:col-end-13 md:row-start-1",
    align: "right",
    drift: { crop: 0.25, statement: -0.25 },
  },
  {
    caption: "Percentiles off the calls the run already made.",
    shot: shots.stats,
    crop: { x: 0.0, y: 0.005, w: 0.52, h: 0.1 },
    highlight: { x: 0.385, y: 0.414, w: 0.385, h: 0.512 },
    cropCell: "md:col-start-1 md:col-end-13 md:row-start-1",
    statementCell: "md:col-start-5 md:col-end-13 md:row-start-2",
    align: "right",
    drift: { crop: -0.25, statement: 0.2 },
  },
  {
    caption: "A perf test runs a schedule, and the bands say which phase a call landed in.",
    shot: shots.stats,
    crop: { x: 0.005, y: 0.11, w: 0.99, h: 0.34 },
    highlight: { x: 0.173, y: 0.076, w: 0.33, h: 0.334 },
    cropCell: "md:col-start-1 md:col-end-13 md:row-start-2",
    statementCell: "md:col-start-1 md:col-end-9 md:row-start-1",
    align: "left",
    drift: { crop: 0.2, statement: -0.3 },
  },
  {
    caption: "A value sits in the file, the keychain or the environment.",
    shot: shots.variables,
    crop: { x: 0.175, y: 0.06, w: 0.545, h: 0.24 },
    highlight: { x: 0.237, y: 0.143, w: 0.141, h: 0.78 },
    cropCell: "md:col-start-1 md:col-end-7 md:row-start-1",
    statementCell: "md:col-start-8 md:col-end-13 md:row-start-1",
    align: "left",
    drift: { crop: 0.35, statement: -0.3 },
  },
  {
    caption: "An endpoint and a token. The switch decides whether anything answers.",
    shot: shots.agent,
    crop: { x: 0.175, y: 0.06, w: 0.82, h: 0.115 },
    highlight: { x: 0.221, y: 0.57, w: 0.331, h: 0.237 },
    cropCell: "md:col-start-1 md:col-end-13 md:row-start-1",
    statementCell: "md:col-start-4 md:col-end-13 md:row-start-2",
    align: "right",
    drift: { crop: -0.2, statement: 0.25 },
  },
  {
    caption: "Ten agents, each with the line it takes.",
    shot: shots.agent,
    crop: { x: 0.178, y: 0.18, w: 0.135, h: 0.33 },
    highlight: { x: 0.037, y: 0.109, w: 0.881, h: 0.868 },
    cropCell: "md:col-start-9 md:col-end-13 md:row-start-1",
    statementCell: "md:col-start-1 md:col-end-8 md:row-start-1",
    align: "left",
    drift: { crop: -0.3, statement: 0.25 },
  },
];
