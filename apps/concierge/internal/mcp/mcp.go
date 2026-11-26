// Package mcp is a small Model Context Protocol server over Streamable HTTP.
//
// It implements the handshake era of the protocol — `initialize`, then
// `tools/list` and `tools/call` — which is what every deployed MCP client
// speaks today. There is no session: this server keeps nothing between calls,
// so there is nothing for a session id to identify, and a client that pins one
// is simply never given one.
//
// Only tools are served. Resources and prompts are the other two halves of MCP
// and neither has anything to say about a cinema.
package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// The revision this server answers with. A client that asks for a different
// one is answered in this one, which is what the specification asks of a
// server that cannot speak what was requested.
const ProtocolVersion = "2025-06-18"

// JSON-RPC error codes. Only the three a server actually produces.
const (
	codeParse          = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
)

// Tool is one tool this server exposes. The schemas are raw JSON so they read
// in the source as the JSON Schema documents they are.
type Tool struct {
	Name        string          `json:"name"`
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
	// OutputSchema is what makes a result structured rather than a wall of
	// text: a client that reads it can hand the caller typed fields.
	OutputSchema json.RawMessage `json:"outputSchema,omitempty"`
	Annotations  *Annotations    `json:"annotations,omitempty"`

	// Call runs the tool. Returning an error whose text is meant for the
	// caller is what ToolError is for; anything else is a fault.
	Call func(arguments json.RawMessage) (Result, error) `json:"-"`
}

// Annotations are the server's hints about what a tool does. They are advice a
// client displays, never a permission.
type Annotations struct {
	ReadOnlyHint    *bool `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	IdempotentHint  *bool `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool `json:"openWorldHint,omitempty"`
}

func ReadOnly() *Annotations {
	yes, no := true, false
	return &Annotations{ReadOnlyHint: &yes, DestructiveHint: &no, OpenWorldHint: &no}
}

// Result is what a tool returns: a line of prose for a human reading the
// transcript, and the same answer as data for the script that asked.
type Result struct {
	Text       string
	Structured any
}

// ToolError is a failure the tool itself reports — a show that isn't in the
// programme, a house with no room left. It comes back as a successful call
// carrying `isError`, which is what the protocol asks for: the call worked,
// the tool has bad news.
type ToolError struct{ Message string }

func (e *ToolError) Error() string { return e.Message }

func Failf(format string, args ...any) error {
	return &ToolError{Message: fmt.Sprintf(format, args...)}
}

// Server serves one set of tools.
type Server struct {
	name         string
	version      string
	instructions string
	tools        []Tool
	byName       map[string]*Tool
}

func NewServer(name, version, instructions string, tools []Tool) *Server {
	s := &Server{name: name, version: version, instructions: instructions, tools: tools, byName: map[string]*Tool{}}
	for i := range s.tools {
		s.byName[s.tools[i].Name] = &s.tools[i]
	}
	return s
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// The transport also defines GET, for a server that pushes to the client.
	// This one only answers what it is asked.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "this MCP endpoint only accepts POST", http.StatusMethodNotAllowed)
		return
	}

	var req request
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, nil, codeParse, "the request is not JSON")
		return
	}

	// A message with no id is a notification: nothing is expected back, and
	// `notifications/initialized` is the only one this server is ever sent.
	if len(req.ID) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	slog.Info("mcp", "method", req.Method)

	switch req.Method {
	case "initialize":
		writeResult(w, req.ID, map[string]any{
			"protocolVersion": ProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": s.name, "version": s.version},
			"instructions":    s.instructions,
		})
	case "ping":
		writeResult(w, req.ID, map[string]any{})
	case "tools/list":
		writeResult(w, req.ID, map[string]any{"tools": s.tools})
	case "tools/call":
		s.call(w, req)
	default:
		// `server/discover` lands here too. A client that probes for the
		// revision without a handshake reads this as "not that era" and opens
		// with `initialize` instead, which is exactly right.
		writeError(w, req.ID, codeMethodNotFound, fmt.Sprintf("this server does not serve %q", req.Method))
	}
}

func (s *Server) call(w http.ResponseWriter, req request) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeError(w, req.ID, codeInvalidParams, "the call has no name and arguments")
		return
	}
	tool := s.byName[params.Name]
	if tool == nil {
		writeError(w, req.ID, codeInvalidParams, fmt.Sprintf("there is no tool called %q", params.Name))
		return
	}

	result, err := tool.Call(params.Arguments)
	if err != nil {
		var reported *ToolError
		if !errors.As(err, &reported) {
			slog.Error("tool failed", "tool", params.Name, "error", err)
			writeError(w, req.ID, codeInvalidRequest, err.Error())
			return
		}
		// The tool's own bad news: a successful call that says so.
		writeResult(w, req.ID, map[string]any{
			"content": []any{textContent(reported.Message)},
			"isError": true,
		})
		return
	}

	payload := map[string]any{"content": []any{textContent(result.Text)}}
	if result.Structured != nil {
		payload["structuredContent"] = result.Structured
	}
	writeResult(w, req.ID, payload)
}

func textContent(text string) map[string]any {
	return map[string]any{"type": "text", "text": text}
}

func writeResult(w http.ResponseWriter, id json.RawMessage, result any) {
	write(w, response{JSONRPC: "2.0", ID: id, Result: result})
}

func writeError(w http.ResponseWriter, id json.RawMessage, code int, message string) {
	write(w, response{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}})
}

// write always answers 200. A JSON-RPC error is a successful exchange carrying
// a failure, and a 4xx over one is what makes a client report the transport
// instead of the message.
func write(w http.ResponseWriter, payload response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("writing response", "error", err)
	}
}
