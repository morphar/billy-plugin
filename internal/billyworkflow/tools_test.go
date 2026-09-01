package billyworkflow

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/cortexium-io/api-mcp/apimcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestPackagedConfigRegistersAllFixedWorkflowTools(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	configPath, err := filepath.Abs(filepath.Join("..", "..", "plugins", "billy-plugin", "config", "billy.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := apimcp.LoadConfigFromArgs([]string{"--config", configPath})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server, err := apimcp.CreateServer(ctx, cfg, Tools()...)
	if err != nil {
		t.Fatal(err)
	}
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	errCh := make(chan error, 1)
	go func() { errCh <- server.Run(ctx, serverTransport) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "billy-workflow-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	toolsByName := map[string]*mcp.Tool{}
	for _, tool := range result.Tools {
		toolsByName[tool.Name] = tool
	}
	for _, name := range []string{
		"api_mcp_diagnostics",
		"diagnose_billy_connection",
		"review_billy_bank_account",
		"review_billy_document_coverage",
		"review_billy_vat_period",
		"review_billy_foreign_currency_purchase",
		"preview_billy_bank_line_cleanup",
		"commit_billy_bank_line_cleanup",
	} {
		if toolsByName[name] == nil {
			t.Fatalf("packaged config omitted tool %q", name)
		}
	}
	commit := toolsByName["commit_billy_bank_line_cleanup"]
	if commit.Annotations == nil || commit.Annotations.DestructiveHint == nil || !*commit.Annotations.DestructiveHint || commit.Annotations.ReadOnlyHint {
		t.Fatalf("commit annotations = %+v", commit.Annotations)
	}
	if schemaRequiresQueryField(t, toolsByName["list_account_groups"], "daybookId") {
		t.Fatal("list_account_groups must not require daybookId")
	}
	if !schemaRequiresQueryField(t, toolsByName["list_daybook_balance_accounts"], "daybookId") {
		t.Fatal("list_daybook_balance_accounts must require daybookId")
	}

	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func schemaRequiresQueryField(t *testing.T, tool *mcp.Tool, field string) bool {
	t.Helper()
	if tool == nil {
		t.Fatal("tool is nil")
	}
	data, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	properties, _ := schema["properties"].(map[string]any)
	query, _ := properties["query"].(map[string]any)
	required, _ := query["required"].([]any)
	for _, value := range required {
		if value == field {
			return true
		}
	}
	return false
}
