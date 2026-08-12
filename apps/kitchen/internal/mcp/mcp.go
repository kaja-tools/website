// Package mcp is a very small Model Context Protocol server: JSON-RPC 2.0 over
// a single HTTP endpoint, the tools half of the protocol and nothing else.
//
// It is hand-rolled and dependency-free on purpose. A demo service that pulls
// in an SDK teaches you about the SDK; this one fits in two files, so what the
// protocol actually is stays readable.
package mcp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
)

// The revision this server speaks. Clients that ask for a different one are
// answered in theirs when we can, and in this one when we cannot.
const ProtocolVersion = "2025-06-18"

// Object is a JSON object that keeps its keys in the order they were written.
// Schema property order is the order a client renders a tool's arguments in,
// and a Go map would sort them alphabetically — so a tool whose first argument
// is the important one would arrive with it in the middle.
//
// A Member is a pair rather than a two-field struct so that a schema can be
// written as nested literals without a field name on every line. Half the value
// of declaring schemas in Go is that they still read like the JSON they become.
type Object []Member

// Member is one key and its value. Element 0 must be a string.
type Member [2]any

func (o Object) MarshalJSON() ([]byte, error) {
	var b strings.Builder
	b.WriteByte('{')
	for i, m := range o {
		name, ok := m[0].(string)
		if !ok {
			return nil, fmt.Errorf("object key %v is not a string", m[0])
		}
		if i > 0 {
			b.WriteByte(',')
		}
		key, err := json.Marshal(name)
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(m[1])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		b.Write(key)
		b.WriteByte(':')
		b.Write(value)
	}
	b.WriteByte('}')
	return []byte(b.String()), nil
}

// Result is what a tool hands back. Text is what a client without a schema
// shows; Structured is the typed answer, which is the one worth having.
//
// IsError is the tool reporting that it could not do the job — the call
// succeeded and the answer is "no". A Go error from Call is different: that is
// the server failing, and it comes back as a JSON-RPC error.
type Result struct {
	Text       string
	Structured any
	IsError    bool
}

type Tool struct {
	Name        string
	Title       string
	Description string
	// JSON Schema for the arguments, and for the structured answer.
	InputSchema  Object
	OutputSchema Object
	Call         func(arguments json.RawMessage) (*Result, error)
}

type Server struct {
	name    string
	version string
	tools   []Tool
}

func NewServer(name, version string, tools []Tool) *Server {
	return &Server{name: name, version: version, tools: tools}
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

const (
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternal       = -32603
)

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// The protocol's optional server-initiated stream. This server never
		// speaks first, so there is nothing to open.
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "POST a JSON-RPC request to this endpoint", http.StatusMethodNotAllowed)
		return
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nil, codeInvalidParams, fmt.Sprintf("could not read the request: %v", err))
		return
	}

	// A notification has no id and takes no answer. `notifications/initialized`
	// is the only one a client sends us.
	if len(req.ID) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	slog.Info("mcp", "method", req.Method)

	result, code, message := s.dispatch(req)
	if code != 0 {
		writeError(w, req.ID, code, message)
		return
	}
	writeResult(w, req.ID, result)
}

func (s *Server) dispatch(req request) (result any, code int, message string) {
	switch req.Method {
	case "initialize":
		return s.initialize(req.Params), 0, ""
	case "tools/list":
		return Object{{"tools", s.toolList()}}, 0, ""
	case "tools/call":
		return s.callTool(req.Params)
	case "ping":
		return Object{}, 0, ""
	default:
		// Everything else, `server/discover` included: a client uses the shape
		// of this refusal to work out which revision we speak.
		return nil, codeMethodNotFound, fmt.Sprintf("this server has tools and nothing else — %q is not one of its methods", req.Method)
	}
}

func (s *Server) initialize(params json.RawMessage) any {
	var p struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	json.Unmarshal(params, &p) //nolint:errcheck // an unreadable version is the same as none

	version := ProtocolVersion
	if p.ProtocolVersion != "" && p.ProtocolVersion <= ProtocolVersion {
		version = p.ProtocolVersion
	}

	return Object{
		{"protocolVersion", version},
		{"capabilities", Object{{"tools", Object{}}}},
		{"serverInfo", Object{{"name", s.name}, {"version", s.version}}},
	}
}

func (s *Server) toolList() []Object {
	out := make([]Object, 0, len(s.tools))
	for _, t := range s.tools {
		tool := Object{
			{"name", t.Name},
			{"title", t.Title},
			{"description", t.Description},
			{"inputSchema", t.InputSchema},
		}
		if t.OutputSchema != nil {
			tool = append(tool, Member{"outputSchema", t.OutputSchema})
		}
		out = append(out, tool)
	}
	return out
}

func (s *Server) callTool(params json.RawMessage) (any, int, string) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, codeInvalidParams, fmt.Sprintf("could not read the call: %v", err)
	}

	for _, t := range s.tools {
		if t.Name != p.Name {
			continue
		}
		result, err := t.Call(p.Arguments)
		if err != nil {
			return nil, codeInternal, err.Error()
		}
		out := Object{
			{"content", []Object{{{"type", "text"}, {"text", result.Text}}}},
			{"isError", result.IsError},
		}
		if result.Structured != nil {
			out = append(out, Member{"structuredContent", result.Structured})
		}
		return out, 0, ""
	}

	names := make([]string, 0, len(s.tools))
	for _, t := range s.tools {
		names = append(names, t.Name)
	}
	sort.Strings(names)
	return nil, codeInvalidParams, fmt.Sprintf("no tool called %q — this kitchen has %s", p.Name, strings.Join(names, ", "))
}

func writeResult(w http.ResponseWriter, id json.RawMessage, result any) {
	write(w, Object{{"jsonrpc", "2.0"}, {"id", json.RawMessage(id)}, {"result", result}})
}

func writeError(w http.ResponseWriter, id json.RawMessage, code int, message string) {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	write(w, Object{
		{"jsonrpc", "2.0"},
		{"id", json.RawMessage(id)},
		{"error", Object{{"code", code}, {"message", message}}},
	})
}

func write(w http.ResponseWriter, body Object) {
	w.Header().Set("Content-Type", "application/json")
	// Every JSON-RPC response is HTTP 200: the error, when there is one, is in
	// the body. A status code here would be about the transport.
	json.NewEncoder(w).Encode(body) //nolint:errcheck // the client hung up
}
