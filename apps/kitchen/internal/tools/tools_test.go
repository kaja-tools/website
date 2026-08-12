package tools

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kaja-tools/website/v2/internal/bakebook"
	"github.com/kaja-tools/website/v2/internal/mcp"
)

// The kitchen's own recipes are real, so the test book only has to agree with
// them about which ids exist and how heavy a cookie is.
func testTools(t *testing.T) map[string]mcp.Tool {
	t.Helper()
	book := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]bakebook.Cookie{
			{ID: "snickerdoodle", Name: "Snickerdoodle", PerTray: 15, DoughGramsPerCookie: 40},
			{ID: "shortbread", Name: "Salted Shortbread", PerTray: 20, DoughGramsPerCookie: 30},
		})
	}))
	t.Cleanup(book.Close)

	out := map[string]mcp.Tool{}
	for _, tool := range All(bakebook.NewClient(book.URL)) {
		out[tool.Name] = tool
	}
	return out
}

func call(t *testing.T, tool mcp.Tool, arguments string) *mcp.Result {
	t.Helper()
	result, err := tool.Call(json.RawMessage(arguments))
	if err != nil {
		t.Fatalf("%s: %v", tool.Name, err)
	}
	return result
}

// structured re-reads the answer the way a client does, through JSON.
func structured(t *testing.T, result *mcp.Result) map[string]any {
	t.Helper()
	raw, err := json.Marshal(result.Structured)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	return out
}

func TestScaleWeighsTheWholeDough(t *testing.T) {
	result := call(t, testTools(t)["scale"], `{"cookieId":"snickerdoodle","cookies":30}`)
	if result.IsError {
		t.Fatalf("scale refused: %s", result.Text)
	}
	answer := structured(t, result)

	if got := answer["doughGrams"]; got != float64(30*40) {
		t.Errorf("doughGrams = %v, want %d", got, 30*40)
	}
	if got := answer["trays"]; got != float64(2) {
		t.Errorf("trays = %v, want 2", got) // 30 cookies, 15 to a tray
	}

	// The ingredient shares add up to the dough, which is the one thing a
	// scaled recipe has to get right.
	var total float64
	for _, line := range answer["lines"].([]any) {
		total += line.(map[string]any)["grams"].(float64)
	}
	if total != float64(30*40) {
		t.Errorf("the lines add up to %vg, want %dg", total, 30*40)
	}
}

// The gap on the shelf is what sends somebody from scale to substitute, so the
// two have to agree about it.
func TestScaleReportsWhatTheShelfIsMissing(t *testing.T) {
	tools := testTools(t)

	answer := structured(t, call(t, tools["scale"], `{"cookieId":"snickerdoodle","cookies":12}`))
	missing := answer["missing"].([]any)
	if len(missing) != 1 || missing[0] != "cream of tartar" {
		t.Fatalf("missing = %v, want [cream of tartar]", missing)
	}
	if note, _ := answer["note"].(string); !strings.Contains(note, "substitute") {
		t.Errorf("note %q does not say what to do about it", note)
	}

	// And the tool it names has an answer for it.
	swap := structured(t, call(t, tools["substitute"], `{"ingredient":"cream of tartar"}`))
	if swap["use"] == "" || swap["inStock"] != false {
		t.Errorf("substitute has nothing for the gap scale reported: %v", swap)
	}

	// A recipe with nothing missing says so rather than staying quiet.
	answer = structured(t, call(t, tools["scale"], `{"cookieId":"shortbread","cookies":20}`))
	if got := answer["missing"].([]any); len(got) != 0 {
		t.Errorf("missing = %v, want none", got)
	}
}

func TestScaleRefusesWhatItCannotBake(t *testing.T) {
	tools := testTools(t)

	if result := call(t, tools["scale"], `{"cookieId":"sourdough","cookies":12}`); !result.IsError {
		t.Errorf("scale accepted an unknown cookie: %s", result.Text)
	} else if !strings.Contains(result.Text, "snickerdoodle") {
		t.Errorf("the refusal %q does not say what there is", result.Text)
	}
	if result := call(t, tools["scale"], `{"cookieId":"shortbread","cookies":0}`); !result.IsError {
		t.Errorf("scale accepted an order for no cookies: %s", result.Text)
	}
}

func TestSubstituteAnswersTheReason(t *testing.T) {
	tool := testTools(t)["substitute"]

	ordinary := structured(t, call(t, tool, `{"ingredient":"butter"}`))
	vegan := structured(t, call(t, tool, `{"ingredient":"butter","why":"one of them is vegan"}`))
	if ordinary["use"] == vegan["use"] {
		t.Errorf("the vegan answer is the ordinary one: %v", vegan["use"])
	}
	if use, _ := vegan["use"].(string); !strings.Contains(use, "vegan") {
		t.Errorf("vegan use = %q", use)
	}
}

// A near miss should still land: somebody types what the recipe called it.
func TestSubstituteFindsTheNearMiss(t *testing.T) {
	tool := testTools(t)["substitute"]

	for _, name := range []string{"chopped dark chocolate", "dark chocolate", "chocolate"} {
		result := call(t, tool, `{"ingredient":`+quote(name)+`}`)
		if result.IsError {
			t.Errorf("no answer for %q: %s", name, result.Text)
		}
	}
	if result := call(t, tool, `{"ingredient":"saffron"}`); !result.IsError {
		t.Errorf("substitute invented an answer for saffron: %s", result.Text)
	}
}

func TestPair(t *testing.T) {
	tools := testTools(t)

	answer := structured(t, call(t, tools["pair"], `{"cookieId":"shortbread"}`))
	if answer["drink"] == "" {
		t.Errorf("no drink in %v", answer)
	}
	if answer["cookie"] != "Salted Shortbread" {
		t.Errorf("cookie = %v, want the name from the book", answer["cookie"])
	}
	if result := call(t, tools["pair"], `{"cookieId":"sourdough"}`); !result.IsError {
		t.Errorf("pair accepted an unknown cookie: %s", result.Text)
	}
}

// Every tool declares what it returns, which is what makes the answer typed
// rather than a wall of JSON.
func TestEveryToolDeclaresItsSchemas(t *testing.T) {
	for name, tool := range testTools(t) {
		if tool.InputSchema == nil {
			t.Errorf("%s has no input schema", name)
		}
		if tool.OutputSchema == nil {
			t.Errorf("%s has no output schema", name)
		}
		if tool.Description == "" {
			t.Errorf("%s has no description", name)
		}
	}
}

func quote(s string) string {
	out, _ := json.Marshal(s)
	return string(out)
}
