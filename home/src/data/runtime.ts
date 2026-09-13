/* What a script calls on `kaja`, one line each.

   This list is a hint that the verb is there and nothing more. The whole of
   each one — every overload, every rule, every example — lives in the
   declaration the editor completes against and the agent reads through
   `describe_type "kaja"`, which is what `declaration` links to. That file is
   checked by the compiler against the runtime it declares, so it cannot drift
   from the code the way a second telling here would; keep this list to the
   sentence that says whether the verb is the one you want.

   Add a verb as a row. Add a `code` only where the sentence cannot say the
   shape, and keep it to the form the page hasn't already shown. */
import * as snippets from "./snippets";
import { shots, snaps, type Crop, type Shot } from "./shots";

export const declaration = "https://github.com/wham/kaja/blob/main/ui/src/kajaModule.ts";

export interface Verb {
  name: string;
  says: string;
  /* A snippet from `data/snippets.ts`, shown under the row. */
  code?: string;
  /* The file name on the snippet's title bar. The real script's, where the
     snippet is one, so a reader can go and find it. */
  file?: string;
  /* What that snippet draws when it is run, as a crop of one of the shots.
     Only worth having where the verb produces something to look at. */
  figure?: { shot: Shot; crop?: Crop; caption?: string };
}

export const verbs: Verb[] = [
  {
    name: "kaja.table",
    says: "Draw a table. Rows appear as they are added and can be rewritten once the work behind them finishes, and a cell can be a promise the table waits for. Hand the rows over as a source instead and the table pages and searches itself.",
    code: snippets.table,
    file: "scripts/movies.ts",
    figure: {
      shot: shots.canvas,
      crop: { x: 0.027, y: 0.084, w: 0.495, h: 0.282 },
      caption: "What that script draws. The rows are one page of a thousand; the box searches them.",
    },
  },
  { name: "kaja.text, kaja.code", says: "Draw a line, or a block of code." },
  {
    name: "kaja.askStr, askInt, askSelect",
    says: "Pause the run and ask for a value: text, a whole number, or one of a list. The question is drawn on the canvas and the run waits there, and the answer arrives as the kind that was asked for, so picking from a list of records hands the record back.",
    code: snippets.ask,
    file: "scripts/a-night-out.ts",
    figure: {
      shot: snaps.ask,
      caption: "That script, two answers in. An answered question keeps its answer and the run waits on the next one.",
    },
  },
  {
    name: "kaja.approve",
    says: "Hold a call until you press Approve. Use it for writes.",
    figure: {
      shot: snaps.approve,
      caption: "The request the call would send, drawn where the run stopped. Nothing leaves until Approve is pressed.",
    },
  },
  { name: "kaja.run", says: "A cell that runs another script when it is clicked." },
  {
    name: "kaja.perfTest",
    says: "Run a body on a schedule, and open the run on its Stats page.",
    code: snippets.perfTest,
    file: "scripts/how-fast.ts",
    figure: {
      shot: shots.stats,
      crop: { x: 0.005, y: 0.056, w: 0.495, h: 0.264 },
      caption: "Where that run opens. The bands behind the latency are the schedule it was given.",
    },
  },
];
