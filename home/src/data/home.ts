/* The lists the home page is built from, out of its markup.

   The crops are regions of the shots in `shots.ts`, drawn by `cropStyle`
   there, so a picture on this page is four numbers rather than an image file
   of its own. `app-hero.png` is the shot the page already shows whole under
   the hero, so the close-ups taken from it add no weight. */
import { shots, type Crop, type Shot } from "./shots";

/* The one flow the page explains before anything else: connect an API, run a
   call, read what happened. Each step is the piece of the window it happens
   in, at the size it is legible at — a dialog is nearly square, a toolbar is a
   strip — so the three are stacked rather than set in a row.

   A crop is read here at about two thirds of the column's width, so a region
   wider than about half the shot is a picture of text nobody can read. That
   is what caps these: the run shows the script and the call it made, and the
   inspect step shows the end of the row, where the status, the duration and
   the size are. */
export interface Step {
  /* The one word the step is, for the number line above the heading. */
  step: string;
  title: string;
  says: string;
  shot: Shot;
  crop: Crop;
}

export const steps: Step[] = [
  {
    step: "Connect",
    title: "Connect your API",
    says: "Add OpenAPI, MCP, gRPC, or Twirp, and sign in with your browser instead of pasting a token.",
    shot: shots.newApp,
    crop: { x: 0.215, y: 0.07, w: 0.54, h: 0.32 },
  },
  {
    step: "Run",
    title: "Run it yourself or ask an agent",
    says: "Click a method and Kaja drafts the call as a typed script, or let an agent write one for a larger task.",
    shot: shots.run,
    crop: { x: 0.209, y: 0.0, w: 0.545, h: 0.222 },
  },
  {
    step: "Inspect",
    title: "See exactly what happened",
    says: "Every request, response, header, status and duration stays visible, whether you made the call or an agent did.",
    shot: shots.run,
    crop: { x: 0.209, y: 0.222, w: 0.5625, h: 0.349 },
  },
];

/* The three beats of the agent example, in the order they happen: somebody
   types a sentence, the agent writes a script, the calls land in the window.
   The third is a strip of the call log rather than the whole console, because
   the claim is about the row, not the response. */
export const agentCalls: Crop = { x: 0.209, y: 0.222, w: 0.525, h: 0.108 };
