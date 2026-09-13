/* The code the site shows, out of the markup so each block reads as the file
   it is — the docs' blocks, and the one script the home page has an agent
   write.
   Every one of these is meant to be copied and run, so keep them runnable:
   no ellipses standing in for lines, and no placeholder that isn't obviously
   one. The scripts are written against the demo's own apps, so a reader can
   paste one into demo.kaja.tools and press Run. */

export const dockerRun = `
docker run --pull always --name kaja -d -p 41520:41520 \\
    -v /my_app/proto:/workspace/proto \\
    -v /my_app/kaja.json:/workspace/kaja.json \\
    -v /my_app/scripts:/workspace/scripts \\
    -e KAJA_TOKEN="$TOKEN" \\
    --add-host=host.docker.internal:host-gateway kajatools/kaja:latest
`;

/* One app of each type, so the four blocks can be read against each other.
   Two or three lines apiece rather than one, so the block fits the column it
   is read in — a snippet you have to scroll sideways is one nobody finishes. */
export const apps = `
{
  "apps": [
    { "name": "users", "twirp": {
        "url": "http://host.docker.internal:41522",
        "proto_dir": "proto/users" } },
    { "name": "teams", "grpc": {
        "url": "host.docker.internal:41523", "reflection": true } },
    { "name": "theatre", "openapi": {
        "spec_url": "https://theatre.kaja.tools/openapi.yaml" } },
    { "name": "concierge", "mcp": {
        "url": "https://concierge.kaja.tools/mcp" } }
  ]
}
`;

/* One variable of each source, and an app reading two of them — including the
   one inside a longer value, which is the part a reader doesn't expect. */
export const variables = `
{
  "variables": {
    "host": "host.docker.internal:41523",
    "token": "\${secret}",
    "tenant": "\${env:TENANT_ID}"
  },
  "apps": [
    { "name": "teams", "grpc": {
        "url": "\${host}",
        "headers": { "Authorization": "Bearer \${token}" } } }
  ]
}
`;

/* Two apps, two protocols, and the one call in the demo that spends money
   behind an approval. */
export const script = `
import { kaja } from "kaja";
import { Theatre } from "theatre";
import { Seating } from "seating";

const { shows } = await Theatre.ListShows({ city: "Chicago" });
kaja.table(["show", "starts"], shows.map((show) => [show.id, show.startsAt]));

await kaja.approve(
  Seating.BookSeats({ showId: shows[0].id, seatIds: ["F7", "F8"] }),
);
`;

/* The same script taking its city from the link that ran it, and asking when
   no link did — so one file works from Run and from a deeplink. */
export const deeplinkScript = `
const city = kaja.input.city ?? (await kaja.askStr("Which city?"));
const { shows } = await Theatre.ListShows({ city });
`;

export const deeplinkDesktop = `
kaja://run/whats-on?city=Chicago
`;

export const deeplinkDocker = `
http://localhost:41520/#run/whats-on?city=Chicago
`;

/* The home page's agent example: the script an agent writes when somebody
   asks it which endpoints are slow. It is the demo's own Theatre app, so a
   reader can paste it into demo.kaja.tools and press Run. */
export const agentScript = `
import { kaja } from "kaja";
import { Theatre } from "theatre";

const calls = {
  ListMovies: () => Theatre.ListMovies({}),
  ListShows: () => Theatre.ListShows({}),
  ListTheaters: () => Theatre.ListTheaters({}),
};

const slow: string[][] = [];

for (const [name, send] of Object.entries(calls)) {
  const started = Date.now();
  await send();
  const ms = Date.now() - started;
  if (ms > 500) slow.push([name, \`\${ms} ms\`]);
}

kaja.table(["method", "duration"], slow);
`;

/* The one kaja.table form the script above doesn't already show: rows handed
   over as a source rather than pushed, which is what gets the table its pager
   and its search box. Written against the demo's Theatre app, cursor and all,
   so it is a whole loop rather than a sketch of one. */
export const table = `
import { kaja } from "kaja";
import { Theatre } from "theatre";

const shows = kaja.table(["show", "theater", "starts"], async function* (search) {
  for (let cursor = ""; ; ) {
    const page = await Theatre.ListShows({ city: search, cursor });
    shows.total(page.total);
    yield* page.shows.map((show) => [show.id, show.theaterId, show.startsAt]);
    if (!(cursor = page.nextCursor)) return;
  }
});
`;
