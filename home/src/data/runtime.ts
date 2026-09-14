/* What a script calls on `kaja`, one line each.

   This list is a hint that the verb is there and nothing more. The whole of
   each one — every overload, every rule, every example — lives in the
   declaration the editor completes against and the agent reads through
   `describe_type "kaja"`, which is what `declaration` links to. That file is
   checked by the compiler against the runtime it declares, so it cannot drift
   from the code the way a second telling here would; keep this list to the
   sentences that say whether the verb is the one you want.

   Add a verb as a row. `use` is the line it gets in the summary table above
   the list, so it is what you reach for the verb to do and nothing else. Add
   a `code` only where the sentences cannot say the shape, and keep it to the
   form the page hasn't already shown. */
import type { Platform } from "./docs";
import * as snippets from "./snippets";
import { shots, snaps, type Crop, type Shot } from "./shots";

export const declaration = "https://github.com/wham/kaja/blob/main/ui/src/kajaModule.ts";

export interface Verb {
  name: string;
  /* The right-hand column of the summary table: what you use it to do. */
  use: string;
  /* The entry itself, one paragraph per element. */
  says: string[];
  /* Where the two builds differ, what replaces `says` on that build's page.
     Only the desktop holds variable values itself, so only kaja.variables
     needs this. */
  saysOn?: Partial<Record<Platform, string[]>>;
  /* A snippet from `data/snippets.ts`, shown under the row. */
  code?: string;
  /* The file name on the snippet's title bar. The real script's, where the
     snippet is one, so a reader can go and find it. */
  file?: string;
  /* What that snippet draws when it is run, as a crop of one of the shots.
     Only worth having where the verb produces something to look at. */
  figure?: { shot: Shot; crop?: Crop; caption?: string };
}

/* The paragraphs this build reads for a verb. */
export function saysFor(verb: Verb, platform: Platform): string[] {
  return verb.saysOn?.[platform] ?? verb.says;
}

export const verbs: Verb[] = [
  {
    name: "kaja.table",
    use: "Show tabular results, including paged data.",
    says: [
      "Draw a table on the canvas. You can add rows and update them while the script runs. A cell can also be a promise or a function, and Kaja fills it in when the work finishes.",
      "For a large result set, pass a row source instead of an array: an async generator that yields a page at a time. Kaja pages through it and searches it.",
    ],
    code: snippets.table,
    file: "scripts/movies.ts",
    figure: {
      shot: shots.canvas,
      crop: { x: 0.027, y: 0.084, w: 0.495, h: 0.282 },
      caption: "What that script draws. The rows are one page of a thousand; the box searches them.",
    },
  },
  {
    name: "kaja.text, kaja.code",
    use: "Add text or code to the canvas.",
    says: ["Add a line of text or a block of code to the canvas."],
  },
  {
    name: "kaja.askStr, askInt, askSelect",
    use: "Ask the user for input while a script runs.",
    says: [
      "Pause the script and ask the user for text, a whole number, or a choice from a list. The question appears on the canvas, and the script continues after the user answers.",
      "askSelect returns the value attached to the option that was picked, including an object if you provided one.",
    ],
    code: snippets.ask,
    file: "scripts/a-night-out.ts",
    figure: {
      shot: snaps.ask,
      caption: "That script, two answers in. An answered question keeps its answer and the run waits on the next one.",
    },
  },
  {
    name: "kaja.approve",
    use: "Require approval before sending a call.",
    says: ["Require approval before sending a call. Use this for operations that create, update, or delete data."],
    figure: {
      shot: snaps.approve,
      caption: "The request the call would send, drawn where the run stopped. Nothing leaves until Approve is pressed.",
    },
  },
  {
    name: "kaja.run",
    use: "Run another script from a table cell.",
    says: ["Put a cell in a table that runs another script when you click it."],
  },
  {
    name: "kaja.rateLimit",
    use: "Keep calls within an API's rate limit.",
    says: [
      "Keep calls within the rate limit an API publishes. Nothing is paced until you call this. After that Kaja spreads the calls out, and waits when the remaining budget is spent.",
      "Kaja reads the budget from the response headers: RateLimit-Limit, RateLimit-Remaining and RateLimit-Reset, the X-RateLimit- and X-Rate-Limit- spellings of the same three, and Retry-After. For an API that publishes none of them, set a ceiling yourself with the perSecond option.",
    ],
    code: snippets.rateLimit,
    file: "scripts/rate-limit.ts",
    figure: {
      shot: snaps.rateLimit,
      caption: "What the limiter draws while that loop runs: the budget, and what obeying it has cost so far.",
    },
  },
  {
    name: "kaja.perfTest",
    use: "Run a timed performance test.",
    says: [
      "Run a function over and over on a schedule: a duration or a number of iterations, plus concurrency, warmup and ramp-up. Kaja opens the run on its Stats page.",
    ],
    code: snippets.perfTest,
    file: "scripts/how-fast.ts",
    figure: {
      shot: shots.stats,
      crop: { x: 0.005, y: 0.056, w: 0.495, h: 0.264 },
      caption: "Where that run opens. The bands behind the latency are the schedule it was given.",
    },
  },
  {
    name: "kaja.input",
    use: "Read input passed to the script.",
    says: [
      "Read the values a deeplink or a kaja.run cell passed to this run, by name: kaja.input.city.",
      "Every value arrives as a string. Run repeats the values the script last ran with, so a script keeps what it was last given. Run with parameters opens those values for editing first.",
    ],
  },
  {
    name: "kaja.variables",
    use: "Read workspace variables.",
    says: ["Read the workspace's variables."],
    saysOn: {
      desktop: [
        "Read the workspace's variables. A variable stored in the Keychain, or read from an environment variable, works the same way as one typed into the workspace.",
      ],
      docker: [
        "Read the variables defined in kaja.json. A secret is resolved inside the container, so a script reads the ${NAME} placeholder rather than the value.",
      ],
    },
  },
  {
    name: "kaja.uuidV4",
    use: "Generate a UUID.",
    says: ["Generate a random UUID. Inside a script, crypto.randomUUID() is this same function."],
  },
  {
    name: "kaja.value, struct, listValue",
    use: "Build protobuf values from JavaScript values.",
    says: ["Convert a plain JavaScript value to a google.protobuf.Value, Struct or ListValue. Kaja builds the oneof fields for you."],
  },
];
