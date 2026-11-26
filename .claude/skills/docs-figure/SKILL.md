---
name: docs-figure
description: Document a kaja runtime verb (kaja.table, kaja.approve, …) on the website docs — a one-line entry, a real script as the snippet, a picture of what that script draws, and a link to the declaration that is the reference. Use when adding or revising a `kaja.*` entry in home/src/data/runtime.ts, when a docs figure is missing or has to be taken by running kaja's web build, when a docs snippet renders too narrow or scrolls sideways, or when picking and verifying a screenshot crop for the docs or the home page.
---

# Documenting a kaja verb

## The rule this is all built on

`ui/src/kajaModule.ts` in [wham/kaja](https://github.com/wham/kaja) is the
**reference** for every `kaja.*` verb. It is the TypeScript the editor
completes against and what an agent reads through `describe_type "kaja"`, and
the compiler checks it against the runtime it declares — so it cannot drift.

What is written elsewhere is spent on **teaching**, never on stating the
surface. This site's docs are one sentence per verb. Kaja's own
`server/pkg/mcp/guide.md` teaches — a worked example per shape worth writing,
plus what is different about being an agent — and names `describe_type "kaja"`
for the rest.

**An example earns its place; a member list does not.** A run reports type
errors checked against those same declarations, so a stale example is caught,
where a retelling of what `.column` or a failed cell does is a second model of
the surface and drifts unnoticed. That is the line to trim on.

So: **the long explanation goes in the declaration, in the kaja repo.** If the
verb is badly documented, fix it there first. This skill is only the index.

## Order of work

1. **Declaration first**, in wham/kaja. Examples, overloads, rules.
2. **Check the guide**, in wham/kaja `server/pkg/mcp/guide.md`. Keep its worked
   examples and what is an agent's alone ("nobody is paging your run"); cut any
   paragraph that restates the surface, naming it in one clause that points at
   `describe_type "kaja"`.
3. **Add the row here.** One sentence. A snippet only if the sentence cannot say
   the shape. A figure if the verb draws something — taken by running that
   script in kaja's web build, below.

Step 2 is the one people skip. **Don't overcorrect it either**: the guide ships
in `instructions` on every session and the declaration costs a call, so cutting
a few hundred tokens of teaching to save them on sessions that never use the
verb makes the common task far more expensive. Do the arithmetic before you cut.

## The row

`home/src/data/runtime.ts` is the list; `home/src/pages/docs/[platform].astro` renders it in
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

**Use a real script from kaja's `workspace/scripts/`**, trimmed to fit. That is
what lets the figure under it be the snippet's own output — you run that script
to take the picture. Name the file on the title bar after the real one
(`scripts/movies.ts`) so a reader can find it.

- Snippets live in `home/src/data/snippets.ts`, never inline in the page.
- **Keep every line at or under ~78 characters.** Past that it scrolls sideways
  in the docs column and nobody finishes it. Check:
  `awk 'length > 78 {print NR": "length}' home/src/data/snippets.ts`
- Trim by dropping columns and parameters, not by replacing lines with `…`.
  Every snippet on this site is meant to be pasted into demo.kaja.tools and run.
- **Trim to what the figure can legibly show.** A snippet declaring three
  columns above a picture showing two is the mismatch a reader notices first.

## The figure

**A verb that draws on the canvas carries a figure**, because the sentence
beside it says what the verb is for and only the picture says what it leaves on
screen. Two of them have no picture and it is a decision rather than a gap:
`kaja.text`/`kaja.code` draws one line of prose, which at the column's width is
a 16:1 band saying what the sentence already said, and nothing in kaja's
`workspace/scripts/` calls `kaja.run` yet — that figure waits on a script that
does, not on a mock-up of one.

**Two kinds of picture, and the docs' own is a snap.** A **shot** is the whole
macOS window (`shots` in `home/src/data/shots.ts`), photographed by
`scripts/demo` in wham/kaja: it is what the home page is built from, because
that page is selling an app and a window is what somebody downloads, and a
picture out of it is a **crop**, four fractions of the shot. A **snap** is one
part of the canvas taken from the **web build** at the width the docs column
reads it at (`snaps`, beside `shots`): it arrives cropped, is drawn whole, and
its type is the size it was on screen. **A new docs figure is a snap** — take
one rather than hunting the shots for a corner that nearly says it. A crop
stays where a shot already says the thing exactly (`kaja.table`'s and the perf
run's do).

### Taking a snap

Run kaja's own web build and drive it. It is the build `demo.kaja.tools`
serves, so this is that page with a run you control — and the sandbox's
Chromium can only reach localhost anyway, so drive the local one.

```bash
cd /path/to/kaja && scripts/server    # port 41520, the demo workspace
```

It is slow the first time (protoc, bun, a Go build) and it holds the port, so
kill it when you are done. Playwright goes outside the project, exactly as
below.

```js
const page = await browser.newPage({ viewport: { width: 800, height: 720 }, deviceScaleFactor: 2 });
await page.goto("http://localhost:41520/", { waitUntil: "domcontentloaded" });
await page.waitForTimeout(20000);                                    // the apps compile
await page.getByText("a-night-out.ts", { exact: false }).first().click();
await page.getByRole("button", { name: "Hide sidebar" }).click();
await page.getByRole("button", { name: /^Run/ }).click();
await page.keyboard.press("Control+Shift+F");                        // Mod is Ctrl off macOS
```

**Drive it by its test ids**, which is what they are for:
`canvas`, `canvas-ask-choice`, `canvas-ask-input`, `canvas-ask-settled`,
`canvas-approve` (and `-send`/`-stop`), `canvas-table-search`/`-next`,
`stats-tiles`, `stats-latency`, `console-view-canvas`, `console-fullscreen`.
`grep -rho 'data-testid="[^"]*"' ui/src | sort -u` is the list.

Three things a run does that reading the script will not tell you:

- **Clicking a choice answers a select.** An `Enter` after it answers the *next*
  question with nothing, which is how a figure ends up with an empty chip in it.
- **An ask focuses its own field**, but type into `canvas-ask-input` and read
  the value back before pressing Enter; a field that was not there yet takes
  the keystrokes nowhere.
- **`a-night-out.ts` parks on `kaja.approve`, and that is the picture.** Never
  press Approve: the seats are a real call to the demo's own service.

**Clip to the blocks, not to the pane they scroll in** — a figure is the
document, and the canvas is mostly empty below it:

```js
const clip = await page.evaluate(([from, to]) => {
  const kids = [...document.querySelector('[data-testid="canvas"]').children];
  const first = kids[from].getBoundingClientRect();
  const last = kids[to ?? kids.length - 1].getBoundingClientRect();
  const pad = 14;
  return { x: first.x - pad, y: first.y - pad, width: first.width + pad * 2, height: last.bottom - first.y + pad * 2 };
}, [from, to]);
await page.screenshot({ path, clip });
```

**The 800px viewport is the whole of the sizing.** With the sidebar hidden the
canvas is ~764 CSS px, which is the docs column (756) — so the type in the
figure is the type in the app, and `deviceScaleFactor: 2` is what makes it
retina. A wider window is a snap the column shrinks, and the words go with it.

Then: the PNG lands in `home/public/assets/` as `docs-<verb>.png`, an entry in
`snaps` carries its **real pixel size** (`file` or a 16-byte read of the IHDR),
and the row's `figure` names it with no crop.

### Picking a crop

A crop is for the shots — the home page's pictures, and the docs figures
already taken from one. The docs column is about **756px** and a shot is 2880px
wide, so the crop is scaled to roughly `756 / (w × 2880)`. Text in a shot is
~32px (13px at 125% zoom, photographed at 2x), so:

| crop `w` | scale | text on page | verdict |
| --- | --- | --- | --- |
| 0.50 | 52% | ~17px | comfortable |
| 0.62 | 42% | ~13px | readable |
| 0.82 | 32% | ~10px | too small |
| 0.95 | 27% | ~9px | a band, texture only |

**So a crop wider than about two thirds of the shot is a picture nobody can read
the words in.** Full-width table crops do not work in this column — accept losing
the right-hand columns rather than shrinking the type.

### Shape, either way

Keep the aspect under about 5:1; past 8:1 it is a 40px band on a phone, which
is what a single drawn line comes out as. The figure's own `mt-8` is for
standing in prose. Inside a verb block it belongs to the snippet above it, so it
is rendered in a `[&>figure]:mt-0` wrapper and the block's `gap-3` is the whole
spacing.

### Verifying either

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
await page.goto("http://localhost:4399/docs/desktop/", { waitUntil: "networkidle" });
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

Copy wraps at `max-w-copy` (720px). **Code and figures take the whole column**
(`max-w-docs-main`), like every other block on the page. Putting `max-w-copy` on
a wrapper that holds a snippet is what makes it render narrower than the snippet
above it.

## Before pushing

From `home/`: `npm run format`, `npm run check`, `npm run build` — CI runs all
three. Then check the PR's preview app, which is the real thing at a real width.

## Kaja repo side

Changes to the declaration or the guide are a **separate PR in wham/kaja**. Its
`AGENTS.md` carries this same rule under "The `kaja` object is a declaration
too". After editing the guide, run `go test -tags development ./pkg/mcp/...`
from `server/`.
