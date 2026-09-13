---
name: docs-figure
description: Document a kaja runtime verb (kaja.table, kaja.approve, …) on the website docs — a one-line entry, a real script as the snippet, a crop of an existing shot as the figure, and a link to the declaration that is the reference. Use when adding or revising a `kaja.*` entry in home/src/data/runtime.ts, when a docs snippet renders too narrow or scrolls sideways, or when picking and verifying a screenshot crop for the docs or the home page.
---

# Documenting a kaja verb

## The rule this is all built on

`ui/src/kajaModule.ts` in [wham/kaja](https://github.com/wham/kaja) is the
**reference** for every `kaja.*` verb. It is the TypeScript the editor
completes against and what an agent reads through `describe_type "kaja"`, and
the compiler checks it against the runtime it declares — so it cannot drift.

Everything else is an **index line into it**: this site's docs, and kaja's own
`server/pkg/mcp/guide.md`. Each carries one sentence saying whether this is the
verb you want, plus whatever is theirs alone. Never retell `.row`/`.update`/
`total` here — that copy goes stale, and did.

So: **the long explanation goes in the declaration, in the kaja repo.** If the
verb is badly documented, fix it there first. This skill is only the index.

## Order of work

1. **Declaration first**, in wham/kaja. Examples, overloads, rules.
2. **Trim the guide**, in wham/kaja `server/pkg/mcp/guide.md`. Keep only what is
   different about being an agent ("nobody is paging your run"); point at
   `describe_type "kaja"` for the rest.
3. **Add the row here.** One sentence. A snippet only if the sentence cannot say
   the shape. A figure only if the verb draws something.

Step 2 is the one people skip. It is where drift comes from.

## The row

`home/src/data/runtime.ts` is the list; `home/src/pages/docs.astro` renders it in
the Scripts section. A verb is a row:

```ts
{
  name: "kaja.table",
  says: "One sentence: what it draws, and the thing about it a reader would not guess.",
  code: snippets.table,        // optional
  file: "scripts/movies.ts",   // the real script's name, on the snippet's title bar
  figure: { shot: shots.canvas, crop: { … }, caption: "…" },  // optional
}
```

The list ends with one link to the declaration, for all of them. Don't add a
link per row — they would all point at the same file.

## The snippet

**Use a real script from kaja's `workspace/scripts/`**, trimmed to fit. Those
are the scripts `scripts/demo` photographs, so a real script is the one thing
that lets the figure under it be that snippet's actual output. Name the file on
the title bar after the real one (`scripts/movies.ts`) so a reader can find it.

- Snippets live in `home/src/data/snippets.ts`, never inline in the page.
- **Keep every line at or under ~78 characters.** Past that it scrolls sideways
  in the docs column and nobody finishes it. Check:
  `awk 'length > 78 {print NR": "length}' home/src/data/snippets.ts`
- Trim by dropping columns and parameters, not by replacing lines with `…`.
  Every snippet on this site is meant to be pasted into demo.kaja.tools and run.
- **Trim to what the figure can legibly show.** A snippet declaring three
  columns above a picture showing two is the mismatch a reader notices first.

## The figure

Every picture on the site is a **crop of one of the eight shots** in
`home/src/data/shots.ts` — four fractions, drawn as a background. Nothing under
`public/assets/` is a cut-out to re-cut, and you do not take new screenshots of
the app here: a fresh set comes from `scripts/demo` in wham/kaja, which drives
the shipped macOS app. Pick the shot whose script matches your snippet.

Known shots worth knowing: `canvas` is `movies.ts` — a live `kaja.table` with a
search box, a pager and `1–100 of 1,283`. `stats` is a perf run. `run` is the
whole window.

### Picking a crop

The docs column is about **756px**, and a shot is 2880px wide, so the crop is
scaled to roughly `756 / (w × 2880)`. Text in a shot is ~32px (13px at 125% zoom,
photographed at 2x), so:

| crop `w` | scale | text on page | verdict |
| --- | --- | --- | --- |
| 0.50 | 52% | ~17px | comfortable |
| 0.62 | 42% | ~13px | readable |
| 0.82 | 32% | ~10px | too small |
| 0.95 | 27% | ~9px | a band, texture only |

**So a crop wider than about two thirds of the shot is a picture nobody can read
the words in.** Full-width table crops do not work in this column — accept losing
the right-hand columns rather than shrinking the type. Keep the aspect under
about 5:1; past 8:1 it is a 40px band on a phone.

The figure's own `mt-8` is for standing in prose. Inside a verb block it belongs
to the snippet above it, so it is rendered in a `[&>figure]:mt-0` wrapper and the
block's `gap-3` is the whole spacing.

### Verifying a crop

Do not eyeball the fractions — render the page and look. Chromium is
pre-installed; **do not add playwright to `home/package.json`** (CI runs
`npm ci` and would try to fetch browsers).

```bash
cd home && npm run build
npx serve dist -p 4399 &

# Install playwright outside the project so package.json stays clean.
cd "$SCRATCHPAD" && npm init -y >/dev/null && npm i playwright >/dev/null
```

```js
// shot.mjs — run from the directory playwright is installed in
import { chromium } from "playwright";
const browser = await chromium.launch({ executablePath: "/opt/pw-browsers/chromium" });
const page = await browser.newPage({ viewport: { width: 1440, height: 1400 }, deviceScaleFactor: 2 });
await page.goto("http://localhost:4399/docs/", { waitUntil: "networkidle" });
await page.evaluate(() => document.querySelector("#scripts").scrollIntoView());
await page.waitForTimeout(1500);
await page.locator("#scripts figure").first().screenshot({ path: process.argv[2] });
await browser.close();
```

Read the PNG back and check three things: the words are legible, the columns
match the snippet, and the snippet above it has not started scrolling sideways.
Screenshot the whole `#scripts` section too — that is how you catch a block
constrained to `max-w-copy` when it should take the column.

## Widths, once

Copy wraps at `max-w-copy` (620px). **Code and figures take the whole column**
(`max-w-docs-main`), like every other block on the page. Putting `max-w-copy` on
a wrapper that holds a snippet is what makes it render narrower than the snippet
above it.

## Before pushing

From `home/`: `npm run format`, `npm run check`, `npm run build` — CI runs all
three. Then check the PR's preview app, which is the real thing at a real width.

## Kaja repo side

Changes to the declaration or the guide are a **separate PR in wham/kaja**. Its
`AGENTS.md` carries this same rule under "The `kaja` object is a declaration
too". After trimming the guide, run `go test -tags development ./pkg/mcp/...`
from `server/`.
