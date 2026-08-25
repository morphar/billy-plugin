# Privacy

[Dansk](PRIVACY.da.md) | **English**

Billy Plugin is fully local middleware. It does not operate a hosted service and does not collect analytics, telemetry, credentials, or accounting records for the plugin author.

## Data flow

- The local `billy-mcp` process reads the selected Billy token from macOS Keychain or, for development, the `BILLY_ACCESS_TOKEN` environment variable.
- It sends requests directly to Billy's API over HTTPS.
- It returns the requested API response to the AI client through the local MCP connection.
- Profile names and tool policies are stored under the current user's application-support directory. Tokens are stored separately in macOS Keychain.

Tool responses may contain personal, financial, or customer data. That response is visible to the AI client handling the conversation and is governed by that client's configuration and privacy terms. Billy's handling of API requests is governed by Billy's own terms and privacy policy.

The plugin does not intentionally log credential values. Exact configured credential values are redacted from upstream responses and errors before they are returned through MCP, but users should still grant the narrowest practical Billy access and enable only the tools they need.

Deleting a profile does not delete its Keychain item or policy file unless those deletion options are explicitly selected. This separation is intentional to prevent accidental credential or configuration loss.
