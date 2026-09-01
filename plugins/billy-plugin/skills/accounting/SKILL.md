---
name: accounting
description: Review and reconcile Billy accounting data, propose entries, assess VAT/tax effects, and safely operate the bundled Billy tools. Use for Billy ledger corrections or bookkeeping questions; verify company accounts, jurisdiction, and filed periods before advising or writing.
---

# Accounting

## Billy Plugin Context

- Use the bundled Billy MCP as the source of live bookkeeping data.
- Never ask the user to paste a Billy token into chat or pass it as a tool argument. The local MCP reads it from the operating system credential store.
- Call `list_api_profiles` when the intended Billy account is not already unambiguous. When multiple profiles exist, pass the stable `profile` ID on every Billy API call. Never infer a profile from record IDs or reuse IDs read from another profile.
- `add_api_profile` creates a separate profile and opens the native macOS secure-input dialog for that account's token. New IDs start read-only. A removed ID with a preserved policy requires `restoreToolConfiguration: true`; explain that restoration explicitly and never set it by default. Profile mutations also open a native local approval dialog. Use `rename_api_profile` for display names. Before `remove_api_profile`, explain whether its Keychain credential and tool-policy file will also be deleted.
- If credentials are missing, call `api_mcp_setup_credential` with the intended `profile` and tell the user to complete the native dialog. After success, retry the original Billy read. Use the plugin's local setup command only for the default profile when the setup tool is unavailable. Do not attempt to retrieve or display a stored credential.
- Use `api_mcp_credential_status` only to check the selected profile's credential availability; its result must never contain credential material.
- Each new profile ID starts with only the `read-only` feature enabled. Use `get_tool_configuration`, `list_tool_features`, and `search_available_tools` with the intended `profile` to inspect or explain its permissions.
- `configure_tools` is the only MCP tool that may change a profile's tool availability. Before calling it, identify the exact profile, explain the exact features or endpoints that will be added or removed, and call out any write capability. The MCP then requires approval in a native local dialog showing the resulting policy. If the request is ambiguous, use the read-only discovery tools first.
- Enabling a feature or endpoint only changes which MCP tools are available. It is a technical ceiling, not blanket authorization to change live bookkeeping data; the per-operation approval and verification rules below still apply.

## Operating Principles

Treat accounting records as production financial data.

- Start with evidence: bank statements, card statements, invoices, receipts, accounting exports, screenshots, live MCP reads, and user explanations.
- Separate facts from judgement. State what is proven, what is inferred, and what needs accountant/user confirmation.
- Use double-entry reasoning. Every proposed fix must name debit account, credit account, amount, date, VAT/tax treatment, and rationale.
- Preserve audit trail. Prefer reversal, replacement, or supplemental correction over silent edits of posted records.
- Do not invent account IDs, tax rates, supplier IDs, invoice IDs, bank-line IDs, or legal/tax rules.
- Do not mutate live accounting data until the discrepancy, plan, and user approval are clear.
- Before every live write, restate the target Billy profile name and stable ID together with the intended bookkeeping change. Treat a profile mismatch as a hard stop.
- Expect the local MCP to show the exact validated write in a native macOS approval dialog. Cancellation means no credential is read and no Billy request is sent; do not retry unless the user asks.
- Stop and ask before changing submitted VAT/sales-tax periods, filed years, payroll, tax filings, locked periods, loans, equity, or materially uncertain tax treatment.
- If the Billy tools are missing, stale, or attached from another plugin version, stop and ask the user to restart the client and begin a new task. Do not launch copied, patched, ad-hoc-signed, or directly invoked MCP binaries as a workaround; macOS Keychain access is bound to the installed executable's identity.

## Session Ledger And Completion Gate

For every task that reviews more than one record, keep one authoritative working ledger. It may be an in-memory structure for a small review or a local CSV/JSON file for a large review, but it must contain:

- stable Billy profile ID and display name
- organization ID/name, jurisdiction, accounting basis, fiscal year, VAT status, and filed/locked periods in scope
- each source examined, its account/period, latest source timestamp, pages read, records read, and completeness state
- every discrepancy or work item with amount, date, source record IDs, classification, evidence, intended treatment, status, and verification result
- every live write with its target IDs, approved intent, result IDs, and read-back evidence

Use only these completeness and visibility states:

- `FOUND`: the requested record or state was found.
- `NOT_FOUND_COMPLETE`: every relevant page and supported source was read and the record was not found.
- `PAGINATION_INCOMPLETE`: one or more relevant result pages were not read or a response was truncated.
- `API_ACCESS_DENIED`: Billy rejected the API resource or operation.
- `UNLINKED`: the object exists but is not linked to the owner/transaction needed by the queried API resource.
- `UI_ONLY`: the web app exposes the object but the documented API does not.
- `WRONG_PROFILE`: the queried Billy profile is not the intended company.
- `WRONG_PERIOD`: the record is outside the reviewed accounting or statement period.

API absence is never proof of business absence unless the result is `NOT_FOUND_COMPLETE`. When the user can see a record in Billy that the MCP cannot see, report the current state and use the signed-in browser for read-only confirmation when available. Do not call undocumented private Billy endpoints.

Do not say `done`, `fully reconciled`, `ready to file`, or an equivalent completion claim until all of these are true:

1. Every in-scope source is `FOUND` or `NOT_FOUND_COMPLETE`, not incomplete or inaccessible without an explicit limitation.
2. Every work item is resolved, intentionally excluded with rationale, or listed as a remaining blocker.
3. All touched records were re-read after writes.
4. Account balances, bank-line approval/associations, VAT fields, void states, and remaining unmatched items were recomputed as relevant.
5. The final answer names the data-through timestamp and any UI-only or external-system limitation.

## First Pass

1. Identify context.
   - Company, jurisdiction, accounting basis, fiscal year, VAT/sales-tax status, accounting system, and whether any periods are filed/closed.
   - Source of truth for each question: bank for cash balances, supplier invoice for expense/VAT, tax authority statement for tax settlement, accounting system for posted ledger state.

2. Build a local model before advising.
   - Map chart-of-account numbers/names to live account IDs.
   - Pull or parse bank lines, card lines, postings, bills, invoices, payments, tax rates, VAT returns, and daybooks as needed.
   - Use conservative pagination with MCPs that cap responses.
   - Create working CSVs/notes when analysis spans many transactions.
   - Record page counts and response `truncated` flags in the session ledger. A page-size cap or result count matching the requested page size is not evidence that the final page was reached.

3. Classify the task.
   - Classification: decide how a transaction should be booked.
   - Reconciliation: explain and eliminate difference between ledger and statement.
   - Correction: reverse/reclassify/delete/recreate wrong entries.
   - VAT/tax review: determine whether reporting was affected.
   - Closing: identify unresolved balances, stale receivables/payables, missing accruals, or owner/equity cleanup.

## Preferred Billy Workflow Tools

Use the fixed Billy workflow tools before composing the same review from raw endpoints:

- `api_mcp_diagnostics`: verify the running plugin/spec version, response limit, profile IDs, credential availability, and feature policy.
- `diagnose_billy_connection`: make a harmless Billy read and verify the selected profile/organization.
- `review_billy_bank_account`: page through one bank/card account, separate booked from reconciled lines, and identify duplicate candidates and source freshness.
- `review_billy_document_coverage`: compare bills and linked attachments for a period while preserving the documented-API inbox limitation.
- `review_billy_vat_period`: return the exact Billy VAT period fields and keep prepared, settled, paid, and externally submitted states separate.
- `review_billy_foreign_currency_purchase`: inspect supplier currency, Billy exchange rate/local value, payment evidence, and realized currency difference separately.
- `preview_billy_bank_line_cleanup`: re-read exact candidate lines and produce the deterministic cleanup digest without writing.
- `commit_billy_bank_line_cleanup`: re-read and compare the digest, reject newly associated or changed lines, request one approval for the exact batch, and verify every deletion.

Raw OpenAPI tools remain the fallback for unsupported records and targeted evidence. A workflow error or incomplete result does not authorize bypassing profile policy, local approval, digest checks, or the completion gate.

## Common Booking Patterns

Supplier purchase:

- Bill/receipt controls expense account and VAT code.
- Payment account controls cash/card/bank movement.
- If only the payment account was wrong, fix payment account; do not change expense/VAT lines.

Bank-to-card funding:

- Debit card/payment account.
- Credit bank account.
- No VAT and no expense.

Card purchase:

- Book purchase to supplier/expense or bill payable.
- Settle payment from card account.
- Fees on the card are separate fee expenses unless the receipt says otherwise.

Bank or card fee:

- Debit bank/card fee expense.
- Credit the account where the fee occurred.
- Usually no input VAT unless local evidence shows VAT on the fee.

Owner deposit:

- Debit bank.
- Credit owner/equity/private contribution account.
- Not revenue and normally not VAT.

Owner withdrawal:

- Debit owner/equity/private withdrawal account.
- Credit bank.
- Not expense and normally not VAT.

VAT/tax settlement:

- Use the relevant VAT/tax payable or receivable account.
- Match the actual bank amount.
- Split tax-account components only when supported by tax authority text or statement detail.

Opening or historical difference:

- If the prior period is open, correct in that period or reverse/recreate the old correction if the system supports a clear audit trail.
- If the period is closed/filed, correct in the first open period and flag accountant review.
- Do not use sales, expense, or VAT accounts just to force a bank balance.

FX and foreign currency:

- Keep the supplier bill in the currency shown on the supplier document when available.
- Use the DKK/local card or bank settlement for payment reconciliation.
- Post realized exchange differences separately when the accounting system does not handle them automatically.

Assets, subscriptions, and prepayments:

- Expense low-value ordinary operating items unless local policy or law requires capitalization.
- Capitalize durable significant assets when appropriate.
- Use prepaid expense or accrual accounts when the cost or income belongs materially to another period.

## Reconciliation Workflow

1. Compute ledger balance from non-voided postings.
2. Compare it to the statement balance for the same account/date.
3. Build a daily bridge: statement balance, ledger balance as-of date, difference, difference change, ledger movement.
4. Investigate jumps in difference by date and amount.
5. Classify each gap: missing bank line, duplicate import, wrong account, wrong side, wrong date, unposted fee, owner movement, tax settlement, opening correction, or stale match.
6. Plan the smallest set of auditable corrections.
7. Execute only after approval.
8. Recompute final balances and check match status.

If same-day bank-line order is ambiguous, reconstruct order from running balances instead of trusting API list order.

Treat bank-line state precisely:

- `bankLine.status = booked` does not prove reconciliation.
- A line is reconciled only when its match is approved and its subject associations fully explain the amount and direction, unless Billy exposes a stronger documented invariant.
- Report the latest bank-feed timestamp separately from the accounting date. A balanced ledger cannot prove that a stale bank feed includes later real-world movements.

## Document Coverage Workflow

1. Read bills/purchases and their linked attachments for the target period.
2. Compare those records to bank/card lines and known suppliers.
3. Treat an uploaded but unprocessed Billy inbox file as `UI_ONLY` or `API_ACCESS_DENIED` when the documented API cannot list it.
4. After the user creates a purchase, re-read the resulting bill and attachment owner reference before marking the document covered.
5. Never report a zero attachment result as "no uploaded receipts" without a complete supported inbox source.

## Foreign-Currency Purchase Workflow

1. Keep the bill amount and currency exactly as shown by the supplier document.
2. Read the actual DKK/local-currency bank or card settlement separately.
3. Verify whether Billy created the expected local-currency payable, payment, and realized exchange-difference postings.
4. Reconcile the bank/card line to the local-currency payment amount, not to the foreign face amount.
5. Report bill currency/amount, booked DKK value, paid DKK amount, exchange difference, payment account, and VAT treatment as separate fields.

## VAT Readiness Workflow

Before saying a VAT period is ready to file:

1. Confirm the intended period, filing frequency, jurisdiction, filed/correction state, and data-through timestamp.
2. Recompute each VAT return field from the live Billy return/report and relevant postings; do not validate only the net payable amount.
3. Check document coverage, unresolved bank/card lines, duplicate imports, foreign-service reverse charge, owner movements, fees, and changes in already-filed periods.
4. Present an exact field-by-field manual filing fallback whenever Billy-to-tax-authority transfer is unavailable or fails.
5. After filing, distinguish `prepared`, `submitted`, `settled in Billy`, `paid at the authority`, and `bank payment reconciled`. These are separate states.
6. A missing tax-authority transaction ID after manual filing is not proof that filing failed; use the return status and settled amount together and state the limitation.

## VAT And Tax Review

Payment-account and reconciliation fixes usually do not change VAT if bill/invoice lines and VAT codes are unchanged.

Check VAT/tax impact explicitly:

- Did any correction touch VAT/tax accounts?
- Did any posting have `taxRateId`, VAT code, or VAT return linkage?
- Was an already submitted VAT/sales-tax period changed?
- Did the fix change expense/revenue timing or taxable profit?

When the user asks about legal/tax correctness, use current authoritative sources or advise accountant review. Do not rely on memory for changing laws, filing deadlines, rates, or threshold rules.

## Live MCP/API Work

For accounting MCP writes:

- Read [references/mcp-patterns.md](references/mcp-patterns.md) before mutating live accounting records.
- Re-read target records immediately before writing.
- Add guard checks to avoid duplicate correction postings.
- Use clear transaction text and `apiType`/metadata when available.
- Verify after writes: balance, bank-line match, tax fields, void state, and remaining unmatched items.

## Response Shape

For analysis-only answers, lead with the conclusion and evidence.

For proposed fixes, include:

- exact entries: date, debit, credit, amount, VAT/tax treatment
- why this is correct
- what it affects: bank balance, profit, VAT, tax, equity, receivables/payables
- risk or accountant-review points

For completed API work, end with:

- status
- records changed
- verification result
- important decisions
- remaining risks
- recommended next step
- data-through timestamp and completeness/visibility limitations
