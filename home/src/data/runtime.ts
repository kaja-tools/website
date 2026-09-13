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
import { shots, type Crop, type Shot } from "./shots";

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
  { name: "kaja.askStr", says: "Pause the run and ask for a value." },
  { name: "kaja.approve", says: "Hold a call until you press Approve. Use it for writes." },
  { name: "kaja.run", says: "A cell that runs another script when it is clicked." },
  { name: "kaja.perfTest", says: "Run a body on a schedule, and open the run on its Stats page." },
];
