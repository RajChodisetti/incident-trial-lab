package investigation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCP owns a dedicated stdio process. There are no caller-supplied command/path
// arguments and no shared-session cancellation or persistent grants.
type MCP struct {
	session *mcp.ClientSession
	schemas map[string]*jsonschema.Resolved
}

func Connect(ctx context.Context) (*MCP, error) {
	cmd := exec.Command("./.build/trial-mcp", "-workers", "1")
	cmd.Stderr = os.Stderr
	client := mcp.NewClient(&mcp.Implementation{Name: "offline-investigation-evaluation", Version: "0.1.0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd, TerminateDuration: 90 * time.Second}, nil)
	if err != nil {
		return nil, err
	}
	m := &MCP{session: session, schemas: map[string]*jsonschema.Resolved{}}
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		session.Close()
		return nil, err
	}
	allowed := map[string]bool{"list_trials": true, "list_scenarios": true, "create_experiment": true, "get_trial": true, "get_evidence": true, "start_trial": true, "cancel_trial": true, "compare_trials": true}
	for _, t := range tools.Tools {
		if !allowed[t.Name] {
			continue
		}
		raw, _ := json.Marshal(t.OutputSchema)
		var schema jsonschema.Schema
		if err = json.Unmarshal(raw, &schema); err != nil {
			session.Close()
			return nil, err
		}
		m.schemas[t.Name], err = schema.Resolve(nil)
		if err != nil {
			session.Close()
			return nil, err
		}
	}
	if len(m.schemas) != len(allowed) {
		session.Close()
		return nil, fmt.Errorf("required typed tools missing")
	}
	return m, nil
}
func (m *MCP) Call(ctx context.Context, name string, args map[string]any, out any) error {
	schema := m.schemas[name]
	if schema == nil {
		return fmt.Errorf("tool not allowlisted")
	}
	r, err := m.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return err
	}
	if r.IsError {
		return fmt.Errorf("%s refused: %v", name, r.Content)
	}
	raw, err := json.Marshal(r.StructuredContent)
	if err != nil {
		return err
	}
	var v any
	if err = json.Unmarshal(raw, &v); err != nil {
		return err
	}
	if err = schema.Validate(v); err != nil {
		return fmt.Errorf("output schema mismatch: %w", err)
	}
	return json.Unmarshal(raw, out)
}
func (m *MCP) Close() error { return m.session.Close() }
