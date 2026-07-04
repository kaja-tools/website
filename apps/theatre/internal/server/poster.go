package server

import (
	"fmt"
	"html"
	"strings"

	"github.com/kaja-tools/website/v2/internal/catalog"
)

var genrePalettes = map[string][2]string{
	"concert": {"#1a0533", "#e94fd2"},
	"play":    {"#0b2818", "#e8c547"},
	"comedy":  {"#2d0a0a", "#ff8c42"},
	"opera":   {"#0a1a2d", "#7fd1e0"},
}

// poster renders a simple generated SVG poster so the catalog can serve a
// binary (non-JSON) response without shipping image assets.
func poster(e catalog.Event) []byte {
	palette, ok := genrePalettes[e.Genre]
	if !ok {
		palette = [2]string{"#111", "#eee"}
	}

	// Naive title wrapping: SVG text doesn't wrap on its own.
	var lines []string
	line := ""
	for _, word := range strings.Fields(e.Title) {
		if line != "" && len(line)+1+len(word) > 16 {
			lines = append(lines, line)
			line = word
			continue
		}
		if line != "" {
			line += " "
		}
		line += word
	}
	lines = append(lines, line)

	var title strings.Builder
	for i, l := range lines {
		title.WriteString(fmt.Sprintf(`<text x="40" y="%d" fill="%s" font-size="44" font-weight="bold" font-family="Georgia, serif">%s</text>`,
			240+i*54, palette[1], html.EscapeString(l)))
	}

	return []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="480" height="640" viewBox="0 0 480 640">
  <defs>
    <linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0%%" stop-color="%s"/>
      <stop offset="100%%" stop-color="#000"/>
    </linearGradient>
  </defs>
  <rect width="480" height="640" fill="url(#bg)"/>
  <circle cx="400" cy="110" r="150" fill="%s" opacity="0.15"/>
  <text x="40" y="120" fill="%s" font-size="20" letter-spacing="6" font-family="Helvetica, sans-serif">%s</text>
  %s
  <text x="40" y="560" fill="#ffffff" opacity="0.85" font-size="18" font-style="italic" font-family="Georgia, serif">%s</text>
  <text x="40" y="600" fill="#ffffff" opacity="0.5" font-size="14" letter-spacing="2" font-family="Helvetica, sans-serif">THE KAJA THEATRE</text>
</svg>`,
		palette[0], palette[1], palette[1], strings.ToUpper(e.Genre), title.String(), html.EscapeString(e.Tagline)))
}
