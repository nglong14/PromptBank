// Function calling: let the LLM iteratively refine the assets based on the user feedback
package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/nglong14/PromptBank/internal/asset"
)

const maxToolCalls = 8

// RefineMessage represents one turn in the refinement conversation history.
type RefineMessage struct {
	Role    string `json:"role"`    // "user" or "agent"
	Content string `json:"content"`
}

// RefineRequest is the input to the refinement agent.
type RefineRequest struct {
	Assets         asset.Assets    `json:"assets"`
	ComposedOutput string          `json:"composedOutput"`
	UserFeedback   string          `json:"userFeedback"`
	History        []RefineMessage `json:"history"`
}

// RefineResponse is the output of the refinement agent.
type RefineResponse struct {
	UpdatedAssets asset.Assets `json:"updatedAssets"`
	Explanation   string       `json:"explanation"`
	ChangedFields []string     `json:"changedFields"`
}

const refinerSystemPrompt = `You are a prompt engineering assistant helping users iteratively improve their AI prompts.
You will be given the current prompt assets, the composed prompt output, and feedback from the user.

Use the tools to make surgical, targeted edits to the prompt assets based on the feedback.
When you are satisfied that all requested changes have been made, call explain_change to finish.

Guidelines:
- Only change what the user explicitly asked for — do not make unrequested edits.
- Make one targeted change at a time; you can make multiple tool calls.
- Always call explain_change when done, even if no changes were needed.`

// refinerTools defines the four tools the agent can call.
var refinerTools = []*genai.Tool{{
	FunctionDeclarations: []*genai.FunctionDeclaration{
		{
			Name:        "update_field",
			Description: "Replace the value of one asset field (persona, context, tone, constraints, or goal).",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"field": {
						Type:        genai.TypeString,
						Description: "One of: persona, context, tone, constraints, goal",
					},
					"value": {
						Type:        genai.TypeString,
						Description: "The new polished value for the field",
					},
				},
				Required: []string{"field", "value"},
			},
		},
		{
			Name:        "add_example",
			Description: "Append a new input/output example pair to the examples list.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"input":  {Type: genai.TypeString, Description: "The example input text"},
					"output": {Type: genai.TypeString, Description: "The expected output text"},
				},
				Required: []string{"input", "output"},
			},
		},
		{
			Name:        "remove_example",
			Description: "Remove an existing example by its zero-based index.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"index": {Type: genai.TypeInteger, Description: "Zero-based index of the example to remove"},
				},
				Required: []string{"index"},
			},
		},
		{
			Name:        "explain_change",
			Description: "Signal that all edits are complete. Call this when done.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"explanation": {
						Type:        genai.TypeString,
						Description: "1-2 sentences describing what was changed and why",
					},
					"changedFields": {
						Type:        genai.TypeArray,
						Items:       &genai.Schema{Type: genai.TypeString},
						Description: "List of asset field names that were modified",
					},
				},
				Required: []string{"explanation", "changedFields"},
			},
		},
	},
}}

// Refine runs the iterative refinement agent. It uses Gemini's function-calling
// in a loop (up to maxToolCalls iterations) to make surgical edits to the assets
// based on userFeedback.
//
// Concurrency: the semaphore is acquired and released per Gemini call, not for
// the entire loop, so other users are not blocked between tool invocations.
// The ctx is checked between every iteration to detect client disconnects early.
func (c *Client) Refine(ctx context.Context, req RefineRequest) (RefineResponse, error) {
	// Local snapshot — mutated by tool calls; never stored on Client.
	snapshot := req.Assets

	m := c.newModel(refinerTools...)
	m.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(refinerSystemPrompt)},
	}

	cs := m.StartChat()
	cs.History = buildChatHistory(req.History)

	// Initial user message includes full asset state + feedback.
	initialMsg := buildRefineUserMsg(snapshot, req.ComposedOutput, req.UserFeedback)

	// Send the initial message.
	if err := c.acquire(ctx); err != nil {
		return RefineResponse{}, err
	}
	resp, err := cs.SendMessage(ctx, genai.Text(initialMsg))
	c.release()
	if err != nil {
		return RefineResponse{}, fmt.Errorf("refine initial message: %w", err)
	}

	var explanation string
	var changedFields []string
	toolCallCount := 0

	for {
		// Check for client disconnect between every iteration.
		select {
		case <-ctx.Done():
			return RefineResponse{}, ctx.Err()
		default:
		}

		funcCalls := extractFuncCalls(resp)
		if len(funcCalls) == 0 {
			// No tool calls — model responded with text only; treat as done.
			if explanation == "" {
				explanation = "No changes were needed based on your feedback."
			}
			break
		}

		// Build function responses and execute mutations.
		var funcResponses []genai.Part
		done := false

		for _, fc := range funcCalls {
			toolCallCount++

			if fc.Name == "explain_change" {
				explanation = argString(fc.Args, "explanation")
				changedFields = argStringSlice(fc.Args, "changedFields")
				done = true
			} else {
				if err := applyToolCall(fc, &snapshot); err != nil {
					return RefineResponse{}, fmt.Errorf("tool %q: %w", fc.Name, err)
				}
			}

			funcResponses = append(funcResponses, genai.FunctionResponse{
				Name:     fc.Name,
				Response: map[string]any{"status": "ok"},
			})
		}

		if done || toolCallCount >= maxToolCalls {
			break
		}

		// Send function responses and get the next model turn.
		if err := c.acquire(ctx); err != nil {
			return RefineResponse{}, err
		}
		resp, err = cs.SendMessage(ctx, funcResponses...)
		c.release()
		if err != nil {
			return RefineResponse{}, fmt.Errorf("refine tool response: %w", err)
		}
	}

	return RefineResponse{
		UpdatedAssets: snapshot,
		Explanation:   explanation,
		ChangedFields: changedFields,
	}, nil
}

// applyToolCall mutates the assets snapshot according to the function call.
func applyToolCall(fc genai.FunctionCall, a *asset.Assets) error {
	switch fc.Name {
	case "update_field":
		field := argString(fc.Args, "field")
		value := argString(fc.Args, "value")
		switch field {
		case "persona":
			a.Persona = value
		case "context":
			a.Context = value
		case "tone":
			a.Tone = value
		case "constraints":
			a.Constraints = value
		case "goal":
			a.Goal = value
		default:
			return fmt.Errorf("unknown field: %q", field)
		}

	case "add_example":
		a.Examples = append(a.Examples, asset.Example{
			Input:  argString(fc.Args, "input"),
			Output: argString(fc.Args, "output"),
		})

	case "remove_example":
		idx := argInt(fc.Args, "index")
		if idx < 0 || idx >= len(a.Examples) {
			return fmt.Errorf("remove_example: index %d out of range (len %d)", idx, len(a.Examples))
		}
		a.Examples = append(a.Examples[:idx], a.Examples[idx+1:]...)

	default:
		return fmt.Errorf("unknown tool: %q", fc.Name)
	}
	return nil
}

// extractFuncCalls collects all FunctionCall parts from the first candidate.
func extractFuncCalls(resp *genai.GenerateContentResponse) []genai.FunctionCall {
	var calls []genai.FunctionCall
	for _, cand := range resp.Candidates {
		if cand.Content == nil {
			continue
		}
		for _, part := range cand.Content.Parts {
			if fc, ok := part.(genai.FunctionCall); ok {
				calls = append(calls, fc)
			}
		}
	}
	return calls
}

// buildChatHistory converts the frontend RefineMessage history into Gemini
// Content entries. Only text turns are included (not internal tool-call details).
func buildChatHistory(history []RefineMessage) []*genai.Content {
	var contents []*genai.Content
	for _, msg := range history {
		role := "user"
		if msg.Role == "agent" {
			role = "model"
		}
		contents = append(contents, &genai.Content{
			Role:  role,
			Parts: []genai.Part{genai.Text(msg.Content)},
		})
	}
	return contents
}

// buildRefineUserMsg constructs the user message for the agent, including a
// full snapshot of the current asset state.
func buildRefineUserMsg(a asset.Assets, composedOutput, feedback string) string {
	var sb strings.Builder
	sb.WriteString("Here are the current prompt assets:\n\n")
	fmt.Fprintf(&sb, "- Goal: %s\n", orEmpty(a.Goal))
	fmt.Fprintf(&sb, "- Persona: %s\n", orEmpty(a.Persona))
	fmt.Fprintf(&sb, "- Context: %s\n", orEmpty(a.Context))
	fmt.Fprintf(&sb, "- Tone: %s\n", orEmpty(a.Tone))
	fmt.Fprintf(&sb, "- Constraints: %s\n", orEmpty(a.Constraints))
	if len(a.Examples) > 0 {
		fmt.Fprintf(&sb, "- Examples (%d):\n", len(a.Examples))
		for i, ex := range a.Examples {
			fmt.Fprintf(&sb, "  [%d] Input: %s | Output: %s\n", i, ex.Input, ex.Output)
		}
	} else {
		sb.WriteString("- Examples: (none)\n")
	}

	sb.WriteString("\nThe composed prompt is:\n---\n")
	sb.WriteString(composedOutput)
	sb.WriteString("\n---\n\n")
	fmt.Fprintf(&sb, "User feedback: %s\n\n", feedback)
	sb.WriteString("Please use the tools to make the requested changes, then call explain_change when done.")
	return sb.String()
}

func orEmpty(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(not set)"
	}
	return s
}

// argString safely extracts a string from a function-call args map.
func argString(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// argInt safely extracts an integer from a function-call args map.
// JSON numbers arrive as float64.
func argInt(args map[string]any, key string) int {
	v, ok := args[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

// argStringSlice safely extracts a []string from a function-call args map.
func argStringSlice(args map[string]any, key string) []string {
	v, ok := args[key]
	if !ok {
		return nil
	}
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
