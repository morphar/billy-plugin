# Contributing

Thank you for helping improve Billy Plugin.

Use a GitHub issue before substantial behavior, permission-model, packaging, or public-contract changes. Report security issues privately as described in [SECURITY.md](SECURITY.md).

Contributions must preserve these invariants:

- no Billy credential in plugin configuration, logs, MCP inputs, or MCP results
- read-only tool access on first start and for every new profile
- explicit profile selection when more than one account exists
- server-side policy enforcement before credential resolution or HTTP requests
- explicit user confirmation before tool access is expanded or live bookkeeping is changed

Run `./scripts/validate` from the repository root before opening a pull request. Changes to the MCP engine belong in `cortexium-io/api-mcp`; update `API_MCP_VERSION` only after that engine version has been released.

By submitting a contribution, you agree that it is licensed under the repository's MIT License and that you have the right to provide it under those terms.
