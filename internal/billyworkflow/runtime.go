package billyworkflow

import (
	"context"

	"github.com/cortexium-io/api-mcp/apimcp"
)

type runtime interface {
	ResolveProfile(string) (apimcp.APIProfile, error)
	ExecuteRead(context.Context, string, string, apimcp.ToolInput) (apimcp.APIResponse, error)
	ExecuteBatch(context.Context, apimcp.ExtensionBatchRequest) ([]apimcp.ExtensionBatchResult, error)
}

type extensionRuntime struct {
	runtime *apimcp.ExtensionRuntime
}

func (r extensionRuntime) ResolveProfile(profileID string) (apimcp.APIProfile, error) {
	return r.runtime.ResolveProfile(profileID)
}

func (r extensionRuntime) ExecuteRead(ctx context.Context, profileID, operation string, input apimcp.ToolInput) (apimcp.APIResponse, error) {
	return r.runtime.ExecuteRead(ctx, profileID, operation, input)
}

func (r extensionRuntime) ExecuteBatch(ctx context.Context, request apimcp.ExtensionBatchRequest) ([]apimcp.ExtensionBatchResult, error) {
	return r.runtime.ExecuteBatch(ctx, request)
}
