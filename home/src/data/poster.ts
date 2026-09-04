/* The seven statements the home page makes below the hero, each beside the
   piece of the app it is about.

   There is one screenshot on the page — `app-hero.png`, 3104x2024 — and every
   crop here is a region of it in fractions of its width and height, so a new
   screenshot at the same size needs no new files and no new numbers anywhere
   else. The crop is drawn as a background at `100 / w` of the frame's width,
   which is what makes the region fill the frame; `Poster.astro` does that
   arithmetic.

   `highlight` is the red box drawn on the crop, in fractions of the crop
   rather than of the image, because it is placed against what the crop shows.
   Motion.astro draws it, and the line from the statement to it, the first time
   the crop is halfway up the viewport.

   `drift` is the share of the scroll past the item that each half moves by.
   The two signs are always opposite, so the statement and its crop pull apart
   as the item goes by. */
export interface Item {
  caption: string;
  /* The crop, as fractions of the screenshot: left, top, width, height. */
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
    caption: "Scripts and runs, yours and your agent’s, in one sidebar.",
    crop: { x: 0.0361, y: 0.0642, w: 0.212, h: 0.252 },
    highlight: { x: 0.0243, y: 0.3137, w: 0.9635, h: 0.1098 },
    cropCell: "md:col-start-1 md:col-end-6 md:row-start-1",
    statementCell: "md:col-start-7 md:col-end-13 md:row-start-1",
    align: "left",
    drift: { crop: 0.35, statement: -0.3 },
  },
  {
    caption: "Every app, service and method in one tree.",
    crop: { x: 0.0361, y: 0.3162, w: 0.212, h: 0.4348 },
    highlight: { x: 0.0912, y: 0.725, w: 0.8708, h: 0.2136 },
    cropCell: "md:col-start-8 md:col-end-13 md:row-start-1",
    statementCell: "md:col-start-1 md:col-end-7 md:row-start-1",
    align: "left",
    drift: { crop: -0.4, statement: 0.25 },
  },
  {
    caption: "Typed imports for every connected app.",
    crop: { x: 0.2484, y: 0.0741, w: 0.3927, h: 0.0988 },
    highlight: { x: 0.0812, y: 0.11, w: 0.4922, h: 0.34 },
    cropCell: "md:col-start-1 md:col-end-13 md:row-start-1",
    statementCell: "md:col-start-1 md:col-end-9 md:row-start-2",
    align: "left",
    drift: { crop: 0.2, statement: -0.2 },
  },
  {
    caption: "Each call a run made, on the record.",
    crop: { x: 0.2484, y: 0.3458, w: 0.7152, h: 0.0988 },
    highlight: { x: 0.0032, y: 0.37, w: 0.9946, h: 0.32 },
    cropCell: "md:col-start-1 md:col-end-13 md:row-start-2",
    statementCell: "md:col-start-4 md:col-end-13 md:row-start-1",
    align: "right",
    drift: { crop: -0.2, statement: 0.3 },
  },
  {
    caption: "The response, as it came back.",
    crop: { x: 0.2484, y: 0.4125, w: 0.2091, h: 0.1309 },
    highlight: { x: 0.0108, y: 0.2453, w: 0.3652, h: 0.6604 },
    cropCell: "md:col-start-1 md:col-end-7 md:row-start-1",
    statementCell: "md:col-start-8 md:col-end-13 md:row-start-1",
    align: "left",
    drift: { crop: 0.4, statement: -0.35 },
  },
  {
    caption: "Or the canvas the script drew instead.",
    crop: { x: 0.4124, y: 0.3409, w: 0.1095, h: 0.0543 },
    highlight: { x: 0.3382, y: 0.2182, w: 0.45, h: 0.6364 },
    cropCell: "md:col-start-7 md:col-end-13 md:row-start-1",
    statementCell: "md:col-start-1 md:col-end-6 md:row-start-1",
    align: "left",
    drift: { crop: -0.45, statement: 0.2 },
  },
  {
    caption: "Status, duration and size on every call.",
    crop: { x: 0.8634, y: 0.4051, w: 0.0975, h: 0.042 },
    highlight: { x: 0.1608, y: 0.1882, w: 0.8232, h: 0.6353 },
    cropCell: "md:col-start-1 md:col-end-6 md:row-start-1",
    statementCell: "md:col-start-7 md:col-end-13 md:row-start-1",
    align: "left",
    drift: { crop: 0.3, statement: -0.3 },
  },
];

/* The screenshot every crop is taken from, and its own aspect ratio, which is
   what turns a crop's fractions into the frame's shape. */
export const shot = { src: "/assets/app-hero.png", width: 3104, height: 2024 };
