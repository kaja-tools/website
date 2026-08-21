/* The docs' code, out of the markup so each block reads as the file it is.
   Every one of these is meant to be copied and run, so keep them runnable:
   no ellipses standing in for lines, and no placeholder that isn't obviously
   one. */

export const dockerRun = `
docker run --pull always --name kaja -d -p 41520:41520 \\
    -v /my_app/proto:/workspace/proto \\
    -v /my_app/kaja.json:/workspace/kaja.json \\
    -v /my_app/scripts:/workspace/scripts \\
    --add-host=host.docker.internal:host-gateway kajatools/kaja:latest
`;

/* One app of each type, so the four blocks can be read against each other.
   Two lines apiece rather than one, so the block fits the column it is read
   in — a snippet you have to scroll sideways is one nobody finishes. */
export const apps = `
{
  "apps": [
    { "name": "users", "twirp": {
        "url": "http://localhost:41522", "proto_dir": "users/proto" } },
    { "name": "teams", "grpc": {
        "url": "localhost:41523", "reflection": true } },
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
    "host": "localhost:41523",
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

export const script = `
import { kaja } from "kaja";
import { Teams } from "teams/proto/teams";

const team = await kaja.approve(
  Teams.CreateTeam({ name: "Acme", tenant: kaja.variables.tenant }),
);

kaja.table(["name", "id"], [[team.name, team.id]]);
`;

export const mcpAdd = `
claude mcp add --transport http kaja http://127.0.0.1:41521/mcp \\
    --header "Authorization: Bearer <token>"
`;
