// Package config reads the demo's settings from the environment, once, at process start.
//
// Workflow code must not read the environment itself, so the worker reads it here and
// hands the values to the workflows when it registers them.
package config

import (
	"os"
	"strconv"
	"strings"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/envconfig"
)

// DefaultGeminiModel matches the Python ADK port's default.
const DefaultGeminiModel = "gemini-3.8-flash"

// Config holds the task queues, the model and the Temporal connection settings.
type Config struct {
	// TaskQueue runs the chat workflow and the tool activities (TEMPORAL_TASK_QUEUE).
	TaskQueue string
	// TaskQueueOpenAccount runs OpenInvestmentAccountWorkflow (TEMPORAL_TASK_QUEUE_OPEN_ACCOUNT).
	// It defaults to TaskQueue; set it to run the workflow on another worker.
	TaskQueueOpenAccount string
	// TaskQueueLLM runs the model calls, so they can be rate limited on their own
	// (TEMPORAL_TASK_QUEUE_LLM, default <TaskQueue>-llm).
	TaskQueueLLM string
	// GeminiModel is the Gemini model every agent uses (GEMINI_MODEL).
	GeminiModel string
	// LLMMaxCallsPerSecond, when above zero, has the Temporal service limit the rate of
	// model calls across all workers (LLM_MAX_CALLS_PER_SECOND).
	LLMMaxCallsPerSecond float64
}

// Load reads the configuration from the environment.
func Load() Config {
	taskQueue := getenv("TEMPORAL_TASK_QUEUE", "Supervisor")
	c := Config{
		TaskQueue:            taskQueue,
		TaskQueueOpenAccount: getenv("TEMPORAL_TASK_QUEUE_OPEN_ACCOUNT", taskQueue),
		TaskQueueLLM:         getenv("TEMPORAL_TASK_QUEUE_LLM", taskQueue+"-llm"),
		GeminiModel:          getenv("GEMINI_MODEL", DefaultGeminiModel),
	}
	if v := os.Getenv("LLM_MAX_CALLS_PER_SECOND"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.LLMMaxCallsPerSecond = f
		}
	}
	return c
}

// ClientOptions returns the Temporal client options from the environment (TEMPORAL_ADDRESS,
// TEMPORAL_NAMESPACE, TLS settings, or a temporal.toml profile), defaulting to a local
// development server at 127.0.0.1:7233.
func ClientOptions() (client.Options, error) {
	opts, err := envconfig.LoadDefaultClientOptions()
	if err != nil {
		return client.Options{}, err
	}
	if opts.HostPort == "" {
		opts.HostPort = "127.0.0.1:7233"
	}
	return opts, nil
}

// Bool reports whether an environment variable is set to true, yes or 1.
func Bool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "true", "t", "yes", "y", "1":
		return true
	}
	return false
}

func getenv(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
