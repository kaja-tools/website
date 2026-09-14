/* The statements the home page makes about the window, each beside the piece
   of the app it is about. They come in groups, because each one belongs to
   the section that makes its argument: what you do yourself, what stays on
   the record, what an agent connects to, and what the window does beyond one
   request. A caption is the thing and what it is good for, in one sentence:
   the crop already shows the control, so the words are the reason to care
   about it.

   A crop is a region of one of the shots in `shots.ts`, drawn by `cropStyle`
   there, so a crop is four numbers rather than an image file of its own and a
   fresh screenshot at the same size costs nothing but the numbers.
   `app-hero.png` is the one the page already shows whole under the hero, so
   the crops taken from it add no weight.

   `highlight` is the red box drawn on the crop, in fractions of the crop
   rather than of the shot, because it is placed against what the crop shows.
   Motion.astro draws it, and the line from the statement to it, the first time
   the crop is halfway up the viewport.

   `drift` is the share of the scroll past the item that each half moves by.
   The two signs are always opposite, so the statement and its crop pull apart
   as the item goes by. */
import { shots, type Crop, type Shot } from "./shots";

export interface Item {
  caption: string;
  /* The screenshot the crop is taken from. */
  shot: Shot;
  crop: Crop;
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

/* Exploring an API by hand: the tree, and the call Kaja writes for a method. */
export const manual: Item[] = [
  {
    caption: "Every app, service and method in one tree.",
    shot: shots.apps,
    crop: { x: 0.006, y: 0.35, w: 0.2, h: 0.46 },
    highlight: { x: 0.006, y: 0.352, w: 0.62, h: 0.33 },
    cropCell: "md:col-start-8 md:col-end-13 md:row-start-1",
    statementCell: "md:col-start-1 md:col-end-7 md:row-start-1",
    align: "left",
    drift: { crop: -0.4, statement: 0.25 },
  },
  {
    caption: "Pick a method and Kaja writes the call, typed.",
    shot: shots.draft,
    crop: { x: 0.212, y: 0.062, w: 0.56, h: 0.125 },
    highlight: { x: 0.085, y: 0.49, w: 0.87, h: 0.235 },
    cropCell: "md:col-start-1 md:col-end-13 md:row-start-2",
    statementCell: "md:col-start-1 md:col-end-9 md:row-start-1",
    align: "left",
    drift: { crop: 0.2, statement: -0.2 },
  },
];

/* What a call leaves behind, whoever made it. */
export const record: Item[] = [
  {
    caption: "Status, duration and size on every call.",
    shot: shots.run,
    crop: { x: 0.212, y: 0.225, w: 0.788, h: 0.145 },
    highlight: { x: 0.725, y: 0.74, w: 0.175, h: 0.2 },
    cropCell: "md:col-start-1 md:col-end-13 md:row-start-1",
    statementCell: "md:col-start-4 md:col-end-13 md:row-start-2",
    align: "right",
    drift: { crop: -0.2, statement: 0.3 },
  },
  {
    caption: "The response, as it came back.",
    shot: shots.run,
    crop: { x: 0.212, y: 0.35, w: 0.425, h: 0.32 },
    highlight: { x: 0.11, y: 0.25, w: 0.66, h: 0.7 },
    cropCell: "md:col-start-7 md:col-end-13 md:row-start-1",
    statementCell: "md:col-start-1 md:col-end-6 md:row-start-1",
    align: "left",
    drift: { crop: -0.35, statement: 0.2 },
  },
];

/* The plug at the top of the sidebar: what an agent needs, and which agents
   are listed. */
export const agents: Item[] = [
  {
    caption: "Turn the server on, then give your agent the endpoint and the token.",
    shot: shots.agent,
    crop: { x: 0.219, y: 0.075, w: 0.781, h: 0.144 },
    highlight: { x: 0.02, y: 0.57, w: 0.532, h: 0.237 },
    cropCell: "md:col-start-1 md:col-end-13 md:row-start-1",
    statementCell: "md:col-start-4 md:col-end-13 md:row-start-2",
    align: "right",
    drift: { crop: -0.2, statement: 0.25 },
  },
  {
    caption: "Pick your agent and copy the line it needs.",
    shot: shots.agent,
    crop: { x: 0.222, y: 0.225, w: 0.169, h: 0.412 },
    highlight: { x: 0.037, y: 0.109, w: 0.881, h: 0.868 },
    cropCell: "md:col-start-9 md:col-end-13 md:row-start-1",
    statementCell: "md:col-start-1 md:col-end-8 md:row-start-1",
    align: "left",
    drift: { crop: -0.3, statement: 0.25 },
  },
];

/* Everything a run is beyond one request: a file, a table, a load test, and
   the values a script should not carry. */
export const features: Item[] = [
  {
    caption: "Try a script as a draft, then save it to a file when it works.",
    shot: shots.draft,
    crop: { x: 0.006, y: 0.056, w: 0.2, h: 0.319 },
    highlight: { x: 0.03, y: 0.112, w: 0.9, h: 0.188 },
    cropCell: "md:col-start-1 md:col-end-6 md:row-start-1",
    statementCell: "md:col-start-7 md:col-end-13 md:row-start-1",
    align: "left",
    drift: { crop: 0.3, statement: -0.3 },
  },
  {
    caption: "Draw the results as a table, and the script pages the rows itself.",
    shot: shots.canvas,
    crop: { x: 0.02, y: 0.075, w: 0.96, h: 0.2 },
    highlight: { x: 0.855, y: 0.06, w: 0.14, h: 0.18 },
    cropCell: "md:col-start-1 md:col-end-13 md:row-start-2",
    statementCell: "md:col-start-5 md:col-end-13 md:row-start-1",
    align: "right",
    drift: { crop: 0.25, statement: -0.25 },
  },
  {
    caption: "Percentiles computed from the calls the run already made.",
    shot: shots.stats,
    crop: { x: 0.0, y: 0.053, w: 0.65, h: 0.073 },
    highlight: { x: 0.385, y: 0.02, w: 0.385, h: 0.95 },
    cropCell: "md:col-start-1 md:col-end-13 md:row-start-1",
    statementCell: "md:col-start-5 md:col-end-13 md:row-start-2",
    align: "right",
    drift: { crop: -0.25, statement: 0.2 },
  },
  {
    caption: "Run a load schedule and see which phase every call landed in.",
    shot: shots.stats,
    crop: { x: 0.006, y: 0.137, w: 0.99, h: 0.425 },
    highlight: { x: 0.173, y: 0.076, w: 0.33, h: 0.334 },
    cropCell: "md:col-start-1 md:col-end-13 md:row-start-2",
    statementCell: "md:col-start-1 md:col-end-9 md:row-start-1",
    align: "left",
    drift: { crop: 0.2, statement: -0.3 },
  },
  {
    caption: "Keep secrets out of scripts with values from the keychain, the environment or the file.",
    shot: shots.variables,
    crop: { x: 0.208, y: 0.075, w: 0.392, h: 0.3 },
    highlight: { x: 0.552, y: 0.13, w: 0.218, h: 0.82 },
    cropCell: "md:col-start-1 md:col-end-7 md:row-start-1",
    statementCell: "md:col-start-8 md:col-end-13 md:row-start-1",
    align: "left",
    drift: { crop: 0.35, statement: -0.3 },
  },
];
