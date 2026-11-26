/* The screenshots the site is built from, and how a region of one is drawn.

   Every shot is the window at 2880x1800, zoomed to 125%, which is what
   `scripts/demo` photographs in kaja's own repository. The zoom is what puts
   the pixels into the words: a crop is read smaller than it was taken, so the
   type has to be taken larger. A fresh set replaces the files under the same
   names, and every crop taken from them still lands.

   A crop is a region of a shot in fractions of its width and height, drawn as
   a background scaled to `100 / w` of the frame's width, which is what makes
   the region fill the frame — arithmetic rather than an image file of its own,
   so a screen contributes as many close-ups as it has things to say. The home
   page's poster and the docs' figures both draw them this way. */
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

/* The docs' own pictures, which are a different thing to a shot. A shot is the
   whole window on macOS, because the home page is selling the app and a window
   is what somebody downloads; a snap is one part of the canvas, photographed
   from the web build at the width the docs column reads it at, because a verb's
   entry is about what the verb draws and nothing around it. So a snap arrives
   cropped and is drawn whole — there are no fractions to pick, and the type is
   the size it was on screen. `.claude/skills/docs-figure` is how one is taken. */
export const snaps = {
  ask: { src: "/assets/docs-ask.png", width: 1528, height: 404 },
  approve: { src: "/assets/docs-approve.png", width: 1528, height: 532 },
  rateLimit: { src: "/assets/docs-rate-limit.png", width: 1600, height: 228 },
} satisfies Record<string, Shot>;

/* Left, top, width, height, as fractions of the shot. */
export interface Crop {
  x: number;
  y: number;
  w: number;
  h: number;
}

export const whole: Crop = { x: 0, y: 0, w: 1, h: 1 };

/* A shot is a 2x capture of the window, so a crop has half its pixels to spend
   on CSS pixels and no more. Drawn wider than that it is upscaled, and on a
   retina screen it is the small close-ups that ask for it — the tree, the
   drafts, the agent list are each a fifth of the shot in a frame twice that
   wide, and soft where the type is what they are for. So a frame is capped
   here and draws smaller in its cell rather than blurred; a crop already
   narrower than its frame is untouched, since the cap is a ceiling. */
export function cropMaxWidth(shot: Shot, crop: Crop = whole) {
  return `${Math.round((crop.w * shot.width) / 2)}px`;
}

/* The inline style that draws `crop` of `shot` inside an element, which keeps
   the region's shape through `aspect-ratio`. A crop the width or height of the
   shot has nowhere to be positioned, which is what the guards are for. */
export function cropStyle(shot: Shot, crop: Crop = whole) {
  const position = (offset: number, size: number) => (size >= 1 ? 0 : ((offset / (1 - size)) * 100).toFixed(3));
  return {
    aspectRatio: ((crop.w / crop.h) * (shot.width / shot.height)).toFixed(4),
    backgroundImage: `url(${shot.src})`,
    backgroundSize: `${(100 / crop.w).toFixed(3)}% auto`,
    backgroundPosition: `${position(crop.x, crop.w)}% ${position(crop.y, crop.h)}%`,
  };
}
