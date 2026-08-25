# Billy Plugin

> [!WARNING]
> **Beta / AI-generated code:** This is experimental beta software, and much of its code, documentation, and Billy OpenAPI contract was generated with AI assistance under human direction and review. Automated tests, code signing, and notarization do not guarantee correct bookkeeping behavior. Test with a non-production Billy account, keep write tools disabled unless needed, and independently verify every accounting result and change.

Billy Plugin is a local-first accounting plugin for ChatGPT desktop, Codex, and other MCP clients such as Claude. It bundles a Billy MCP server, a safe tool configuration, and an accounting workflow skill. The MCP executable runs locally and calls Billy directly; there is no hosted intermediary.

This is an independent, unofficial project. It is not affiliated with, endorsed by, or sponsored by Billy.

## Supported platform

The initial package supports macOS on Apple Silicon. Other platforms need their own prebuilt executable and credential-store adapter before they can be advertised as supported.

The checked-in executable is an ad-hoc-signed development build. A public macOS release must be rebuilt from committed source, signed with an Apple Developer ID, notarized, and accompanied by the updated `SHA256SUMS` file.

## First-use credential setup

The MCP can start without a stored Billy token. On the first Billy API call for
a selected profile, it opens a native macOS secure-input dialog, asks for that
profile's token twice, stores it directly in Keychain, and continues the
requested operation. ChatGPT receives only setup status; the token is never an
MCP tool argument or result.

The bundled `api_mcp_setup_credential` tool can open the same dialog explicitly,
and `api_mcp_credential_status` reports only whether a Keychain item exists.

Manual setup remains available for the `default` profile from the plugin directory:

```bash
./scripts/setup-macos
./scripts/credential-status-macos
```

In Finder, macOS users can instead double-click `Setup Billy.command` and follow the non-echoing prompts.

Both setup paths read the credential twice without echo and store it as a
device-only, non-synchronizing macOS Keychain item. The credential is never
accepted as a command-line argument or MCP tool input.

`BILLY_ACCESS_TOKEN` remains available as an environment override for development. When it is unset, the MCP falls back to Keychain.

## Tool configuration

On first start, the MCP automatically creates this user-local configuration:

`~/Library/Application Support/Billy Plugin/config.json`

```json
{
  "version": 1,
  "enabledFeatures": ["read-only"],
  "enabledEndpoints": [],
  "disabledEndpoints": []
}
```

The default is deny-by-default: only Billy `GET` endpoints are available. No create, update, or delete endpoint is enabled. The file contains tool permissions only—never the Billy token—and is created with owner-only permissions.

The MCP exposes four configuration tools:

- `get_tool_configuration` shows the selected profile's configuration and its path.
- `list_tool_features` explains every named feature, risk level, and endpoint group for the selected profile.
- `search_available_tools` searches and explains individual endpoints and reports whether each is enabled for the selected profile.
- `configure_tools` adds or removes named features or individual endpoint selectors for the selected profile, saves the user-local file atomically, and refreshes the live MCP tool list.

`configure_tools` is the only configuration-changing tool. The bundled accounting skill instructs the assistant to explain the exact access change and obtain explicit confirmation before calling it, especially when enabling writes. Changing tool availability does not authorize an accounting write; live bookkeeping changes still require their own explicit approval and post-write verification.

Example requests:

- “Explain which Billy feature groups are available.”
- “Which tools would let you reconcile a bank line?”
- “Enable bank reconciliation tools.”
- “Disable deletion of bank-line matches.”
- “Show my current Billy tool configuration.”

Eksempler på dansk:

- “Forklar hvilke Billy-funktioner der er tilgængelige.”
- “Hvilke værktøjer skal du bruge for at afstemme en banklinje?”
- “Aktivér værktøjer til bankafstemning for `default`.”
- “Deaktivér sletning af banklinjematches.”
- “Vis min nuværende Billy-værktøjskonfiguration.”

### Bundled feature groups

| Feature | Risk | Enables |
| --- | --- | --- |
| `read-only` | Read | All Billy `GET` endpoints |
| `bank-payments` | Write | Create and update bank payments |
| `bank-reconciliation` | Write | Create, update, and delete bank-line matches and their subject associations |
| `daybook-corrections` | Write | Create and update daybook transactions and lines |

Individual selectors may be an MCP tool name such as `delete_bank_line_match`, an OpenAPI operation ID, a method/path such as `DELETE /bankLineMatches/{bankLineMatchId}`, a tag selector such as `tag:BankLineMatches`, or a wildcard. An entry in `disabledEndpoints` overrides feature and endpoint enables.

You may edit the user-local JSON manually, but MCP-driven changes are safer because they are validated against the bundled Billy OpenAPI contract and take effect immediately. Restart the MCP client after a manual edit. Do not edit configuration inside the installed plugin cache; upgrades replace those files.

## Multiple Billy accounts

The plugin supports multiple local Billy account profiles. Every profile has:

- a stable lowercase ID and friendly display name
- a separate token stored under a distinct macOS Keychain account
- a separate read-only-by-default tool policy
- server-side permission enforcement before any Billy HTTP request

The first profile is created automatically as `default`. It reuses the existing `com.morphar.billy-plugin` / `default` Keychain item and the existing `config.json` tool policy, so upgrading does not copy, expose, or reset the current token or settings.

Profile metadata is stored in:

`~/Library/Application Support/Billy Plugin/accounts.json`

```json
{
  "version": 1,
  "profiles": [
    {"id": "default", "name": "Default Billy account"},
    {"id": "company-b", "name": "Company B"}
  ]
}
```

This registry never contains tokens. New profile tokens use Keychain accounts such as `profile:company-b`, and their policies live at `~/Library/Application Support/Billy Plugin/profiles/company-b/tools.json`.

Account tools:

- `list_api_profiles` lists profiles, credential availability, policy paths, and enabled endpoint counts without reading token values into chat.
- `add_api_profile` creates a profile and immediately opens the native secure token dialog. A new ID starts read-only; reusing a previously removed ID restores its preserved policy if that policy file was not deleted.
- `rename_api_profile` changes only the friendly name.
- `remove_api_profile` removes a profile; optional flags separately control permanent deletion of its Keychain credential and policy file. At least one profile must remain.
- `api_mcp_credential_status` and `api_mcp_setup_credential` operate on the selected profile.

Example requests:

- “Add a Billy account named Company B with the ID `company-b`.”
- “Show my Billy accounts and their enabled features.”
- “Enable bank reconciliation only for `company-b`.”
- “Rename `default` to Main Company.”
- “Remove `company-b`, but keep its Keychain token and settings for recovery.”

Eksempler på dansk:

- “Tilføj en Billy-konto med navnet Firma B og ID'et `firma-b`.”
- “Vis mine Billy-konti og deres aktiverede funktioner.”
- “Aktivér kun bankafstemning for `firma-b`.”
- “Omdøb `default` til Hovedfirma.”
- “Fjern `firma-b`, men behold token og indstillinger i Nøglering.”

There is deliberately no mutable global active account. When more than one profile exists, Billy API tools require the `profile` ID. Tool results also identify the profile used. This prevents one chat or accounting operation from silently switching another account.

The MCP tool list is shared across profiles. If a write tool is enabled for one profile, it is visible globally, but the server rejects it for every profile whose own policy does not allow it before making an HTTP request.

Changes made through profile and configuration tools update the current MCP process immediately. Restart another already-running MCP client after changing profiles or policies from ChatGPT, Codex, Claude, or a manual file edit so that client reloads the registry.

## Privacy and security

The plugin has no hosted service, telemetry, or analytics. Requests needed to answer the user's accounting task travel directly from the local MCP process to Billy's API. Model-visible tool results can contain accounting data, so users must also consider the privacy settings and terms of their selected AI client.

Billy tokens are stored in macOS Keychain and are not exposed as MCP inputs or results. For the complete boundaries and reporting process, see the repository's `PRIVACY.md` and `SECURITY.md`.

## License

MIT. The license and third-party notices are bundled with the plugin.
