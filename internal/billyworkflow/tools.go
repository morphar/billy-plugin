package billyworkflow

import (
	"context"
	"fmt"

	"github.com/cortexium-io/api-mcp/apimcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tools returns the fixed Billy workflow tools layered on top of generated OpenAPI operations.
func Tools() []apimcp.ExtensionTool {
	return []apimcp.ExtensionTool{
		readTool("diagnose_billy_connection", "Diagnose Billy connection", "Verify the selected Billy profile and return the authenticated organization without changing data.", handleDiagnoseConnection),
		readTool("review_billy_bank_account", "Review Billy bank account", "Review every Billy bank line for one cash account in an inclusive date range. Fetches all pages, distinguishes match approval from subject association, and reports duplicate candidates and completeness.", handleReviewBankAccount),
		readTool("review_billy_document_coverage", "Review Billy document coverage", "Review persisted Billy bills and attachments in an inclusive date range and state the documented API limitation for pending inbox documents.", handleReviewDocumentCoverage),
		readTool("review_billy_vat_period", "Review Billy VAT period", "Review one Billy sales-tax return with its exact return/report fields, payment records, and separate prepared, settled, and paid states.", handleReviewVATPeriod),
		readTool("review_billy_foreign_currency_purchase", "Review Billy foreign-currency purchase", "Review one Billy bill against its organization base currency, bill currency, exchange rate, lines, and payment-account currency.", handleReviewForeignCurrencyPurchase),
		readTool("preview_billy_bank_line_cleanup", "Preview Billy bank-line cleanup", "Preview deletion of exact duplicate bank-line IDs against exact retained IDs. Rejects approved or associated targets and returns a deterministic SHA-256 plan digest; it performs no writes.", handlePreviewBankLineCleanup),
		{
			Tool: &mcp.Tool{
				Name:        "commit_billy_bank_line_cleanup",
				Title:       "Commit Billy bank-line cleanup",
				Description: "Commit an unchanged cleanup plan returned by preview_billy_bank_line_cleanup. Re-reads every exact target and association before and immediately after one native batch approval, treats already-absent DELETE targets idempotently, and verifies aggregate absence.",
				InputSchema: cleanupCommitSchema(),
				Annotations: &mcp.ToolAnnotations{
					Title:           "Commit Billy bank-line cleanup",
					DestructiveHint: boolPointer(true),
					IdempotentHint:  true,
					ReadOnlyHint:    false,
				},
			},
			Handler: func(ctx context.Context, rt *apimcp.ExtensionRuntime, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return handleCommitBankLineCleanup(ctx, extensionRuntime{runtime: rt}, request)
			},
		},
	}
}

func readTool(name, title, description string, handler func(context.Context, runtime, *mcp.CallToolRequest) (*mcp.CallToolResult, error)) apimcp.ExtensionTool {
	return apimcp.ExtensionTool{
		Tool: &mcp.Tool{
			Name:        name,
			Title:       title,
			Description: description,
			InputSchema: inputSchemaFor(name),
			Annotations: &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true},
		},
		Handler: func(ctx context.Context, rt *apimcp.ExtensionRuntime, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handler(ctx, extensionRuntime{runtime: rt}, request)
		},
	}
}

func inputSchemaFor(name string) map[string]any {
	switch name {
	case "diagnose_billy_connection":
		return strictObject(map[string]any{"profile": stringSchema("Billy profile ID. Omit only when profile resolution is unambiguous.")})
	case "review_billy_bank_account":
		return strictObject(map[string]any{
			"profile":        stringSchema("Billy profile ID. Omit only when profile resolution is unambiguous."),
			"accountId":      requiredStringSchema("Exact Billy cash-account ID."),
			"startDate":      dateSchema("Inclusive first bank-line entry date."),
			"endDate":        dateSchema("Inclusive last bank-line entry date."),
			"maxLineDetails": integerSchema("Maximum line details returned; all fetched lines are still included in summaries.", 1, 1000, 200),
		}, "accountId", "startDate", "endDate")
	case "review_billy_document_coverage":
		return strictObject(map[string]any{
			"profile":           stringSchema("Billy profile ID. Omit only when profile resolution is unambiguous."),
			"startDate":         dateSchema("Inclusive first bill entry date."),
			"endDate":           dateSchema("Inclusive last bill entry date."),
			"maxMissingDetails": integerSchema("Maximum missing-document details returned; all bills are still counted.", 1, 1000, 200),
		}, "startDate", "endDate")
	case "review_billy_vat_period":
		return strictObject(map[string]any{
			"profile":          stringSchema("Billy profile ID. Omit only when profile resolution is unambiguous."),
			"salesTaxReturnId": requiredStringSchema("Exact Billy sales-tax-return ID."),
		}, "salesTaxReturnId")
	case "review_billy_foreign_currency_purchase":
		return strictObject(map[string]any{
			"profile": stringSchema("Billy profile ID. Omit only when profile resolution is unambiguous."),
			"billId":  requiredStringSchema("Exact Billy bill ID."),
		}, "billId")
	case "preview_billy_bank_line_cleanup":
		return cleanupPreviewSchema()
	default:
		panic(fmt.Sprintf("missing input schema for %s", name))
	}
}

func strictObject(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringSchema(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func requiredStringSchema(description string) map[string]any {
	return map[string]any{"type": "string", "minLength": 1, "description": description}
}

func dateSchema(description string) map[string]any {
	return map[string]any{"type": "string", "format": "date", "description": description}
}

func integerSchema(description string, minimum, maximum, defaultValue int) map[string]any {
	return map[string]any{
		"type":        "integer",
		"minimum":     minimum,
		"maximum":     maximum,
		"default":     defaultValue,
		"description": description,
	}
}

func boolPointer(value bool) *bool {
	return &value
}

func resultResponse(message string, value any) *mcp.CallToolResult {
	result, err := responseMap(value)
	if err != nil {
		return apimcp.ErrorResponse(err.Error())
	}
	return apimcp.JSONResponse(message, result)
}

func invalidInputResponse(err error) *mcp.CallToolResult {
	return apimcp.ErrorResponse("invalid workflow input: " + err.Error())
}

func workflowErrorResponse(err error) *mcp.CallToolResult {
	return apimcp.ErrorResponse("Billy workflow failed: " + err.Error())
}
