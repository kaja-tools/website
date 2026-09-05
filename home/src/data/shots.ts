/* The screenshots the site is built from, and how a region of one is drawn.

   Every shot is the window at 2880x1800, which is what `scripts/demo`
   photographs in kaja's own repository. A fresh set replaces the files under
   the same names, and every crop taken from them still lands.

   A crop is a region of a shot in fractions of its width and height, drawn as
   a background scaled to `100 / w` of the frame's width, which is what makes
   the region fill the frame — arithmetic rather than an image file of its own,
   so a screen contributes as many close-ups as it has things to say. The home
   page's poster and the docs' rail both draw them this way. */
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

/* Left, top, width, height, as fractions of the shot. */
export interface Crop {
  x: number;
  y: number;
  w: number;
  h: number;
}

export const whole: Crop = { x: 0, y: 0, w: 1, h: 1 };

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
