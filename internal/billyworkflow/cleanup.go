package billyworkflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/cortexium-io/api-mcp/apimcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	cleanupPlanVersion = 1
	maxCleanupTargets  = 100
)

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type cleanupPairInput struct {
	DeleteBankLineID string `json:"deleteBankLineId"`
	RetainBankLineID string `json:"retainBankLineId"`
}

type cleanupPreviewInput struct {
	Profile   string             `json:"profile,omitempty"`
	AccountID string             `json:"accountId"`
	Targets   []cleanupPairInput `json:"targets"`
}

type bankLineSnapshot struct {
	ID          string   `json:"id"`
	AccountID   string   `json:"accountId"`
	MatchID     string   `json:"matchId"`
	EntryDate   string   `json:"entryDate"`
	Description string   `json:"description"`
	Amount      float64  `json:"amount"`
	Side        string   `json:"side"`
	Status      string   `json:"status"`
	Provider    string   `json:"provider"`
	ExternalID  string   `json:"externalId"`
	GroupID     string   `json:"groupId"`
	Balance     *float64 `json:"balance"`
}

type bankLineMatchSnapshot struct {
	ID             string   `json:"id"`
	AccountID      string   `json:"accountId"`
	DifferenceType string   `json:"differenceType"`
	FeeAccountID   string   `json:"feeAccountId"`
	EntryDate      string   `json:"entryDate"`
	Amount         *float64 `json:"amount"`
	Side           string   `json:"side"`
	IsApproved     bool     `json:"isApproved"`
	ApprovedTime   string   `json:"approvedTime"`
}

type cleanupTarget struct {
	Delete      bankLineSnapshot       `json:"delete"`
	Retain      bankLineSnapshot       `json:"retain"`
	DeleteMatch *bankLineMatchSnapshot `json:"deleteMatch"`
}

type cleanupPlan struct {
	Version   int             `json:"version"`
	ProfileID string          `json:"profileId"`
	AccountID string          `json:"accountId"`
	Targets   []cleanupTarget `json:"targets"`
}

type cleanupCommitInput struct {
	Profile string      `json:"profile,omitempty"`
	Digest  string      `json:"digest"`
	Plan    cleanupPlan `json:"plan"`
}

type cleanupItemResult struct {
	DeleteBankLineID string `json:"deleteBankLineId"`
	RetainBankLineID string `json:"retainBankLineId"`
	Preflight        string `json:"preflight"`
	BatchOperation   string `json:"batchOperation,omitempty"`
	HTTPStatus       int    `json:"httpStatus,omitempty"`
	Outcome          string `json:"outcome"`
	Error            string `json:"error,omitempty"`
	VerifiedAbsent   bool   `json:"verifiedAbsent"`
	RetainVerified   bool   `json:"retainVerified"`
}

func cleanupPreviewSchema() map[string]any {
	pair := strictObject(map[string]any{
		"deleteBankLineId": requiredStringSchema("Exact bank-line ID proposed for deletion."),
		"retainBankLineId": requiredStringSchema("Exact duplicate bank-line ID that must remain."),
	}, "deleteBankLineId", "retainBankLineId")
	return strictObject(map[string]any{
		"profile":   stringSchema("Billy profile ID. Omit only when profile resolution is unambiguous."),
		"accountId": requiredStringSchema("Exact Billy cash-account ID containing every target pair."),
		"targets": map[string]any{
			"type":        "array",
			"description": "Exact delete/retain pairs. Broad filters and inferred deletion sets are not accepted.",
			"minItems":    1,
			"maxItems":    maxCleanupTargets,
			"items":       pair,
		},
	}, "accountId", "targets")
}

func cleanupCommitSchema() map[string]any {
	line := strictObject(map[string]any{
		"id":          requiredStringSchema("Exact bank-line ID."),
		"accountId":   requiredStringSchema("Exact accountId observed during preview."),
		"matchId":     stringSchema("Exact matchId observed during preview, or an empty string when absent."),
		"entryDate":   dateSchema("Exact entryDate observed during preview."),
		"description": map[string]any{"type": "string"},
		"amount":      map[string]any{"type": "number"},
		"side":        requiredStringSchema("Exact side observed during preview."),
		"status":      map[string]any{"type": "string"},
		"provider":    map[string]any{"type": "string"},
		"externalId":  map[string]any{"type": "string"},
		"groupId":     map[string]any{"type": "string"},
		"balance": map[string]any{
			"type": []string{"number", "null"},
		},
	}, "id", "accountId", "matchId", "entryDate", "description", "amount", "side", "status", "provider", "externalId", "groupId", "balance")
	match := strictObject(map[string]any{
		"id":             requiredStringSchema("Exact bank-line-match ID."),
		"accountId":      map[string]any{"type": "string"},
		"differenceType": map[string]any{"type": "string"},
		"feeAccountId":   map[string]any{"type": "string"},
		"entryDate":      map[string]any{"type": "string"},
		"amount":         map[string]any{"type": []string{"number", "null"}},
		"side":           map[string]any{"type": "string"},
		"isApproved":     map[string]any{"type": "boolean"},
		"approvedTime":   map[string]any{"type": "string"},
	}, "id", "accountId", "differenceType", "feeAccountId", "entryDate", "amount", "side", "isApproved", "approvedTime")
	target := strictObject(map[string]any{
		"delete": line,
		"retain": line,
		"deleteMatch": map[string]any{
			"anyOf": []any{match, map[string]any{"type": "null"}},
		},
	}, "delete", "retain", "deleteMatch")
	plan := strictObject(map[string]any{
		"version":   map[string]any{"type": "integer", "const": cleanupPlanVersion},
		"profileId": requiredStringSchema("Resolved Billy profile ID from preview."),
		"accountId": requiredStringSchema("Exact Billy cash-account ID from preview."),
		"targets": map[string]any{
			"type":     "array",
			"minItems": 1,
			"maxItems": maxCleanupTargets,
			"items":    target,
		},
	}, "version", "profileId", "accountId", "targets")
	return strictObject(map[string]any{
		"profile": stringSchema("Billy profile selector used for this commit."),
		"digest": map[string]any{
			"type":        "string",
			"pattern":     "^[0-9a-f]{64}$",
			"description": "Deterministic SHA-256 digest returned by preview for this exact plan.",
		},
		"plan": plan,
	}, "digest", "plan")
}

func handlePreviewBankLineCleanup(ctx context.Context, rt runtime, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input cleanupPreviewInput
	if err := apimcp.DecodeInput(request, &input); err != nil {
		return invalidInputResponse(err), nil
	}
	result, err := previewBankLineCleanup(ctx, rt, input)
	if err != nil {
		return workflowErrorResponse(err), nil
	}
	return resultResponse("Billy bank-line cleanup previewed; no data was changed.", result), nil
}

func previewBankLineCleanup(ctx context.Context, rt runtime, input cleanupPreviewInput) (map[string]any, error) {
	if err := validateCleanupPairInput(input.AccountID, input.Targets); err != nil {
		return nil, err
	}
	profile, err := rt.ResolveProfile(input.Profile)
	if err != nil {
		return nil, err
	}
	associations, associationPaging, err := fetchAll(ctx, rt, input.Profile, "list_bank_line_subject_associations", "bankLineSubjectAssociations", nil)
	if err != nil {
		return nil, err
	}
	if !associationPaging.Complete {
		return nil, errors.New("bank-line subject associations were not fetched completely; cleanup preview is unsafe")
	}
	associationsByMatch := associationsByMatchID(associations)

	plan := cleanupPlan{
		Version:   cleanupPlanVersion,
		ProfileID: profile.ID,
		AccountID: input.AccountID,
		Targets:   make([]cleanupTarget, 0, len(input.Targets)),
	}
	evidence := make([]map[string]any, 0, len(input.Targets))
	for _, pair := range input.Targets {
		deleteLine, found, err := readBankLineSnapshot(ctx, rt, input.Profile, pair.DeleteBankLineID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("delete target %q does not exist", pair.DeleteBankLineID)
		}
		retainLine, found, err := readBankLineSnapshot(ctx, rt, input.Profile, pair.RetainBankLineID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("retain target %q does not exist", pair.RetainBankLineID)
		}
		reason, err := validateDuplicatePair(input.AccountID, deleteLine, retainLine)
		if err != nil {
			return nil, err
		}

		var matchSnapshot *bankLineMatchSnapshot
		if deleteLine.MatchID != "" {
			match, found, err := readBankLineMatchSnapshot(ctx, rt, input.Profile, deleteLine.MatchID)
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, fmt.Errorf("delete target %q references missing bank-line match %q", deleteLine.ID, deleteLine.MatchID)
			}
			if match.IsApproved {
				return nil, fmt.Errorf("delete target %q is unsafe: match %q is approved", deleteLine.ID, match.ID)
			}
			if count := len(associationsByMatch[match.ID]); count > 0 {
				return nil, fmt.Errorf("delete target %q is unsafe: match %q has %d subject association(s)", deleteLine.ID, match.ID, count)
			}
			matchSnapshot = &match
		}
		plan.Targets = append(plan.Targets, cleanupTarget{Delete: deleteLine, Retain: retainLine, DeleteMatch: matchSnapshot})
		evidence = append(evidence, map[string]any{
			"deleteBankLineId":                   deleteLine.ID,
			"retainBankLineId":                   retainLine.ID,
			"duplicateEvidence":                  reason,
			"deleteMatchApproved":                false,
			"deleteMatchSubjectAssociationCount": len(associationsByMatch[deleteLine.MatchID]),
		})
	}
	canonicalizeCleanupPlan(&plan)
	sort.Slice(evidence, func(i, j int) bool {
		return stringValue(evidence[i]["deleteBankLineId"]) < stringValue(evidence[j]["deleteBankLineId"])
	})
	digest, err := cleanupPlanDigest(plan)
	if err != nil {
		return nil, err
	}
	deleteDebit, deleteCredit := cleanupSideTotals(plan, true)
	retainDebit, retainCredit := cleanupSideTotals(plan, false)

	return map[string]any{
		"profile":         profile,
		"safeToCommit":    true,
		"digestAlgorithm": "SHA-256",
		"digest":          digest,
		"plan":            plan,
		"evidence":        evidence,
		"summary": map[string]any{
			"targetCount":             len(plan.Targets),
			"deleteDebitAmount":       deleteDebit,
			"deleteCreditAmount":      deleteCredit,
			"deleteSideSignedAmount":  deleteDebit - deleteCredit,
			"retainDebitAmount":       retainDebit,
			"retainCreditAmount":      retainCredit,
			"retainSideSignedAmount":  retainDebit - retainCredit,
			"sideSignedAmountMeaning": "Debit is positive and credit is negative. This is transaction-side evidence, not an inferred account-type balance.",
		},
		"guardSemantics": map[string]any{
			"persistedPreviewToken": false,
			"optimisticConcurrency": "Commit recomputes the caller-supplied plan digest, re-reads every delete and retain line plus every delete match, and rejects any field change or new subject association before requesting approval.",
			"writeScope":            "Only the exact plan.targets[].delete.id bank lines are submitted to delete_bank_line. Retained lines and match records are never written by this workflow.",
		},
		"completeness": map[string]any{
			"complete": true,
			"sources":  map[string]any{"bankLineSubjectAssociations": associationPaging},
		},
	}, nil
}

func handleCommitBankLineCleanup(ctx context.Context, rt runtime, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input cleanupCommitInput
	if err := apimcp.DecodeInput(request, &input); err != nil {
		return invalidInputResponse(err), nil
	}
	result, err := commitBankLineCleanup(ctx, rt, input)
	if err != nil {
		return workflowErrorResponse(err), nil
	}
	return cleanupCommitResultResponse(result), nil
}

func cleanupCommitResultResponse(result map[string]any) *mcp.CallToolResult {
	verification, _ := result["verification"].(map[string]any)
	message := "Billy bank-line cleanup commit completed and was verified."
	if verification["success"] != true {
		message = "Billy bank-line cleanup did not reach a fully verified final state. Inspect the structured per-item results and retry only after reviewing the failure."
	}
	response := resultResponse(message, result)
	if verification["success"] != true {
		response.IsError = true
	}
	return response
}

func commitBankLineCleanup(ctx context.Context, rt runtime, input cleanupCommitInput) (map[string]any, error) {
	if !sha256Pattern.MatchString(input.Digest) {
		return nil, errors.New("digest must be exactly 64 lowercase hexadecimal SHA-256 characters")
	}
	if err := validateCleanupPlan(input.Plan); err != nil {
		return nil, err
	}
	canonicalizeCleanupPlan(&input.Plan)
	computedDigest, err := cleanupPlanDigest(input.Plan)
	if err != nil {
		return nil, err
	}
	if computedDigest != input.Digest {
		return nil, fmt.Errorf("cleanup plan digest mismatch: supplied %s, computed %s", input.Digest, computedDigest)
	}
	profile, err := rt.ResolveProfile(input.Profile)
	if err != nil {
		return nil, err
	}
	if profile.ID != input.Plan.ProfileID {
		return nil, fmt.Errorf("cleanup plan profile %q does not match resolved profile %q", input.Plan.ProfileID, profile.ID)
	}

	associations, associationPaging, err := fetchAll(ctx, rt, input.Profile, "list_bank_line_subject_associations", "bankLineSubjectAssociations", nil)
	if err != nil {
		return nil, err
	}
	if !associationPaging.Complete {
		return nil, errors.New("bank-line subject associations were not fetched completely; cleanup commit is unsafe")
	}
	associationsByMatch := associationsByMatchID(associations)

	items := make([]cleanupItemResult, len(input.Plan.Targets))
	batchItems := make([]apimcp.ExtensionBatchItem, 0, len(input.Plan.Targets))
	batchTargetIndexes := map[string]int{}
	for index, target := range input.Plan.Targets {
		items[index] = cleanupItemResult{
			DeleteBankLineID: target.Delete.ID,
			RetainBankLineID: target.Retain.ID,
			Preflight:        "pending",
			Outcome:          "pending",
		}
		retainLine, found, err := readBankLineSnapshot(ctx, rt, input.Profile, target.Retain.ID)
		if err != nil {
			return nil, err
		}
		if !found || !reflect.DeepEqual(retainLine, target.Retain) {
			return nil, fmt.Errorf("retain bank line %q is missing or changed since preview", target.Retain.ID)
		}
		deleteLine, found, err := readBankLineSnapshot(ctx, rt, input.Profile, target.Delete.ID)
		if err != nil {
			return nil, err
		}
		if !found {
			items[index].Preflight = "alreadyAbsent"
			items[index].Outcome = "alreadyAbsent"
			continue
		}
		if !reflect.DeepEqual(deleteLine, target.Delete) {
			return nil, fmt.Errorf("delete bank line %q changed since preview", target.Delete.ID)
		}
		if _, err := validateDuplicatePair(input.Plan.AccountID, deleteLine, retainLine); err != nil {
			return nil, err
		}
		if target.DeleteMatch == nil {
			if deleteLine.MatchID != "" {
				return nil, fmt.Errorf("delete bank line %q gained match %q since preview", deleteLine.ID, deleteLine.MatchID)
			}
		} else {
			match, found, err := readBankLineMatchSnapshot(ctx, rt, input.Profile, target.DeleteMatch.ID)
			if err != nil {
				return nil, err
			}
			if !found || !reflect.DeepEqual(match, *target.DeleteMatch) {
				return nil, fmt.Errorf("bank-line match %q is missing or changed since preview", target.DeleteMatch.ID)
			}
			if match.IsApproved {
				return nil, fmt.Errorf("delete bank line %q is unsafe: match %q is approved", deleteLine.ID, match.ID)
			}
			if count := len(associationsByMatch[match.ID]); count > 0 {
				return nil, fmt.Errorf("delete bank line %q is unsafe: match %q now has %d subject association(s)", deleteLine.ID, match.ID, count)
			}
		}
		itemID := "delete:" + target.Delete.ID
		items[index].Preflight = "unchangedAndSafe"
		items[index].BatchOperation = "delete_bank_line"
		batchTargetIndexes[itemID] = index
		batchItems = append(batchItems, apimcp.ExtensionBatchItem{
			ID:        itemID,
			Operation: "delete_bank_line",
			Input:     apimcp.ToolInput{Path: map[string]any{"id": target.Delete.ID}},
		})
	}

	batchCalled := false
	if len(batchItems) > 0 {
		batchCalled = true
		batchResults, err := rt.ExecuteBatch(ctx, apimcp.ExtensionBatchRequest{
			Profile: input.Profile,
			Action:  "Delete reviewed duplicate Billy bank lines",
			Details: map[string]any{
				"digest":      input.Digest,
				"accountId":   input.Plan.AccountID,
				"targetCount": len(batchItems),
			},
			Items:       batchItems,
			StopOnError: false,
		})
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		for _, batchResult := range batchResults {
			index, ok := batchTargetIndexes[batchResult.ID]
			if !ok {
				return nil, fmt.Errorf("batch returned unknown result ID %q", batchResult.ID)
			}
			seen[batchResult.ID] = true
			item := &items[index]
			if batchResult.Response != nil {
				item.HTTPStatus = batchResult.Response.Status
			}
			switch {
			case batchResult.Response != nil && batchResult.Response.Status == 404:
				item.Outcome = "alreadyAbsent"
			case batchResult.Error != "":
				item.Outcome = "deleteError"
				item.Error = batchResult.Error
			case batchResult.Response != nil && batchResult.Response.OK:
				item.Outcome = "deleted"
			default:
				item.Outcome = "deleteError"
				item.Error = "batch returned no successful response"
			}
		}
		for itemID, index := range batchTargetIndexes {
			if !seen[itemID] {
				items[index].Outcome = "deleteError"
				items[index].Error = "batch returned no result for this target"
			}
		}
	}

	allAbsent := true
	allRetained := true
	verificationComplete := true
	for index, target := range input.Plan.Targets {
		_, found, err := readBankLineSnapshot(ctx, rt, input.Profile, target.Delete.ID)
		if err != nil {
			items[index].Error = appendError(items[index].Error, "delete verification: "+err.Error())
			allAbsent = false
			verificationComplete = false
		} else {
			items[index].VerifiedAbsent = !found
			if found {
				allAbsent = false
				items[index].Outcome = "verificationFailed"
				items[index].Error = appendError(items[index].Error, "delete target still exists after commit")
			}
		}
		retain, found, err := readBankLineSnapshot(ctx, rt, input.Profile, target.Retain.ID)
		if err != nil || !found || !reflect.DeepEqual(retain, target.Retain) {
			allRetained = false
			message := "retained duplicate is missing or changed during verification"
			if err != nil {
				verificationComplete = false
				message = "retain verification: " + err.Error()
			}
			items[index].Error = appendError(items[index].Error, message)
		} else {
			items[index].RetainVerified = true
		}
	}

	return map[string]any{
		"profile": profile,
		"digest":  input.Digest,
		"batch": map[string]any{
			"executeBatchCalled": batchCalled,
			"approvalCount":      map[bool]int{true: 1, false: 0}[batchCalled],
			"submittedItemCount": len(batchItems),
		},
		"items": items,
		"verification": map[string]any{
			"complete":                    verificationComplete,
			"allDeleteTargetsAbsent":      allAbsent,
			"allRetainedTargetsUnchanged": allRetained,
			"success":                     verificationComplete && allAbsent && allRetained,
		},
		"completeness": map[string]any{
			"complete": true,
			"sources":  map[string]any{"bankLineSubjectAssociations": associationPaging},
		},
	}, nil
}

func cleanupSideTotals(plan cleanupPlan, deleteSide bool) (float64, float64) {
	debit := 0.0
	credit := 0.0
	for _, target := range plan.Targets {
		line := target.Retain
		if deleteSide {
			line = target.Delete
		}
		switch strings.ToLower(line.Side) {
		case "debit":
			debit += line.Amount
		case "credit":
			credit += line.Amount
		}
	}
	return debit, credit
}

func validateCleanupPairInput(accountID string, targets []cleanupPairInput) error {
	if strings.TrimSpace(accountID) == "" {
		return errors.New("accountId is required")
	}
	if len(targets) == 0 || len(targets) > maxCleanupTargets {
		return fmt.Errorf("targets must contain between 1 and %d exact pairs", maxCleanupTargets)
	}
	deleteIDs := map[string]bool{}
	for index, target := range targets {
		deleteID := strings.TrimSpace(target.DeleteBankLineID)
		retainID := strings.TrimSpace(target.RetainBankLineID)
		if deleteID == "" || retainID == "" {
			return fmt.Errorf("target %d requires deleteBankLineId and retainBankLineId", index+1)
		}
		if deleteID == retainID {
			return fmt.Errorf("target %d cannot delete and retain the same bank line %q", index+1, deleteID)
		}
		if deleteIDs[deleteID] {
			return fmt.Errorf("duplicate delete target %q", deleteID)
		}
		deleteIDs[deleteID] = true
	}
	for _, target := range targets {
		if deleteIDs[target.RetainBankLineID] {
			return fmt.Errorf("bank line %q cannot be both a delete and retain target", target.RetainBankLineID)
		}
	}
	return nil
}

func validateCleanupPlan(plan cleanupPlan) error {
	if plan.Version != cleanupPlanVersion {
		return fmt.Errorf("cleanup plan version must be %d", cleanupPlanVersion)
	}
	if strings.TrimSpace(plan.ProfileID) == "" || strings.TrimSpace(plan.AccountID) == "" {
		return errors.New("cleanup plan requires profileId and accountId")
	}
	if len(plan.Targets) == 0 || len(plan.Targets) > maxCleanupTargets {
		return fmt.Errorf("cleanup plan targets must contain between 1 and %d items", maxCleanupTargets)
	}
	pairs := make([]cleanupPairInput, 0, len(plan.Targets))
	for _, target := range plan.Targets {
		if err := validateBankLineSnapshot(target.Delete); err != nil {
			return fmt.Errorf("delete target %q: %w", target.Delete.ID, err)
		}
		if err := validateBankLineSnapshot(target.Retain); err != nil {
			return fmt.Errorf("retain target %q: %w", target.Retain.ID, err)
		}
		if target.Delete.AccountID != plan.AccountID || target.Retain.AccountID != plan.AccountID {
			return fmt.Errorf("cleanup target pair %q/%q does not belong to account %q", target.Delete.ID, target.Retain.ID, plan.AccountID)
		}
		if target.DeleteMatch == nil && target.Delete.MatchID != "" {
			return fmt.Errorf("delete target %q requires its exact match snapshot", target.Delete.ID)
		}
		if target.DeleteMatch != nil {
			if target.DeleteMatch.ID == "" || target.DeleteMatch.ID != target.Delete.MatchID {
				return fmt.Errorf("delete target %q match snapshot does not match matchId %q", target.Delete.ID, target.Delete.MatchID)
			}
			if target.DeleteMatch.IsApproved {
				return fmt.Errorf("delete target %q plan contains an approved match", target.Delete.ID)
			}
		}
		pairs = append(pairs, cleanupPairInput{DeleteBankLineID: target.Delete.ID, RetainBankLineID: target.Retain.ID})
	}
	return validateCleanupPairInput(plan.AccountID, pairs)
}

func validateBankLineSnapshot(snapshot bankLineSnapshot) error {
	if snapshot.ID == "" || snapshot.AccountID == "" || snapshot.EntryDate == "" || snapshot.Side == "" {
		return errors.New("id, accountId, entryDate, and side are required exact fields")
	}
	return nil
}

func validateDuplicatePair(accountID string, deleteLine, retainLine bankLineSnapshot) (string, error) {
	if deleteLine.ID == retainLine.ID {
		return "", fmt.Errorf("bank line %q cannot be both deleted and retained", deleteLine.ID)
	}
	if deleteLine.AccountID != accountID || retainLine.AccountID != accountID {
		return "", fmt.Errorf("bank lines %q and %q must both belong to account %q", deleteLine.ID, retainLine.ID, accountID)
	}
	if deleteLine.EntryDate != retainLine.EntryDate ||
		deleteLine.Amount != retainLine.Amount ||
		deleteLine.Side != retainLine.Side ||
		normalizedDescription(deleteLine.Description) != normalizedDescription(retainLine.Description) {
		return "", fmt.Errorf("bank lines %q and %q do not have the same date, amount, side, and normalized description", deleteLine.ID, retainLine.ID)
	}
	if deleteLine.Provider == "" || deleteLine.Provider != retainLine.Provider {
		return "", fmt.Errorf("bank lines %q and %q lack the same non-empty import provider", deleteLine.ID, retainLine.ID)
	}
	if deleteLine.ExternalID != "" && deleteLine.ExternalID == retainLine.ExternalID {
		return "same provider and externalId plus identical transaction fields", nil
	}
	if deleteLine.GroupID != "" && retainLine.GroupID != "" && deleteLine.GroupID != retainLine.GroupID {
		return "same provider and transaction fields in distinct import groups", nil
	}
	return "", fmt.Errorf("bank lines %q and %q lack strong provider/externalId or distinct-import-group duplicate evidence", deleteLine.ID, retainLine.ID)
}

func readBankLineSnapshot(ctx context.Context, rt runtime, profileID, id string) (bankLineSnapshot, bool, error) {
	response, err := rt.ExecuteRead(ctx, profileID, "get_bank_line", apimcp.ToolInput{Path: map[string]any{"id": id}})
	if err != nil {
		return bankLineSnapshot{}, false, err
	}
	if response.Status == 404 {
		return bankLineSnapshot{}, false, nil
	}
	if response.Truncated {
		return bankLineSnapshot{}, false, fmt.Errorf("get_bank_line %q response was truncated", id)
	}
	if !response.OK {
		return bankLineSnapshot{}, false, apiResponseError("get_bank_line", response)
	}
	body, err := objectBody(response.Body)
	if err != nil {
		return bankLineSnapshot{}, false, err
	}
	line, ok := body["bankLine"].(map[string]any)
	if !ok {
		return bankLineSnapshot{}, false, errors.New("get_bank_line response is missing bankLine")
	}
	snapshot, err := bankLineSnapshotFromMap(line)
	return snapshot, err == nil, err
}

func bankLineSnapshotFromMap(line map[string]any) (bankLineSnapshot, error) {
	amount, ok := floatValue(line["amount"])
	if !ok {
		return bankLineSnapshot{}, errors.New("bank line amount is missing or not numeric")
	}
	var balance *float64
	if value, ok := floatValue(line["balance"]); ok {
		balance = &value
	}
	snapshot := bankLineSnapshot{
		ID:          stringValue(line["id"]),
		AccountID:   stringValue(line["accountId"]),
		MatchID:     stringValue(line["matchId"]),
		EntryDate:   stringValue(line["entryDate"]),
		Description: stringValue(line["description"]),
		Amount:      amount,
		Side:        stringValue(line["side"]),
		Status:      stringValue(line["status"]),
		Provider:    stringValue(line["provider"]),
		ExternalID:  stringValue(line["externalId"]),
		GroupID:     stringValue(line["groupId"]),
		Balance:     balance,
	}
	if err := validateBankLineSnapshot(snapshot); err != nil {
		return bankLineSnapshot{}, err
	}
	return snapshot, nil
}

func readBankLineMatchSnapshot(ctx context.Context, rt runtime, profileID, id string) (bankLineMatchSnapshot, bool, error) {
	response, err := rt.ExecuteRead(ctx, profileID, "get_bank_line_match", apimcp.ToolInput{Path: map[string]any{"id": id}})
	if err != nil {
		return bankLineMatchSnapshot{}, false, err
	}
	if response.Status == 404 {
		return bankLineMatchSnapshot{}, false, nil
	}
	if response.Truncated {
		return bankLineMatchSnapshot{}, false, fmt.Errorf("get_bank_line_match %q response was truncated", id)
	}
	if !response.OK {
		return bankLineMatchSnapshot{}, false, apiResponseError("get_bank_line_match", response)
	}
	body, err := objectBody(response.Body)
	if err != nil {
		return bankLineMatchSnapshot{}, false, err
	}
	match, ok := body["bankLineMatch"].(map[string]any)
	if !ok {
		return bankLineMatchSnapshot{}, false, errors.New("get_bank_line_match response is missing bankLineMatch")
	}
	var amount *float64
	if value, ok := floatValue(match["amount"]); ok {
		amount = &value
	}
	snapshot := bankLineMatchSnapshot{
		ID:             stringValue(match["id"]),
		AccountID:      stringValue(match["accountId"]),
		DifferenceType: stringValue(match["differenceType"]),
		FeeAccountID:   stringValue(match["feeAccountId"]),
		EntryDate:      stringValue(match["entryDate"]),
		Amount:         amount,
		Side:           stringValue(match["side"]),
		IsApproved:     boolValue(match["isApproved"]),
		ApprovedTime:   stringValue(match["approvedTime"]),
	}
	if snapshot.ID == "" {
		return bankLineMatchSnapshot{}, false, errors.New("bank-line match id is missing")
	}
	return snapshot, true, nil
}

func associationsByMatchID(associations []map[string]any) map[string][]map[string]any {
	result := map[string][]map[string]any{}
	for _, association := range associations {
		if matchID := stringValue(association["matchId"]); matchID != "" {
			result[matchID] = append(result[matchID], association)
		}
	}
	return result
}

func canonicalizeCleanupPlan(plan *cleanupPlan) {
	sort.Slice(plan.Targets, func(i, j int) bool {
		if plan.Targets[i].Delete.ID != plan.Targets[j].Delete.ID {
			return plan.Targets[i].Delete.ID < plan.Targets[j].Delete.ID
		}
		return plan.Targets[i].Retain.ID < plan.Targets[j].Retain.ID
	})
}

func cleanupPlanDigest(plan cleanupPlan) (string, error) {
	canonicalizeCleanupPlan(&plan)
	data, err := json.Marshal(plan)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func appendError(current, next string) string {
	if current == "" {
		return next
	}
	return current + "; " + next
}
