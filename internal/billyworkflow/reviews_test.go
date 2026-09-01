package billyworkflow

import (
	"context"
	"testing"

	"github.com/cortexium-io/api-mcp/apimcp"
)

func TestReviewBankAccountSeparatesBookedApprovalAndAssociation(t *testing.T) {
	rt := &fakeRuntime{}
	rt.read = func(_ context.Context, _ string, operation string, _ apimcp.ToolInput) (apimcp.APIResponse, error) {
		switch operation {
		case "list_bank_lines":
			return pagedResponse("bankLines", []map[string]any{
				{"id": "line-unapproved", "accountId": "bank", "matchId": "match-unapproved", "entryDate": "2026-06-01", "description": "Card", "amount": 100.0, "side": "credit", "status": "booked"},
				{"id": "line-reconciled", "accountId": "bank", "matchId": "match-reconciled", "entryDate": "2026-06-02", "description": "Invoice", "amount": 200.0, "side": "debit", "status": "booked"},
			}, 1, 1, 2), nil
		case "list_bank_line_matches":
			return pagedResponse("bankLineMatches", []map[string]any{
				{"id": "match-unapproved", "isApproved": false},
				{"id": "match-reconciled", "isApproved": true, "approvedTime": "2026-06-03T12:00:00Z"},
			}, 1, 1, 2), nil
		case "list_bank_line_subject_associations":
			return pagedResponse("bankLineSubjectAssociations", []map[string]any{
				{"id": "association", "matchId": "match-reconciled", "subjectReference": "transaction:123"},
			}, 1, 1, 1), nil
		default:
			t.Fatalf("unexpected operation %q", operation)
			return apimcp.APIResponse{}, nil
		}
	}

	result, err := reviewBankAccount(context.Background(), rt, bankReviewInput{
		Profile: "techbase", AccountID: "bank", StartDate: "2026-01-01", EndDate: "2026-06-30",
	})
	if err != nil {
		t.Fatal(err)
	}
	summary := result["summary"].(map[string]int)
	if summary["unapprovedWithoutAssociation"] != 1 || summary["approvedAndAssociated"] != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	lines := result["lines"].([]map[string]any)
	if lines[0]["reconciliationState"] != "unapprovedWithoutAssociation" {
		t.Fatalf("booked line state = %v", lines[0]["reconciliationState"])
	}
	completeness := result["completeness"].(map[string]any)
	if completeness["complete"] != true || completeness["dataThrough"] != "2026-06-02" || completeness["queryCoverageThrough"] != "2026-06-30" {
		t.Fatalf("completeness = %#v", completeness)
	}
	freshness := result["feedFreshness"].(map[string]any)
	if freshness["determined"] != false || freshness["latestImportedEntryDate"] != "2026-06-02" {
		t.Fatalf("feed freshness = %#v", freshness)
	}
}

func TestReviewDocumentCoverageDeclaresPendingInboxBlindSpot(t *testing.T) {
	rt := &fakeRuntime{}
	rt.read = func(_ context.Context, _ string, operation string, _ apimcp.ToolInput) (apimcp.APIResponse, error) {
		switch operation {
		case "list_bills":
			return pagedResponse("bills", []map[string]any{
				{"id": "with-doc", "entryDate": "2026-03-01", "contactName": "Hetzner"},
				{"id": "without-doc", "entryDate": "2026-04-01", "contactName": "GitHub"},
			}, 1, 1, 2), nil
		case "list_attachments":
			return pagedResponse("attachments", []map[string]any{
				{"id": "attachment", "ownerReference": "bill:with-doc"},
			}, 1, 1, 1), nil
		case "get_current_organization":
			return objectResponse("organization", map[string]any{"id": "org", "billEmailAddress": "inbox@example.test"}), nil
		default:
			t.Fatalf("unexpected operation %q", operation)
			return apimcp.APIResponse{}, nil
		}
	}

	result, err := reviewDocumentCoverage(context.Background(), rt, documentCoverageInput{
		Profile: "techbase", StartDate: "2026-01-01", EndDate: "2026-06-30",
	})
	if err != nil {
		t.Fatal(err)
	}
	summary := result["summary"].(map[string]any)
	if summary["withAttachment"] != 1 || summary["withoutAttachment"] != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	limitation := result["documentInboxLimitation"].(map[string]any)
	if limitation["pendingInboxReviewSupported"] != false {
		t.Fatalf("inbox limitation = %#v", limitation)
	}
	completeness := result["completeness"].(map[string]any)
	if completeness["completeForPersistedBills"] != true || completeness["completeForPendingInbox"] != false {
		t.Fatalf("completeness = %#v", completeness)
	}
}

func TestReviewVATPeriodDoesNotInferTaxAuthoritySubmission(t *testing.T) {
	rt := &fakeRuntime{}
	rt.read = func(_ context.Context, _ string, operation string, _ apimcp.ToolInput) (apimcp.APIResponse, error) {
		switch operation {
		case "get_sales_tax_return":
			return objectResponse("salesTaxReturn", map[string]any{
				"id": "vat", "startDate": "2026-01-01", "endDate": "2026-06-30", "isSettled": true, "isPaid": false, "totalAmount": 15411.0,
			}), nil
		case "list_sales_tax_payments":
			return pagedResponse("salesTaxPayments", []map[string]any{
				{"id": "payment", "salesTaxReturnId": "vat", "entryDate": "2026-08-31", "amount": 15411.0},
			}, 1, 1, 1), nil
		default:
			t.Fatalf("unexpected operation %q", operation)
			return apimcp.APIResponse{}, nil
		}
	}

	result, err := reviewVATPeriod(context.Background(), rt, vatPeriodInput{Profile: "techbase", SalesTaxReturnID: "vat"})
	if err != nil {
		t.Fatal(err)
	}
	states := result["states"].(map[string]any)
	if states["settled"].(map[string]any)["value"] != true || states["paid"].(map[string]any)["value"] != false {
		t.Fatalf("states = %#v", states)
	}
	submission := states["submissionToTaxAuthority"].(map[string]any)
	if submission["determined"] != false {
		t.Fatalf("submission = %#v", submission)
	}
	if result["filingReadiness"].(map[string]any)["determined"] != false {
		t.Fatalf("filing readiness = %#v", result["filingReadiness"])
	}
}

func TestReviewForeignCurrencyPurchaseUsesBillRateAndAccountCurrency(t *testing.T) {
	rt := &fakeRuntime{}
	rt.read = func(_ context.Context, _ string, operation string, input apimcp.ToolInput) (apimcp.APIResponse, error) {
		switch operation {
		case "get_bill":
			return objectResponse("bill", map[string]any{
				"id": "bill", "organizationId": "org", "entryDate": "2026-06-28", "currencyId": "USD", "exchangeRate": 0.0,
				"paymentAccountId": "mastercard", "amount": 4.0, "tax": 0.0, "isPaid": false,
			}), nil
		case "get_organization":
			return objectResponse("organization", map[string]any{"id": "org", "name": "Techbase", "baseCurrencyId": "DKK"}), nil
		case "get_currency":
			id := input.Path["id"].(string)
			return objectResponse("currency", map[string]any{"id": id, "name": id}), nil
		case "list_bill_lines":
			return pagedResponse("billLines", []map[string]any{
				{"id": "line", "billId": "bill", "amount": 4.0, "tax": 0.0, "taxRateId": "reverse", "priority": 1},
			}, 1, 1, 1), nil
		case "get_tax_rate":
			return objectResponse("taxRate", map[string]any{
				"id": "reverse", "name": "Services outside EU", "rate": 0.0, "appliesToPurchases": true,
			}), nil
		case "get_account":
			return objectResponse("account", map[string]any{"id": "mastercard", "name": "Mastercard", "currencyId": "DKK", "isBankAccount": true}), nil
		default:
			t.Fatalf("unexpected operation %q", operation)
			return apimcp.APIResponse{}, nil
		}
	}

	result, err := reviewForeignCurrencyPurchase(context.Background(), rt, foreignCurrencyPurchaseInput{Profile: "techbase", BillID: "bill"})
	if err != nil {
		t.Fatal(err)
	}
	classification := result["classification"].(map[string]any)
	if classification["isForeignCurrency"] != true || classification["billCurrencyId"] != "USD" || classification["baseCurrencyId"] != "DKK" {
		t.Fatalf("classification = %#v", classification)
	}
	checks := result["checks"].([]map[string]any)
	foundRateError := false
	for _, item := range checks {
		if item["name"] == "positiveBillExchangeRate" && item["passed"] == false && item["severity"] == "error" {
			foundRateError = true
		}
	}
	if !foundRateError {
		t.Fatalf("checks = %#v", checks)
	}
	if result["paymentAccount"].(map[string]any)["currencyId"] != "DKK" {
		t.Fatalf("payment account = %#v", result["paymentAccount"])
	}
	vatTreatment := result["vatTreatment"].(map[string]any)
	if vatTreatment["independentFromCurrency"] != true || len(vatTreatment["taxRates"].([]map[string]any)) != 1 {
		t.Fatalf("VAT treatment = %#v", vatTreatment)
	}
}
