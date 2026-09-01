package billyworkflow

import (
	"context"
	"testing"

	"github.com/cortexium-io/api-mcp/apimcp"
)

func TestFetchAllRetriesTruncationAndUsesProvenPagination(t *testing.T) {
	rt := &fakeRuntime{}
	rt.read = func(_ context.Context, _ string, operation string, input apimcp.ToolInput) (apimcp.APIResponse, error) {
		if operation != "list_things" {
			t.Fatalf("operation = %q", operation)
		}
		pageSize := queryInt(t, input, "pageSize")
		page := queryInt(t, input, "page")
		if pageSize == initialPageSize {
			return apimcp.APIResponse{Truncated: true}, nil
		}
		if pageSize != initialPageSize/2 {
			t.Fatalf("pageSize = %d, want %d", pageSize, initialPageSize/2)
		}
		if page == 1 {
			return pagedResponse("things", []map[string]any{{"id": "one"}}, 1, 2, 2), nil
		}
		return pagedResponse("things", []map[string]any{{"id": "two"}}, 2, 2, 2), nil
	}

	records, info, err := fetchAll(context.Background(), rt, "techbase", "list_things", "things", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || !info.Complete || info.PagesFetched != 2 || info.TruncationRetries != 1 {
		t.Fatalf("records/info = %d/%+v", len(records), info)
	}
	if info.PageSize != initialPageSize/2 || !info.ReportedTotalPresent || info.ReportedTotal != 2 {
		t.Fatalf("pagination evidence = %+v", info)
	}
}

func TestFetchAllDoesNotInferCompletenessFromShortPage(t *testing.T) {
	rt := &fakeRuntime{
		read: func(_ context.Context, _ string, _ string, _ apimcp.ToolInput) (apimcp.APIResponse, error) {
			return apimcp.APIResponse{
				OK:         true,
				Status:     200,
				StatusText: "200 OK",
				Body:       map[string]any{"things": []any{map[string]any{"id": "one"}}},
			}, nil
		},
	}

	records, info, err := fetchAll(context.Background(), rt, "techbase", "list_things", "things", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || info.Complete || info.IncompleteReason == "" {
		t.Fatalf("records/info = %d/%+v", len(records), info)
	}
	if len(rt.readCalls) != 1 {
		t.Fatalf("read calls = %d, want 1", len(rt.readCalls))
	}
}

func TestFetchAllRejectsInconsistentReportedTotal(t *testing.T) {
	rt := &fakeRuntime{
		read: func(_ context.Context, _ string, _ string, _ apimcp.ToolInput) (apimcp.APIResponse, error) {
			return pagedResponse("things", []map[string]any{{"id": "one"}}, 1, 1, 2), nil
		},
	}

	_, info, err := fetchAll(context.Background(), rt, "techbase", "list_things", "things", nil)
	if err != nil {
		t.Fatal(err)
	}
	if info.Complete || info.ReportedTotal != 2 || info.RecordsFetched != 1 {
		t.Fatalf("info = %+v", info)
	}
}
