/* The docs' code, out of the markup so each block reads as the file it is.
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
