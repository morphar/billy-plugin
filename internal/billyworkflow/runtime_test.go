package billyworkflow

import (
	"context"
	"fmt"
	"testing"

	"github.com/cortexium-io/api-mcp/apimcp"
)

type readCall struct {
	Profile   string
	Operation string
	Input     apimcp.ToolInput
}

type fakeRuntime struct {
	profile    apimcp.APIProfile
	read       func(context.Context, string, string, apimcp.ToolInput) (apimcp.APIResponse, error)
	batch      func(context.Context, apimcp.ExtensionBatchRequest) ([]apimcp.ExtensionBatchResult, error)
	readCalls  []readCall
	batchCalls []apimcp.ExtensionBatchRequest
}

func (f *fakeRuntime) ResolveProfile(profileID string) (apimcp.APIProfile, error) {
	if f.profile.ID == "" {
		f.profile = apimcp.APIProfile{ID: "techbase", Name: "Techbase"}
	}
	if profileID != "" && profileID != f.profile.ID {
		return apimcp.APIProfile{}, fmt.Errorf("unknown profile %q", profileID)
	}
	return f.profile, nil
}

func (f *fakeRuntime) ExecuteRead(ctx context.Context, profileID, operation string, input apimcp.ToolInput) (apimcp.APIResponse, error) {
	f.readCalls = append(f.readCalls, readCall{Profile: profileID, Operation: operation, Input: input})
	if f.read == nil {
		return apimcp.APIResponse{}, fmt.Errorf("unexpected read operation %q", operation)
	}
	return f.read(ctx, profileID, operation, input)
}

func (f *fakeRuntime) ExecuteBatch(ctx context.Context, request apimcp.ExtensionBatchRequest) ([]apimcp.ExtensionBatchResult, error) {
	f.batchCalls = append(f.batchCalls, request)
	if f.batch == nil {
		return nil, fmt.Errorf("unexpected batch call")
	}
	return f.batch(ctx, request)
}

func pagedResponse(root string, records []map[string]any, page, pageCount, total int) apimcp.APIResponse {
	items := make([]any, len(records))
	for i := range records {
		items[i] = records[i]
	}
	return apimcp.APIResponse{
		OK:         true,
		Status:     200,
		StatusText: "200 OK",
		Body: map[string]any{
			root: items,
			"meta": map[string]any{
				"paging": map[string]any{
					"page":      page,
					"pageCount": pageCount,
					"pageSize":  1000,
					"total":     total,
				},
			},
		},
		Pagination: &apimcp.PaginationInfo{
			Source:    "meta.paging",
			Page:      page,
			PageCount: pageCount,
			PageSize:  1000,
			Total:     total,
			FinalPage: page >= pageCount,
		},
	}
}

func objectResponse(root string, record map[string]any) apimcp.APIResponse {
	return apimcp.APIResponse{
		OK:         true,
		Status:     200,
		StatusText: "200 OK",
		Body:       map[string]any{root: record},
	}
}

func queryInt(t *testing.T, input apimcp.ToolInput, key string) int {
	t.Helper()
	value, ok := input.Query[key].(int)
	if !ok {
		t.Fatalf("query %q = %#v, want int", key, input.Query[key])
	}
	return value
}
