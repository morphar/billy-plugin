# Billy Plugin

> [!WARNING]
> **Beta / AI-generated code:** This is experimental beta software, and much of its code, documentation, and Billy OpenAPI contract was generated with AI assistance under human direction and review. Automated tests, code signing, and notarization do not guarantee correct bookkeeping behavior. Test with a non-production Billy account, keep write tools disabled unless needed, and independently verify every accounting result and change.

Billy Plugin packages a fully local Billy MCP server together with an accounting workflow skill for ChatGPT desktop and Codex.

The plugin does not use a hosted intermediary. Its standalone Go executable runs on the user's computer, reads the Billy credential from the local operating system credential store, calls Billy directly, and redacts exact credential values from model-visible responses and errors.

This is an independent, unofficial project. It is not affiliated with, endorsed by, or sponsored by Billy.

## Current beta target

The first beta release supports macOS on Apple Silicon. Users do not need Go or Node.js installed.

The executable in a source checkout is an ad-hoc-signed development build. Official GitHub release archives are built from tagged api-mcp source, signed with an Apple Developer ID, submitted to Apple's notarization service, and published with checksums.

## Install a release

Download the macOS Apple Silicon archive and `SHA256SUMS` from the GitHub release. Verify the archive, extract it, then add the extracted directory as a local marketplace:

```bash
shasum -a 256 -c SHA256SUMS
codex plugin marketplace add /absolute/path/to/billy-plugin
codex plugin add billy-plugin@billy
```

Restart ChatGPT desktop or Codex and start a new task so the bundled skill and MCP tools are loaded. No Go or Node.js installation is required.

## Install from this checkout

Install the plugin, restart ChatGPT desktop, and start a new task so the bundled
skill and MCP tools are loaded. The first Billy API call will open a native
macOS secure-input dialog if the credential is not already in Keychain.

Manual enrollment and status checks remain available for the `default` profile:

```bash
./plugins/billy-plugin/scripts/setup-macos
./plugins/billy-plugin/scripts/credential-status-macos
```

Alternatively, double-click `plugins/billy-plugin/Setup Billy.command` in Finder.

Add this repository as a local marketplace and install the plugin:

```bash
codex plugin marketplace add /absolute/path/to/billy-plugin
codex plugin add billy-plugin@billy
```

## Safe tool configuration

The first MCP start automatically creates `~/Library/Application Support/Billy Plugin/config.json` with only the named `read-only` feature enabled. No create, update, or delete endpoint is available by default.

ChatGPT, Codex, or Claude can explain and manage this policy through `get_tool_configuration`, `list_tool_features`, `search_available_tools`, and the single mutating configuration tool, `configure_tools`. Configuration changes are validated against the bundled OpenAPI contract, then a native macOS dialog shows the selected account, requested change, and resulting policy. Only local approval lets the MCP persist and apply the change; a model cannot approve it by passing a flag.

Multiple Billy accounts are managed by one local MCP process. `list_api_profiles`, `add_api_profile`, `rename_api_profile`, and `remove_api_profile` manage a token-free local registry. Profile changes require the same local native approval. Each profile has its own Keychain item and read-only-by-default tool policy, and API calls require an explicit profile whenever more than one account exists. Existing `default` credentials and settings migrate in place.

Every Billy create, update, or delete request is validated and checked against the selected profile's policy, then displayed in a native macOS approval dialog before the token is read and before Billy is contacted. Read-only Billy requests do not show this dialog. Credential setup continues to use a separate hidden-input dialog.

The plugin metadata remains in English because the current manifest has one description field rather than locale-specific variants. The bundled starter prompts and documentation include Danish examples, so Danish requests work naturally in ChatGPT, Codex, and Claude.


## Repository layout

- `.agents/plugins/marketplace.json` exposes the repository as the `billy` marketplace.
- `plugins/billy-plugin/.codex-plugin/plugin.json` defines the installable plugin.
- `plugins/billy-plugin/.mcp.json` starts the local MCP over stdio.
- `plugins/billy-plugin/skills/accounting` contains the accounting workflow and safety policy.
- `plugins/billy-plugin/config/billy.json` defines the immutable local server settings; `config/features.json` explains the supported feature groups; the active user policy lives outside the plugin cache.
- `plugins/billy-plugin/bin` contains release executables, never credentials.

## Maintainer build

The generic MCP engine currently lives in a sibling `api-mcp` checkout. Its clean `HEAD` must match the full hash in `API_MCP_COMMIT`. Rebuild the macOS executable with:

```bash
./plugins/billy-plugin/scripts/build-macos-arm64
```

Set `API_MCP_SOURCE=/absolute/path/to/api-mcp` when the checkout is elsewhere. The build does not embed the Billy token or configuration.

Release requirements and signing-secret setup are documented in [docs/RELEASING.md](docs/RELEASING.md). Security and local-data boundaries are documented in [SECURITY.md](SECURITY.md) and [PRIVACY.md](PRIVACY.md).

## License

MIT. See [LICENSE](LICENSE).
