// Package api is the REST API the shared React front end calls. It talks only to
// Temporal and Redis.
package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"

	"github.com/temporal-sa/adk-go-wealth-management/internal/chaos"
	"github.com/temporal-sa/adk-go-wealth-management/internal/domain"
	"github.com/temporal-sa/adk-go-wealth-management/internal/eventstream"
	"github.com/temporal-sa/adk-go-wealth-management/internal/workflows"
)

// ChaosStore is the chaos flags and failure log, in Redis.
type ChaosStore interface {
	SetEnabled(ctx context.Context, f chaos.Flag, enabled bool) error
	GetFlags(ctx context.Context) (map[chaos.Flag]bool, error)
	GetEvents(ctx context.Context, limit int64) ([]chaos.Event, error)
	ClearEvents(ctx context.Context) error
}

// Server serves the API.
type Server struct {
	Temporal  client.Client
	TaskQueue string
	Events    eventstream.Stream
	Chaos     ChaosStore
}

// allowedOrigins are where the React development server runs.
var allowedOrigins = map[string]bool{"http://localhost:3000": true, "http://127.0.0.1:3000": true}

// Handler returns the API's HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"message": "Google ADK (Go) + Temporal Agent!"})
	})
	mux.HandleFunc("POST /start-workflow", s.startWorkflow)
	mux.HandleFunc("POST /send-prompt", s.sendPrompt)
	mux.HandleFunc("GET /get-chat-history", s.getChatHistory)
	mux.HandleFunc("POST /end-chat", s.endChat)
	mux.HandleFunc("GET /chaos", s.getChaos)
	mux.HandleFunc("POST /chaos", s.setChaos)
	mux.HandleFunc("POST /chaos/reset", s.resetChaos)
	return cors(mux)
}

func (s *Server) startWorkflow(w http.ResponseWriter, r *http.Request) {
	id, ok := required(w, r, "workflow_id")
	if !ok {
		return
	}
	_, err := s.Temporal.ExecuteWorkflow(r.Context(), client.StartWorkflowOptions{
		ID:                    id,
		TaskQueue:             s.TaskQueue,
		WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
	}, workflows.WealthManagementWorkflowName)
	if err != nil {
		log.Printf("Exception occurred starting workflow %v", err)
		writeJSON(w, http.StatusOK, map[string]string{"message": "An error occurred starting the workflow " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Workflow started."})
}

func (s *Server) sendPrompt(w http.ResponseWriter, r *http.Request) {
	id, ok := required(w, r, "workflow_id")
	if !ok {
		return
	}
	prompt, ok := required(w, r, "prompt")
	if !ok {
		return
	}
	response := "Message sent"
	if err := s.Temporal.SignalWorkflow(r.Context(), id, "", workflows.SignalProcessUserMessage,
		domain.ProcessUserMessageInput{UserInput: prompt}); err != nil {
		response = "Error: " + err.Error()
	}
	writeJSON(w, http.StatusOK, map[string]string{"response": response})
}

func (s *Server) getChatHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := required(w, r, "workflow_id")
	if !ok {
		return
	}
	from, _ := strconv.ParseInt(r.URL.Query().Get("from_index"), 10, 64)
	events, err := s.Events.EventsFrom(r.Context(), id, from)
	if err != nil {
		log.Printf("Redis error retrieving chat history: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"detail": "Internal server error while querying workflow. " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) endChat(w http.ResponseWriter, r *http.Request) {
	id, ok := required(w, r, "workflow_id")
	if !ok {
		return
	}
	if err := s.Temporal.SignalWorkflow(r.Context(), id, "", workflows.SignalEndWorkflow, nil); err != nil {
		// Workflow not found: nothing to end.
		log.Println(err)
		writeJSON(w, http.StatusOK, map[string]string{})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "End chat signal sent."})
}

type flagJSON struct {
	Flag        chaos.Flag `json:"flag"`
	Label       string     `json:"label"`
	Description string     `json:"description"`
	Enabled     bool       `json:"enabled"`
}

func (s *Server) getChaos(w http.ResponseWriter, r *http.Request) {
	enabled, err := s.Chaos.GetFlags(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
		return
	}
	events, err := s.Chaos.GetEvents(r.Context(), 20)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
		return
	}
	flags := []flagJSON{}
	for _, info := range chaos.Flags {
		flags = append(flags, flagJSON{info.Flag, info.Label, info.Description, enabled[info.Flag]})
	}
	writeJSON(w, http.StatusOK, map[string]any{"flags": flags, "events": events})
}

func (s *Server) setChaos(w http.ResponseWriter, r *http.Request) {
	f, ok := required(w, r, "flag")
	if !ok {
		return
	}
	flag := chaos.Flag(f)
	if !chaos.IsKnown(flag) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Unknown chaos flag: " + f})
		return
	}
	enabled, err := strconv.ParseBool(r.URL.Query().Get("enabled"))
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "enabled must be true or false"})
		return
	}
	if err := s.Chaos.SetEnabled(r.Context(), flag, enabled); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"flag": flag, "enabled": enabled})
}

func (s *Server) resetChaos(w http.ResponseWriter, r *http.Request) {
	for _, info := range chaos.Flags {
		if err := s.Chaos.SetEnabled(r.Context(), info.Flag, false); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
			return
		}
	}
	if err := s.Chaos.ClearEvents(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "All failure injection turned off."})
}

// required returns a required query parameter, or writes a 422 like FastAPI does.
func required(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	v := r.URL.Query().Get(name)
	if v == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "missing query parameter " + name})
		return "", false
	}
	return v, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				if h := r.Header.Get("Access-Control-Request-Headers"); h != "" {
					w.Header().Set("Access-Control-Allow-Headers", h)
				}
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
