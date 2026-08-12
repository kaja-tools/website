package mcp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The whole reason Object exists: a map would sort these three keys and the
// client would render the arguments in the wrong order.
func TestObjectKeepsItsOrder(t *testing.T) {
	got, err := json.Marshal(Object{
		{"zebra", 1},
		{"apple", 2},
		{"mongoose", Object{{"nested", true}}},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"zebra":1,"apple":2,"mongoose":{"nested":true}}`
	if string(got) != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func testServer(tools ...Tool) *Server {
	return NewServer("Test Kitchen", "0.0.1", tools)
}

func post(t *testing.T, s *Server, body string) (int, map[string]any) {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	raw, _ := io.ReadAll(w.Body)
	if len(raw) == 0 {
		return w.Code, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unreadable response %q: %v", raw, err)
	}
	return w.Code, out
}

func TestInitialize(t *testing.T) {
	s := testServer()
	_, resp := post(t, s, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`)

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result in %v", resp)
	}
	// A client that asks for an older revision is answered in its own.
	if got := result["protocolVersion"]; got != "2025-03-26" {
		t.Errorf("protocolVersion = %v, want the client's own", got)
	}
	if _, ok := result["capabilities"].(map[string]any)["tools"]; !ok {
		t.Errorf("no tools capability in %v", result)
	}
}

// A client works out which revision this server speaks from the shape of the
// refusal, so an unknown method has to come back as a JSON-RPC error rather
// than an HTTP failure.
func TestUnknownMethodIsAJSONRPCError(t *testing.T) {
	s := testServer()
	code, resp := post(t, s, `{"jsonrpc":"2.0","id":1,"method":"server/discover"}`)

	if code != http.StatusOK {
		t.Errorf("status = %d, want 200", code)
	}
	err, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("no error in %v", resp)
	}
	if err["code"] != float64(codeMethodNotFound) {
		t.Errorf("code = %v, want %d", err["code"], codeMethodNotFound)
	}
}

func TestNotificationIsNotAnswered(t *testing.T) {
	s := testServer()
	code, resp := post(t, s, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)

	if code != http.StatusAccepted {
		t.Errorf("status = %d, want 202", code)
	}
	if resp != nil {
		t.Errorf("a notification was answered with %v", resp)
	}
}

// The two failures a tool can have are different things and must not look
// alike: "no" is a result, and a broken server is an error.
func TestToolRefusalIsAResultAndFailureIsAnError(t *testing.T) {
	s := testServer(
		Tool{
			Name: "refuses",
			Call: func(json.RawMessage) (*Result, error) {
				return &Result{Text: "no", IsError: true}, nil
			},
		},
		Tool{
			Name: "breaks",
			Call: func(json.RawMessage) (*Result, error) {
				return nil, errors.New("the mixer caught fire")
			},
		},
	)

	_, resp := post(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"refuses","arguments":{}}}`)
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("a refusal came back as %v", resp)
	}
	if result["isError"] != true {
		t.Errorf("isError = %v, want true", result["isError"])
	}

	_, resp = post(t, s, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"breaks","arguments":{}}}`)
	if _, ok := resp["error"]; !ok {
		t.Errorf("a failing tool came back as %v", resp)
	}
}

func TestCallingAToolThatIsNotThere(t *testing.T) {
	s := testServer(Tool{Name: "scale"})
	_, resp := post(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"knead"}}`)

	err, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("no error in %v", resp)
	}
	// The refusal names what there is, so one round trip is enough.
	if message, _ := err["message"].(string); !strings.Contains(message, "scale") {
		t.Errorf("message %q does not say what this server has", message)
	}
}

func TestOnlyPostIsAllowed(t *testing.T) {
	w := httptest.NewRecorder()
	testServer().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/mcp", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET = %d, want 405", w.Code)
	}
}
