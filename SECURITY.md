# Security Policy

## Supported versions

Security fixes are made on the latest published release. The initial supported runtime is macOS on Apple Silicon.

## Reporting a vulnerability

Report suspected vulnerabilities privately through this repository's GitHub Security Advisory form. Do not open a public issue for token exposure, Keychain access, authorization or tool-policy bypasses, profile isolation failures, unsafe update behavior, or exposure of private accounting data.

Include the affected plugin version, macOS version, reproduction steps, and expected impact. Remove organization IDs, accounting records, tokens, Keychain exports, and other private data before submitting the report. You should receive an acknowledgement within seven days.

## Security boundaries

- The MCP executable and configuration run locally; there is no plugin-operated intermediary.
- macOS Keychain prevents the token from becoming plugin configuration or a model-visible tool argument. It does not protect against a compromised local account or executable.
- The bundled policy starts read-only and checks the selected profile and OpenAPI-declared input before resolving a credential or sending a Billy request.
- Tool-policy changes, profile changes, and every non-read Billy request require a native local macOS approval after validation and before credentials or side effects. Approval is server-enforced and cannot be supplied as an MCP argument by a model.
- Enabling a write tool only makes that request type technically available; each live bookkeeping change still requires its own approval.
- Tool responses contain live Billy data and are visible to the selected AI client.
- Official release archives are signed, submitted for Apple notarization, and published with SHA-256 checksums. Source-checkout binaries are development builds and must not be presented as official releases.
