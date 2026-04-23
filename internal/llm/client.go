// Gemini client wrapper for concurrent API calls
// Initialize the client with calls limit, generate a single-turn completion, return the response text.
package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Gemini SDK with semaphore
type Client struct {
	inner *genai.Client
	model string
	sem   chan struct{}
}

// Initialize NewClient and maximum concurrent calls
func NewClient(apiKey, model string, maxConcurrent int) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set")
	}
	gc, err := genai.NewClient(context.Background(), option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("genai client: %w", err)
	}
	return &Client{
		inner: gc,
		model: model,
		sem:   make(chan struct{}, maxConcurrent),
	}, nil
}

// Close releases the underlying Gemini HTTP client.
func (c *Client) Close() {
	_ = c.inner.Close()
}

// Acquire blocks until a semaphore slot is available or the context is cancelled.
func (c *Client) acquire(ctx context.Context) error {
	select {
	case c.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Returns a semaphore slot.
func (c *Client) release() {
	<-c.sem
}

// Generate a single-turn completion with system prompt and user prompt, return the response text
func (c *Client) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if err := c.acquire(ctx); err != nil {
		return "", err
	}
	defer c.release()

	m := c.inner.GenerativeModel(c.model)
	m.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}

	resp, err := m.GenerateContent(ctx, genai.Text(userPrompt))
	if err != nil {
		return "", fmt.Errorf("gemini generate: %w", err)
	}
	return responseText(resp), nil
}

// Creates a GenerativeModel with the client's configured name and optional tools
func (c *Client) newModel(tools ...*genai.Tool) *genai.GenerativeModel {
	m := c.inner.GenerativeModel(c.model)
	if len(tools) > 0 {
		m.Tools = tools
	}
	return m
}

// Extracts all text parts from the first candidate.
func responseText(resp *genai.GenerateContentResponse) string {
	var sb strings.Builder
	for _, cand := range resp.Candidates {
		if cand.Content == nil {
			continue
		}
		for _, part := range cand.Content.Parts {
			if t, ok := part.(genai.Text); ok {
				sb.WriteString(string(t))
			}
		}
	}
	return sb.String()
}

// Strips markdown code fences
func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	if after, ok := strings.CutPrefix(s, "```json"); ok {
		s = after
	} else if after, ok := strings.CutPrefix(s, "```"); ok {
		s = after
	}
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
