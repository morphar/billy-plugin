# MCP Patterns For Accounting Work

Use this reference before live writes through Billy-style or similar accounting MCPs.

## Discovery

Discover IDs dynamically for each company:

- current organization
- accounts, account groups, account natures
- tax rates and VAT/sales-tax accounts
- daybooks
- bank lines and bank-line matches
- bank-line subject associations
- postings and transactions
- bills, invoices, and payments
- contacts, attachments, files, and VAT returns when relevant

Never reuse IDs from another company or prior session unless just re-read from the active MCP.

When list tools are paginated, use smaller page sizes if responses truncate. Some MCP wrappers cap returned text even when the API supports large pages.

Record `page`, `pageSize`, returned-record count, total/page metadata when available, and `truncated` for every collection. Continue until the API proves the final page. If the wrapper truncates JSON, discard the partial body and retry with a smaller page size; never infer from a partial payload.

## Balance Calculation

For asset bank/card accounts:

```text
ledger_balance = sum(non_voided debit postings) - sum(non_voided credit postings)
bank_difference = latest_bank_running_balance - ledger_balance
```

Use `isVoided`/`isVoid` fields to exclude voided records. Use `entryDate` for accounting date analysis and `createdTime` only for audit/history questions.

## Relationship Map

Typical Billy-style relationship:

```text
bankLine.matchId -> bankLineMatch.id
bankLineSubjectAssociation.matchId -> bankLineMatch.id
bankLineSubjectAssociation.subjectReference -> "posting:<postingId>"
posting.transactionId -> transaction.id
transaction.originatorReference -> bill, invoice, bank payment, daybook transaction, etc.
```

Use live reads to confirm exact field names for the active MCP.

## Daybook Posting Pattern

For simple two-sided corrections, many Billy-style APIs support one daybook transaction line with a contra account:

```json
{
  "daybookTransaction": {
    "organizationId": "<organizationId>",
    "daybookId": "<daybookId>",
    "entryDate": "YYYY-MM-DD",
    "description": "Short description",
    "extendedDescription": "Evidence and rationale",
    "apiType": "codex-accounting-correction",
    "state": "approved",
    "lines": [
      {
        "text": "Line text",
        "accountId": "<debitAccountId>",
        "contraAccountId": "<creditAccountId>",
        "amount": 100.00,
        "side": "debit",
        "taxRateId": null
      }
    ]
  }
}
```

Verify resulting postings rather than trusting the request shape.

## Bank-Line Matching

After creating a posting intended to reconcile a bank line:

1. Create the accounting transaction/posting.
2. Create a subject association:

```json
{
  "bankLineSubjectAssociation": {
    "matchId": "<bankLineMatchId>",
    "subjectReference": "posting:<postingId>"
  }
}
```

3. Approve the match only when amount and direction fully explain the bank line:

```json
{
  "bankLineMatch": {
    "isApproved": true,
    "differenceType": null,
    "feeAccountId": null
  }
}
```

4. Re-read the match and associations to verify approval and no unintended difference.

## Duplicate Bank Lines

Accounting APIs may contain duplicate bank lines from multiple imports/providers. Before matching:

- Compare date, amount, side, description, balance, provider, external ID, and group ID.
- Pick the source the user is actively reconciling against.
- Do not create multiple accounting entries for duplicate representations of the same real movement.
- Report duplicate unapproved matches as import cleanup candidates.

For destructive cleanup, separate preview from commit. The preview must include exact line IDs, provider/group/external IDs, dates, descriptions, amounts, running balances, match/association state, count, signed sum, and a deterministic digest. Commit must re-read every target before approval, require one approval for the complete batch, then immediately re-read and reject changed, approved, or associated targets before any delete. This narrows but does not eliminate an external race when the upstream delete endpoint has no conditional-write precondition.

## Guard Checks Before Writes

Before creating a correction:

- Search existing postings by account, date, amount, side, text, and `apiType`.
- Check whether the bank-line match already has associations.
- Check whether the target bill/payment/invoice is already voided, paid, matched, or in a closed/submitted period.
- Verify the tax rate and VAT return linkage are intentional.

If any guard fails, stop and report instead of making a second correction.

## Verification After Writes

Always re-read and verify:

- ledger balances for touched accounts
- bank-line match approval and difference fields
- created postings and transaction text
- `taxRateId` and VAT/sales-tax return linkage
- void state of replaced records
- remaining unmatched or excluded items

For a multi-record operation, also verify the preview digest, attempted/succeeded/skipped/failed counts, per-record result, idempotent retry behavior, and the final aggregate balance. Never report the batch complete when only the individual API requests succeeded.
