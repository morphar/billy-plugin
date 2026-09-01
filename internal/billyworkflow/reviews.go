package billyworkflow

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/cortexium-io/api-mcp/apimcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type profileInput struct {
	Profile string `json:"profile,omitempty"`
}

type bankReviewInput struct {
	Profile        string `json:"profile,omitempty"`
	AccountID      string `json:"accountId"`
	StartDate      string `json:"startDate"`
	EndDate        string `json:"endDate"`
	MaxLineDetails int    `json:"maxLineDetails,omitempty"`
}

type documentCoverageInput struct {
	Profile           string `json:"profile,omitempty"`
	StartDate         string `json:"startDate"`
	EndDate           string `json:"endDate"`
	MaxMissingDetails int    `json:"maxMissingDetails,omitempty"`
}

type vatPeriodInput struct {
	Profile          string `json:"profile,omitempty"`
	SalesTaxReturnID string `json:"salesTaxReturnId"`
}

type foreignCurrencyPurchaseInput struct {
	Profile string `json:"profile,omitempty"`
	BillID  string `json:"billId"`
}

func handleDiagnoseConnection(ctx context.Context, rt runtime, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input profileInput
	if err := apimcp.DecodeInput(request, &input); err != nil {
		return invalidInputResponse(err), nil
	}
	profile, err := rt.ResolveProfile(input.Profile)
	if err != nil {
		return workflowErrorResponse(err), nil
	}
	organization, response, err := executeObjectRead(ctx, rt, input.Profile, "get_current_organization", "organization", apimcp.ToolInput{})
	if err != nil {
		return workflowErrorResponse(err), nil
	}

	result := map[string]any{
		"connected": true,
		"profile":   profile,
		"http": map[string]any{
			"status":     response.Status,
			"statusText": response.StatusText,
		},
		"organization": exactFields(organization,
			"id", "name", "registrationNo", "baseCurrencyId", "hasVat", "vatPeriod", "isLocked", "lockedCode", "lockedReason", "billEmailAddress"),
	}
	return resultResponse("Billy connection diagnosed.", result), nil
}

func handleReviewBankAccount(ctx context.Context, rt runtime, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input bankReviewInput
	if err := apimcp.DecodeInput(request, &input); err != nil {
		return invalidInputResponse(err), nil
	}
	result, err := reviewBankAccount(ctx, rt, input)
	if err != nil {
		return workflowErrorResponse(err), nil
	}
	return resultResponse("Billy bank account reviewed.", result), nil
}

func reviewBankAccount(ctx context.Context, rt runtime, input bankReviewInput) (map[string]any, error) {
	start, end, err := parseDateRange(input.StartDate, input.EndDate)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.AccountID) == "" {
		return nil, fmt.Errorf("accountId is required")
	}
	if input.MaxLineDetails == 0 {
		input.MaxLineDetails = 200
	}
	if input.MaxLineDetails < 1 || input.MaxLineDetails > 1000 {
		return nil, fmt.Errorf("maxLineDetails must be between 1 and 1000")
	}
	profile, err := rt.ResolveProfile(input.Profile)
	if err != nil {
		return nil, err
	}

	lines, linePaging, err := fetchAll(ctx, rt, input.Profile, "list_bank_lines", "bankLines", map[string]any{"accountId": input.AccountID})
	if err != nil {
		return nil, err
	}
	matches, matchPaging, err := fetchAll(ctx, rt, input.Profile, "list_bank_line_matches", "bankLineMatches", nil)
	if err != nil {
		return nil, err
	}
	associations, associationPaging, err := fetchAll(ctx, rt, input.Profile, "list_bank_line_subject_associations", "bankLineSubjectAssociations", nil)
	if err != nil {
		return nil, err
	}

	matchesByID := make(map[string]map[string]any, len(matches))
	for _, match := range matches {
		if id := stringValue(match["id"]); id != "" {
			matchesByID[id] = match
		}
	}
	associationsByMatch := make(map[string][]map[string]any)
	for _, association := range associations {
		matchID := stringValue(association["matchId"])
		if matchID != "" {
			associationsByMatch[matchID] = append(associationsByMatch[matchID], association)
		}
	}

	var filtered []map[string]any
	invalidEntryDates := 0
	for _, line := range lines {
		inRange, valid := dateInRange(stringValue(line["entryDate"]), start, end)
		if !valid {
			invalidEntryDates++
			continue
		}
		if inRange {
			filtered = append(filtered, line)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		left := stringValue(filtered[i]["entryDate"]) + "\x00" + stringValue(filtered[i]["id"])
		right := stringValue(filtered[j]["entryDate"]) + "\x00" + stringValue(filtered[j]["id"])
		return left < right
	})

	details := make([]map[string]any, 0, min(len(filtered), input.MaxLineDetails))
	counts := map[string]int{
		"linesInRange":                 len(filtered),
		"withoutMatchId":               0,
		"matchRecordMissing":           0,
		"matchApproved":                0,
		"hasSubjectAssociation":        0,
		"approvedAndAssociated":        0,
		"approvedWithoutAssociation":   0,
		"unapprovedWithAssociation":    0,
		"unapprovedWithoutAssociation": 0,
	}
	for _, line := range filtered {
		detail, state := bankLineDetail(line, matchesByID, associationsByMatch)
		counts[state]++
		if matchInfo, ok := detail["match"].(map[string]any); ok {
			if boolValue(matchInfo["approved"]) {
				counts["matchApproved"]++
			}
			if count, ok := integerValue(matchInfo["subjectAssociationCount"]); ok && count > 0 {
				counts["hasSubjectAssociation"]++
			}
		}
		if len(details) < input.MaxLineDetails {
			details = append(details, detail)
		}
	}

	latestRecordDate, _ := latestDate(lines, "entryDate")
	sourceComplete := linePaging.Complete && matchPaging.Complete && associationPaging.Complete
	dateFilterComplete := invalidEntryDates == 0
	queryCoverageThrough := ""
	if sourceComplete && dateFilterComplete {
		queryCoverageThrough = input.EndDate
	}

	return map[string]any{
		"profile": profile,
		"request": map[string]any{
			"accountId": input.AccountID,
			"startDate": input.StartDate,
			"endDate":   input.EndDate,
		},
		"summary":             counts,
		"lines":               details,
		"duplicateCandidates": duplicateCandidates(filtered, matchesByID, associationsByMatch),
		"semantics": map[string]any{
			"bankLineStatus":      "A bank-line status such as booked is reported as import/processing state and is not treated as reconciliation approval.",
			"matchApproval":       "Approval is taken only from bankLineMatch.isApproved.",
			"subjectAssociation":  "Bookkeeping linkage is taken only from bankLineSubjectAssociation records for the match ID.",
			"fullyReconciledRule": "The review labels a line approvedAndAssociated only when its match exists, isApproved is true, and at least one subject association exists.",
		},
		"feedFreshness": map[string]any{
			"determined":              false,
			"latestImportedEntryDate": latestRecordDate,
			"reason":                  "Complete API pagination proves collection coverage, not that Billy's bank feed includes every movement on the real bank statement. Compare this date with the external statement or feed timestamp.",
		},
		"completeness": map[string]any{
			"complete":              sourceComplete && dateFilterComplete,
			"dataThrough":           latestRecordDate,
			"queryCoverageThrough":  queryCoverageThrough,
			"latestRecordDate":      latestRecordDate,
			"invalidEntryDateCount": invalidEntryDates,
			"detailsReturned":       len(details),
			"detailsComplete":       len(details) == len(filtered),
			"sources": map[string]any{
				"bankLines":                   linePaging,
				"bankLineMatches":             matchPaging,
				"bankLineSubjectAssociations": associationPaging,
			},
		},
	}, nil
}

func bankLineDetail(line map[string]any, matchesByID map[string]map[string]any, associationsByMatch map[string][]map[string]any) (map[string]any, string) {
	matchID := stringValue(line["matchId"])
	match, matchFound := matchesByID[matchID]
	approved := matchFound && boolValue(match["isApproved"])
	associations := associationsByMatch[matchID]
	associationDetails := make([]map[string]any, 0, len(associations))
	for _, association := range associations {
		associationDetails = append(associationDetails, exactFields(association, "id", "matchId", "subjectReference"))
	}
	sort.Slice(associationDetails, func(i, j int) bool {
		return stringValue(associationDetails[i]["id"]) < stringValue(associationDetails[j]["id"])
	})

	state := "unapprovedWithoutAssociation"
	switch {
	case matchID == "":
		state = "withoutMatchId"
	case !matchFound:
		state = "matchRecordMissing"
	case approved && len(associations) > 0:
		state = "approvedAndAssociated"
	case approved:
		state = "approvedWithoutAssociation"
	case len(associations) > 0:
		state = "unapprovedWithAssociation"
	}

	return map[string]any{
		"bankLine": exactFields(line, "id", "accountId", "matchId", "entryDate", "description", "amount", "side", "status", "provider", "externalId", "groupId", "balance"),
		"match": map[string]any{
			"id":                      matchID,
			"recordFound":             matchFound,
			"approved":                approved,
			"approvedTime":            match["approvedTime"],
			"subjectAssociationCount": len(associations),
			"subjectAssociations":     associationDetails,
		},
		"reconciliationState": state,
	}, state
}

func duplicateCandidates(lines []map[string]any, matchesByID map[string]map[string]any, associationsByMatch map[string][]map[string]any) []map[string]any {
	type group struct {
		reason     string
		confidence string
		lines      []map[string]any
	}
	groups := map[string]*group{}
	for _, line := range lines {
		provider := stringValue(line["provider"])
		externalID := stringValue(line["externalId"])
		amount, amountOK := floatValue(line["amount"])
		if !amountOK {
			continue
		}
		key := ""
		reason := ""
		confidence := ""
		if provider != "" && externalID != "" {
			key = strings.Join([]string{"providerExternalId", stringValue(line["accountId"]), provider, externalID}, "\x00")
			reason = "sameProviderAndExternalId"
			confidence = "high"
		} else {
			key = strings.Join([]string{
				"transactionFingerprint",
				stringValue(line["accountId"]),
				stringValue(line["entryDate"]),
				canonicalAmount(amount),
				stringValue(line["side"]),
				normalizedDescription(stringValue(line["description"])),
			}, "\x00")
			reason = "sameDateAmountSideAndDescription"
			confidence = "medium"
		}
		if groups[key] == nil {
			groups[key] = &group{reason: reason, confidence: confidence}
		}
		groups[key].lines = append(groups[key].lines, line)
	}

	var result []map[string]any
	for _, candidate := range groups {
		if len(candidate.lines) < 2 {
			continue
		}
		lineEvidence := make([]map[string]any, 0, len(candidate.lines))
		for _, line := range candidate.lines {
			matchID := stringValue(line["matchId"])
			match, found := matchesByID[matchID]
			lineEvidence = append(lineEvidence, map[string]any{
				"id":                      stringValue(line["id"]),
				"groupId":                 stringValue(line["groupId"]),
				"matchId":                 matchID,
				"matchFound":              found,
				"matchApproved":           found && boolValue(match["isApproved"]),
				"subjectAssociationCount": len(associationsByMatch[matchID]),
			})
		}
		sort.Slice(lineEvidence, func(i, j int) bool {
			return stringValue(lineEvidence[i]["id"]) < stringValue(lineEvidence[j]["id"])
		})
		result = append(result, map[string]any{
			"reason":                    candidate.reason,
			"confidence":                candidate.confidence,
			"lines":                     lineEvidence,
			"automaticDeletionApproved": false,
			"nextStep":                  "Use preview_billy_bank_line_cleanup with exact delete and retain IDs; preview applies stricter duplicate and safety guards.",
		})
	}
	sort.Slice(result, func(i, j int) bool {
		left := result[i]["lines"].([]map[string]any)
		right := result[j]["lines"].([]map[string]any)
		return stringValue(left[0]["id"]) < stringValue(right[0]["id"])
	})
	return result
}

func handleReviewDocumentCoverage(ctx context.Context, rt runtime, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input documentCoverageInput
	if err := apimcp.DecodeInput(request, &input); err != nil {
		return invalidInputResponse(err), nil
	}
	result, err := reviewDocumentCoverage(ctx, rt, input)
	if err != nil {
		return workflowErrorResponse(err), nil
	}
	return resultResponse("Billy document coverage reviewed.", result), nil
}

func reviewDocumentCoverage(ctx context.Context, rt runtime, input documentCoverageInput) (map[string]any, error) {
	start, end, err := parseDateRange(input.StartDate, input.EndDate)
	if err != nil {
		return nil, err
	}
	if input.MaxMissingDetails == 0 {
		input.MaxMissingDetails = 200
	}
	if input.MaxMissingDetails < 1 || input.MaxMissingDetails > 1000 {
		return nil, fmt.Errorf("maxMissingDetails must be between 1 and 1000")
	}
	profile, err := rt.ResolveProfile(input.Profile)
	if err != nil {
		return nil, err
	}
	bills, billPaging, err := fetchAll(ctx, rt, input.Profile, "list_bills", "bills", map[string]any{
		"minEntryDate": input.StartDate,
		"maxEntryDate": input.EndDate,
	})
	if err != nil {
		return nil, err
	}
	attachments, attachmentPaging, err := fetchAll(ctx, rt, input.Profile, "list_attachments", "attachments", nil)
	if err != nil {
		return nil, err
	}
	organization, _, err := executeObjectRead(ctx, rt, input.Profile, "get_current_organization", "organization", apimcp.ToolInput{})
	if err != nil {
		return nil, err
	}

	attachmentsByOwner := map[string][]map[string]any{}
	for _, attachment := range attachments {
		owner := stringValue(attachment["ownerReference"])
		attachmentsByOwner[owner] = append(attachmentsByOwner[owner], attachment)
	}

	missingCount := 0
	invalidDates := 0
	coveredCount := 0
	missingDetails := make([]map[string]any, 0, min(len(bills), input.MaxMissingDetails))
	for _, bill := range bills {
		inRange, valid := dateInRange(stringValue(bill["entryDate"]), start, end)
		if !valid {
			invalidDates++
			continue
		}
		if !inRange {
			continue
		}
		billID := stringValue(bill["id"])
		attachmentIDs := billAttachmentIDs(bill, attachmentsByOwner)
		if len(attachmentIDs) > 0 {
			coveredCount++
			continue
		}
		missingCount++
		if len(missingDetails) < input.MaxMissingDetails {
			missingDetails = append(missingDetails, map[string]any{
				"billId":             billID,
				"entryDate":          bill["entryDate"],
				"contactName":        bill["contactName"],
				"suppliersInvoiceNo": bill["suppliersInvoiceNo"],
				"voucherNo":          bill["voucherNo"],
				"state":              bill["state"],
				"amount":             bill["amount"],
				"attachmentCount":    0,
			})
		}
	}
	sort.Slice(missingDetails, func(i, j int) bool {
		left := stringValue(missingDetails[i]["entryDate"]) + "\x00" + stringValue(missingDetails[i]["billId"])
		right := stringValue(missingDetails[j]["entryDate"]) + "\x00" + stringValue(missingDetails[j]["billId"])
		return left < right
	})

	complete := billPaging.Complete && attachmentPaging.Complete && invalidDates == 0
	dataThrough := ""
	if complete {
		dataThrough = input.EndDate
	}
	return map[string]any{
		"profile": profile,
		"request": map[string]any{"startDate": input.StartDate, "endDate": input.EndDate},
		"summary": map[string]any{
			"persistedBillsInRange": coveredCount + missingCount,
			"withAttachment":        coveredCount,
			"withoutAttachment":     missingCount,
		},
		"billsWithoutAttachment": missingDetails,
		"documentInboxLimitation": map[string]any{
			"pendingInboxReviewSupported": false,
			"billEmailAddress":            organization["billEmailAddress"],
			"documentedAPILimitation":     "The documented Billy API exposes the organization's billEmailAddress plus persisted bills, files, and attachments, but no documented endpoint for enumerating pending or unprocessed purchase-document inbox items.",
			"coverageMeaning":             "This result can establish attachment coverage for bills already persisted through the documented API. It cannot establish whether every document sent to Billy's purchase inbox has been processed into a bill.",
		},
		"completeness": map[string]any{
			"completeForPersistedBills": complete,
			"completeForPendingInbox":   false,
			"dataThrough":               dataThrough,
			"invalidEntryDateCount":     invalidDates,
			"missingDetailsReturned":    len(missingDetails),
			"missingDetailsComplete":    len(missingDetails) == missingCount,
			"sources": map[string]any{
				"bills":       billPaging,
				"attachments": attachmentPaging,
			},
		},
	}, nil
}

func billAttachmentIDs(bill map[string]any, attachmentsByOwner map[string][]map[string]any) []string {
	ids := map[string]bool{}
	if embedded, ok := bill["attachments"].([]any); ok {
		for _, value := range embedded {
			if attachment, ok := value.(map[string]any); ok {
				if id := stringValue(attachment["id"]); id != "" {
					ids[id] = true
				}
			}
		}
	}
	billID := stringValue(bill["id"])
	for owner, attachments := range attachmentsByOwner {
		if !ownerReferenceMatchesBill(owner, billID) {
			continue
		}
		for _, attachment := range attachments {
			if id := stringValue(attachment["id"]); id != "" {
				ids[id] = true
			}
		}
	}
	result := make([]string, 0, len(ids))
	for id := range ids {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

func ownerReferenceMatchesBill(reference, billID string) bool {
	if reference == "" || billID == "" {
		return false
	}
	return reference == billID ||
		reference == "bill:"+billID ||
		reference == "bills:"+billID ||
		strings.HasSuffix(reference, "/bills/"+billID)
}

func handleReviewVATPeriod(ctx context.Context, rt runtime, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input vatPeriodInput
	if err := apimcp.DecodeInput(request, &input); err != nil {
		return invalidInputResponse(err), nil
	}
	result, err := reviewVATPeriod(ctx, rt, input)
	if err != nil {
		return workflowErrorResponse(err), nil
	}
	return resultResponse("Billy VAT period reviewed.", result), nil
}

func reviewVATPeriod(ctx context.Context, rt runtime, input vatPeriodInput) (map[string]any, error) {
	if strings.TrimSpace(input.SalesTaxReturnID) == "" {
		return nil, fmt.Errorf("salesTaxReturnId is required")
	}
	profile, err := rt.ResolveProfile(input.Profile)
	if err != nil {
		return nil, err
	}
	vatReturn, _, err := executeObjectRead(ctx, rt, input.Profile, "get_sales_tax_return", "salesTaxReturn", apimcp.ToolInput{
		Path: map[string]any{"id": input.SalesTaxReturnID},
	})
	if err != nil {
		return nil, err
	}
	payments, paymentPaging, err := fetchAll(ctx, rt, input.Profile, "list_sales_tax_payments", "salesTaxPayments", nil)
	if err != nil {
		return nil, err
	}
	var periodPayments []map[string]any
	for _, payment := range payments {
		if stringValue(payment["salesTaxReturnId"]) == input.SalesTaxReturnID {
			periodPayments = append(periodPayments, exactFields(payment, "id", "salesTaxReturnId", "entryDate", "accountId", "amount", "side", "isVoided"))
		}
	}
	sort.Slice(periodPayments, func(i, j int) bool {
		left := stringValue(periodPayments[i]["entryDate"]) + "\x00" + stringValue(periodPayments[i]["id"])
		right := stringValue(periodPayments[j]["entryDate"]) + "\x00" + stringValue(periodPayments[j]["id"])
		return left < right
	})

	returnFields := exactFields(vatReturn,
		"id", "organizationId", "createdTime", "periodType", "period", "periodText", "correctionNo",
		"startDate", "endDate", "reportDeadline", "isSettled", "isPaid", "isPayable", "totalAmount",
		"settledAmount", "paymentDate", "skatTransactionId", "report")
	return map[string]any{
		"profile":        profile,
		"salesTaxReturn": returnFields,
		"states": map[string]any{
			"prepared": map[string]any{
				"value":    true,
				"evidence": "The exact salesTaxReturn record was retrieved.",
			},
			"settled": map[string]any{
				"value":       boolValue(vatReturn["isSettled"]),
				"sourceField": "salesTaxReturn.isSettled",
			},
			"paid": map[string]any{
				"value":       boolValue(vatReturn["isPaid"]),
				"sourceField": "salesTaxReturn.isPaid",
			},
			"submissionToTaxAuthority": map[string]any{
				"determined": false,
				"reason":     "Prepared, settled, and paid are reported separately. This read-only workflow does not infer tax-authority submission from settlement or payment state.",
			},
		},
		"filingReadiness": map[string]any{
			"determined": false,
			"reason":     "Retrieving one Billy sales-tax return does not prove field-by-field agreement with the ledger, document coverage, unresolved bank lines, foreign-service reverse charge, filing at the tax authority, or a locked/filed-period correction state.",
		},
		"salesTaxPayments": periodPayments,
		"paymentEvidence": map[string]any{
			"recordCount": len(periodPayments),
			"note":        "Sales-tax payment records are supporting evidence and are not substituted for salesTaxReturn.isPaid.",
		},
		"completeness": map[string]any{
			"complete":    paymentPaging.Complete,
			"dataThrough": vatReturn["endDate"],
			"sources": map[string]any{
				"salesTaxReturn":   map[string]any{"complete": true, "recordsFetched": 1},
				"salesTaxPayments": paymentPaging,
			},
		},
	}, nil
}

func handleReviewForeignCurrencyPurchase(ctx context.Context, rt runtime, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input foreignCurrencyPurchaseInput
	if err := apimcp.DecodeInput(request, &input); err != nil {
		return invalidInputResponse(err), nil
	}
	result, err := reviewForeignCurrencyPurchase(ctx, rt, input)
	if err != nil {
		return workflowErrorResponse(err), nil
	}
	return resultResponse("Billy foreign-currency purchase reviewed.", result), nil
}

func reviewForeignCurrencyPurchase(ctx context.Context, rt runtime, input foreignCurrencyPurchaseInput) (map[string]any, error) {
	if strings.TrimSpace(input.BillID) == "" {
		return nil, fmt.Errorf("billId is required")
	}
	profile, err := rt.ResolveProfile(input.Profile)
	if err != nil {
		return nil, err
	}
	bill, _, err := executeObjectRead(ctx, rt, input.Profile, "get_bill", "bill", apimcp.ToolInput{Path: map[string]any{"id": input.BillID}})
	if err != nil {
		return nil, err
	}
	organizationID := stringValue(bill["organizationId"])
	organization, _, err := executeObjectRead(ctx, rt, input.Profile, "get_organization", "organization", apimcp.ToolInput{Path: map[string]any{"id": organizationID}})
	if err != nil {
		return nil, err
	}
	billCurrencyID := stringValue(bill["currencyId"])
	baseCurrencyID := stringValue(organization["baseCurrencyId"])
	billCurrency, err := readCurrency(ctx, rt, input.Profile, billCurrencyID)
	if err != nil {
		return nil, err
	}
	baseCurrency := billCurrency
	if baseCurrencyID != billCurrencyID {
		baseCurrency, err = readCurrency(ctx, rt, input.Profile, baseCurrencyID)
		if err != nil {
			return nil, err
		}
	}
	allLines, linePaging, err := fetchAll(ctx, rt, input.Profile, "list_bill_lines", "billLines", nil)
	if err != nil {
		return nil, err
	}
	var lines []map[string]any
	lineAmountTotal := float64(0)
	lineTaxTotal := float64(0)
	taxRateIDs := map[string]any{}
	for _, line := range allLines {
		if stringValue(line["billId"]) != input.BillID {
			continue
		}
		lines = append(lines, exactFields(line, "id", "billId", "accountId", "taxRateId", "description", "amount", "tax", "priority"))
		if amount, ok := floatValue(line["amount"]); ok {
			lineAmountTotal += amount
		}
		if tax, ok := floatValue(line["tax"]); ok {
			lineTaxTotal += tax
		}
		if taxRateID := stringValue(line["taxRateId"]); taxRateID != "" {
			taxRateIDs[taxRateID] = true
		}
	}
	sort.Slice(lines, func(i, j int) bool {
		left, _ := integerValue(lines[i]["priority"])
		right, _ := integerValue(lines[j]["priority"])
		if left != right {
			return left < right
		}
		return stringValue(lines[i]["id"]) < stringValue(lines[j]["id"])
	})
	taxRates := make([]map[string]any, 0, len(taxRateIDs))
	for _, taxRateID := range sortedKeys(taxRateIDs) {
		taxRate, _, readErr := executeObjectRead(ctx, rt, input.Profile, "get_tax_rate", "taxRate", apimcp.ToolInput{Path: map[string]any{"id": taxRateID}})
		if readErr != nil {
			return nil, readErr
		}
		taxRates = append(taxRates, exactFields(taxRate, "id", "name", "abbreviation", "description", "rate", "appliesToSales", "appliesToPurchases", "isPredefined", "isActive", "deductionComponents"))
	}

	var paymentAccount map[string]any
	if paymentAccountID := stringValue(bill["paymentAccountId"]); paymentAccountID != "" {
		account, _, readErr := executeObjectRead(ctx, rt, input.Profile, "get_account", "account", apimcp.ToolInput{Path: map[string]any{"id": paymentAccountID}})
		if readErr != nil {
			return nil, readErr
		}
		paymentAccount = exactFields(account, "id", "name", "accountNo", "currencyId", "isPaymentEnabled", "isBankAccount")
	}

	isForeign := billCurrencyID != "" && baseCurrencyID != "" && billCurrencyID != baseCurrencyID
	checks := []map[string]any{}
	checks = append(checks, check("currencyClassification", billCurrencyID != "" && baseCurrencyID != "", map[bool]string{true: "pass", false: "warning"}[billCurrencyID != "" && baseCurrencyID != ""], "Bill and organization currency IDs are required to classify the purchase."))
	if isForeign {
		exchangeRate, exchangeRateOK := floatValue(bill["exchangeRate"])
		checks = append(checks, check("positiveBillExchangeRate", exchangeRateOK && exchangeRate > 0, map[bool]string{true: "pass", false: "error"}[exchangeRateOK && exchangeRate > 0], "A foreign-currency bill should carry a positive bill.exchangeRate."))
	}
	if paymentAccount != nil {
		accountCurrencyID := stringValue(paymentAccount["currencyId"])
		checks = append(checks, check("paymentAccountCurrencyKnown", accountCurrencyID != "", map[bool]string{true: "pass", false: "warning"}[accountCurrencyID != ""], "The payment account currency is needed to assess later settlement exchange differences."))
	}
	if isPaid := boolValue(bill["isPaid"]); isPaid {
		balance, balanceOK := floatValue(bill["balance"])
		checks = append(checks, check("paidBillHasZeroBalance", balanceOK && mathAbs(balance) < 0.000001, map[bool]string{true: "pass", false: "warning"}[balanceOK && mathAbs(balance) < 0.000001], "Billy reports the bill as paid; its remaining balance should also be reviewed."))
	}

	return map[string]any{
		"profile": profile,
		"bill": exactFields(bill,
			"id", "organizationId", "contactId", "contactName", "entryDate", "paymentDate", "dueDate", "state",
			"suppliersInvoiceNo", "voucherNo", "taxMode", "amount", "tax", "currencyId", "exchangeRate", "balance", "isPaid", "paymentAccountId", "balanceModifiers"),
		"organization":   exactFields(organization, "id", "name", "baseCurrencyId", "defaultTaxMode", "defaultBillBankAccountId"),
		"billCurrency":   exactFields(billCurrency, "id", "name", "exchangeRate"),
		"baseCurrency":   exactFields(baseCurrency, "id", "name", "exchangeRate"),
		"paymentAccount": paymentAccount,
		"classification": map[string]any{
			"isForeignCurrency": isForeign,
			"billCurrencyId":    billCurrencyID,
			"baseCurrencyId":    baseCurrencyID,
		},
		"billLines": lines,
		"vatTreatment": map[string]any{
			"independentFromCurrency": true,
			"billTaxMode":             bill["taxMode"],
			"billTax":                 bill["tax"],
			"lineTaxTotal":            lineTaxTotal,
			"taxRates":                taxRates,
			"reverseChargeDetermined": false,
			"reason":                  "Currency classification does not determine VAT treatment. Review the returned exact Billy tax-rate records and their deduction components against the supplier location and service type.",
		},
		"lineTotals": map[string]any{
			"amount": lineAmountTotal,
			"tax":    lineTaxTotal,
		},
		"checks":         checks,
		"rateLimitation": "The bill.exchangeRate is the transaction-specific Billy field. Currency.exchangeRate values are current Billy currency records and are not treated as independent historical market-rate evidence for the bill entry date.",
		"completeness": map[string]any{
			"complete":    linePaging.Complete,
			"dataThrough": bill["entryDate"],
			"sources": map[string]any{
				"bill":         map[string]any{"complete": true, "recordsFetched": 1},
				"organization": map[string]any{"complete": true, "recordsFetched": 1},
				"billLines":    linePaging,
			},
		},
	}, nil
}

func readCurrency(ctx context.Context, rt runtime, profileID, currencyID string) (map[string]any, error) {
	if currencyID == "" {
		return map[string]any{}, nil
	}
	currency, _, err := executeObjectRead(ctx, rt, profileID, "get_currency", "currency", apimcp.ToolInput{Path: map[string]any{"id": currencyID}})
	return currency, err
}

func check(name string, passed bool, severity, explanation string) map[string]any {
	return map[string]any{
		"name":        name,
		"passed":      passed,
		"severity":    severity,
		"explanation": explanation,
	}
}

func mathAbs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
