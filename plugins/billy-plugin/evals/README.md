# Accounting regression cases

`accounting-regressions.json` captures failure modes observed in real Billy sessions. Use every case when reviewing changes to the accounting skill, OpenAPI contract, or custom workflow tools.

Each case states the minimum expected evidence and an unsafe claim or action that must not occur. The executable Go tests cover transport, policy, pagination, digest, and batch behavior; these cases cover the model-facing accounting decisions that code-level tests cannot prove by themselves.
