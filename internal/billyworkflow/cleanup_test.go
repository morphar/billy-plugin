package billyworkflow

import (
	"context"
	"strings"
	"testing"

	"github.com/cortexium-io/api-mcp/apimcp"
)

type cleanupFake struct {
	runtime       fakeRuntime
	lines         map[string]map[string]any
	matches       map[string]map[string]any
	associations  []map[string]any
	batchError    error
	keepDeleted   bool
	afterApproval func()
	deleteCalls   int
}

func newCleanupFake() *cleanupFake {
	f := &cleanupFake{
		lines: map[string]map[string]any{
			"delete": cleanupLine("delete", "", "import-a"),
			"retain": cleanupLine("retain", "", "import-b"),
		},
		matches: map[string]map[string]any{},
	}
	f.runtime.profile = apimcp.APIProfile{ID: "techbase", Name: "Techbase"}
	f.runtime.read = f.read
	f.runtime.batch = f.batch
	return f
}

func cleanupLine(id, matchID, groupID string) map[string]any {
	return map[string]any{
		"id":          id,
		"accountId":   "mastercard",
		"matchId":     matchID,
		"entryDate":   "2026-06-08",
		"description": "GITHUB, INC.",
		"amount":      90.49,
		"side":        "credit",
		"status":      "booked",
		"provider":    "mastercard-csv",
		"externalId":  "github-2026-06-08",
		"groupId":     groupID,
		"balance":     1000.0,
	}
}

func (f *cleanupFake) read(_ context.Context, _ string, operation string, input apimcp.ToolInput) (apimcp.APIResponse, error) {
	switch operation {
	case "list_bank_line_subject_associations":
		return pagedResponse("bankLineSubjectAssociations", f.associations, 1, 1, len(f.associations)), nil
	case "get_bank_line":
		id := input.Path["id"].(string)
		line, ok := f.lines[id]
		if !ok {
			return apimcp.APIResponse{Status: 404, StatusText: "404 Not Found"}, nil
		}
		return objectResponse("bankLine", line), nil
	case "get_bank_line_match":
		id := input.Path["id"].(string)
		match, ok := f.matches[id]
		if !ok {
			return apimcp.APIResponse{Status: 404, StatusText: "404 Not Found"}, nil
		}
		return objectResponse("bankLineMatch", match), nil
	default:
		return apimcp.APIResponse{}, nil
	}
}

func (f *cleanupFake) batch(_ context.Context, request apimcp.ExtensionBatchRequest) ([]apimcp.ExtensionBatchResult, error) {
	if f.batchError != nil {
		return nil, f.batchError
	}
	if f.afterApproval != nil {
		f.afterApproval()
	}
	if request.AfterApprovalCheck != nil {
		if err := request.AfterApprovalCheck(context.Background()); err != nil {
			return nil, err
		}
	}
	results := make([]apimcp.ExtensionBatchResult, 0, len(request.Items))
	for _, item := range request.Items {
		id := item.Input.Path["id"].(string)
		f.deleteCalls++
		if !f.keepDeleted {
			delete(f.lines, id)
		}
		response := apimcp.APIResponse{OK: true, Status: 204, StatusText: "204 No Content"}
		results = append(results, apimcp.ExtensionBatchResult{ID: item.ID, Operation: item.Operation, Response: &response})
	}
	return results, nil
}

func cleanupPreview(t *testing.T, f *cleanupFake) (cleanupPlan, string) {
	t.Helper()
	result, err := previewBankLineCleanup(context.Background(), &f.runtime, cleanupPreviewInput{
		Profile:   "techbase",
		AccountID: "mastercard",
		Targets:   []cleanupPairInput{{DeleteBankLineID: "delete", RetainBankLineID: "retain"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result["safeToCommit"] != true {
		t.Fatalf("preview = %#v", result)
	}
	summary := result["summary"].(map[string]any)
	if summary["targetCount"] != 1 || summary["deleteCreditAmount"] != 90.49 || summary["deleteSideSignedAmount"] != -90.49 {
		t.Fatalf("preview summary = %#v", summary)
	}
	return result["plan"].(cleanupPlan), result["digest"].(string)
}

func TestCleanupPreviewRejectsApprovedMatch(t *testing.T) {
	f := newCleanupFake()
	f.lines["delete"]["matchId"] = "match"
	f.matches["match"] = map[string]any{"id": "match", "isApproved": true}

	_, err := previewBankLineCleanup(context.Background(), &f.runtime, cleanupPreviewInput{
		Profile: "techbase", AccountID: "mastercard",
		Targets: []cleanupPairInput{{DeleteBankLineID: "delete", RetainBankLineID: "retain"}},
	})
	if err == nil || !strings.Contains(err.Error(), "approved") {
		t.Fatalf("error = %v", err)
	}
	if len(f.runtime.batchCalls) != 0 {
		t.Fatalf("batch calls = %d", len(f.runtime.batchCalls))
	}
}

func TestCleanupPreviewRejectsSubjectAssociation(t *testing.T) {
	f := newCleanupFake()
	f.lines["delete"]["matchId"] = "match"
	f.matches["match"] = map[string]any{"id": "match", "isApproved": false}
	f.associations = []map[string]any{{"id": "association", "matchId": "match", "subjectReference": "transaction:1"}}

	_, err := previewBankLineCleanup(context.Background(), &f.runtime, cleanupPreviewInput{
		Profile: "techbase", AccountID: "mastercard",
		Targets: []cleanupPairInput{{DeleteBankLineID: "delete", RetainBankLineID: "retain"}},
	})
	if err == nil || !strings.Contains(err.Error(), "subject association") {
		t.Fatalf("error = %v", err)
	}
}

func TestCleanupCommitRejectsChangedPlanBeforeBatch(t *testing.T) {
	f := newCleanupFake()
	plan, digest := cleanupPreview(t, f)
	plan.Targets[0].Delete.Description = "changed after preview"

	_, err := commitBankLineCleanup(context.Background(), &f.runtime, cleanupCommitInput{
		Profile: "techbase", Digest: digest, Plan: plan,
	})
	if err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("error = %v", err)
	}
	if len(f.runtime.batchCalls) != 0 {
		t.Fatalf("batch calls = %d", len(f.runtime.batchCalls))
	}
}

func TestCleanupCommitUsesOneBatchAndVerifiesAggregate(t *testing.T) {
	f := newCleanupFake()
	f.lines["delete-two"] = cleanupLine("delete-two", "", "import-c")
	f.lines["retain-two"] = cleanupLine("retain-two", "", "import-d")
	preview, err := previewBankLineCleanup(context.Background(), &f.runtime, cleanupPreviewInput{
		Profile: "techbase", AccountID: "mastercard",
		Targets: []cleanupPairInput{
			{DeleteBankLineID: "delete", RetainBankLineID: "retain"},
			{DeleteBankLineID: "delete-two", RetainBankLineID: "retain-two"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := commitBankLineCleanup(context.Background(), &f.runtime, cleanupCommitInput{
		Profile: "techbase", Digest: preview["digest"].(string), Plan: preview["plan"].(cleanupPlan),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.runtime.batchCalls) != 1 || len(f.runtime.batchCalls[0].Items) != 2 {
		t.Fatalf("batch calls = %#v", f.runtime.batchCalls)
	}
	verification := result["verification"].(map[string]any)
	if verification["success"] != true || verification["allDeleteTargetsAbsent"] != true || verification["allRetainedTargetsUnchanged"] != true {
		t.Fatalf("verification = %#v", verification)
	}
}

func TestCleanupCommitRetryIsIdempotentWhenTargetAlreadyAbsent(t *testing.T) {
	f := newCleanupFake()
	plan, digest := cleanupPreview(t, f)
	delete(f.lines, "delete")

	result, err := commitBankLineCleanup(context.Background(), &f.runtime, cleanupCommitInput{
		Profile: "techbase", Digest: digest, Plan: plan,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.runtime.batchCalls) != 0 {
		t.Fatalf("batch calls = %d, want 0", len(f.runtime.batchCalls))
	}
	items := result["items"].([]cleanupItemResult)
	if items[0].Outcome != "alreadyAbsent" || !items[0].VerifiedAbsent || !items[0].RetainVerified {
		t.Fatalf("item = %+v", items[0])
	}
}

func TestCleanupCommitReportsVerificationFailure(t *testing.T) {
	f := newCleanupFake()
	f.keepDeleted = true
	plan, digest := cleanupPreview(t, f)

	result, err := commitBankLineCleanup(context.Background(), &f.runtime, cleanupCommitInput{
		Profile: "techbase", Digest: digest, Plan: plan,
	})
	if err != nil {
		t.Fatal(err)
	}
	verification := result["verification"].(map[string]any)
	if verification["success"] != false || verification["allDeleteTargetsAbsent"] != false {
		t.Fatalf("verification = %#v", verification)
	}
	items := result["items"].([]cleanupItemResult)
	if items[0].Outcome != "verificationFailed" {
		t.Fatalf("item = %+v", items[0])
	}
	if response := cleanupCommitResultResponse(result); !response.IsError {
		t.Fatalf("unverified commit response must be an MCP error: %+v", response)
	}
}

func TestCleanupCommitRejectsMatchChangeAfterApprovalBeforeDelete(t *testing.T) {
	f := newCleanupFake()
	f.lines["delete"]["matchId"] = "match"
	f.matches["match"] = map[string]any{"id": "match", "isApproved": false}
	plan, digest := cleanupPreview(t, f)
	f.afterApproval = func() {
		f.matches["match"]["isApproved"] = true
		f.matches["match"]["approvedTime"] = "2026-09-01T12:00:00Z"
	}

	_, err := commitBankLineCleanup(context.Background(), &f.runtime, cleanupCommitInput{
		Profile: "techbase", Digest: digest, Plan: plan,
	})
	if err == nil || !strings.Contains(err.Error(), "changed after approval") {
		t.Fatalf("error = %v", err)
	}
	if f.deleteCalls != 0 || f.lines["delete"] == nil {
		t.Fatalf("unsafe delete crossed post-approval guard: deleteCalls=%d line=%#v", f.deleteCalls, f.lines["delete"])
	}
}
