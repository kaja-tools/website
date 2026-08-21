/* The docs' syntax colours, as a Shiki theme. Shiki writes whatever string it
   is given straight into the token's inline `style`, so the values here are
   the `code-*` custom properties from global.css rather than copies of them —
   there is one palette and this is a mapping onto it, not a second one.

   Three hues and two greys, on one rule: what names a thing takes the cool
   stop, what holds a value takes the warm one, and what *is* a value — a
   number, a boolean, a call — takes the sky. Everything else is the ramp. */

const key = "var(--color-code-key)";
const string = "var(--color-code-string)";
const literal = "var(--color-code-literal)";
const muted = "var(--color-code-muted)";
const plain = "var(--color-code)";

export const codeTheme = {
  name: "kaja-docs",
  type: "dark" as const,
  colors: {
    "editor.background": "var(--color-card)",
    "editor.foreground": plain,
  },
  settings: [
    {
      /* JSON keys, TypeScript keywords, and the command a shell line runs —
         all of them the name of the thing rather than the thing. */
      scope: ["support.type.property-name", "keyword", "storage", "entity.name.command", "entity.name.function.call"],
      settings: { foreground: key },
    },
    {
      scope: ["string", "string.quoted"],
      settings: { foreground: string },
    },
    {
      scope: ["constant.numeric", "constant.language", "entity.name.function", "support.class"],
      settings: { foreground: literal },
    },
    {
      /* A shell flag and a line continuation are punctuation on the command,
         so they sit back and let the paths beside them read. */
      scope: ["constant.other.option", "constant.character.escape", "comment"],
      settings: { foreground: muted },
    },
    {
      /* An unquoted shell argument is a path, a port or an image name — the
         grammar can't tell which, so it stays plain rather than borrowing a
         hue that would mean something. An operator is punctuation and is only
         here because TextMate files `=` and `await` under the one scope. */
      scope: ["string.unquoted.argument", "meta.argument", "keyword.operator"],
      settings: { foreground: plain },
    },
  ],
};
