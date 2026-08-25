# Releasing Billy Plugin

Billy Plugin is released personally from `morphar/billy-plugin`. Official releases are fully local packages: the archive contains the plugin, accounting skill, configuration, OpenAPI contract, and a prebuilt macOS Apple Silicon executable.

## One-time setup

1. Publish `cortexium-io/api-mcp` and create the tag named in `API_MCP_VERSION`.
2. Enrol in the Apple Developer Program and obtain a **Developer ID Application** certificate. An Apple Development certificate is not sufficient for public distribution.
3. Add these GitHub Actions repository secrets:

   - `APPLE_DEVELOPER_ID_P12`: base64-encoded export of the Developer ID Application certificate and private key
   - `APPLE_DEVELOPER_ID_PASSWORD`: password used when exporting the `.p12`
   - `APPLE_NOTARY_APPLE_ID`: Apple ID used for notarization
   - `APPLE_NOTARY_TEAM_ID`: Apple Developer Team ID
   - `APPLE_NOTARY_PASSWORD`: app-specific password for the Apple ID

4. Protect the default branch and require the validation workflow.
5. Make the GitHub repository public only after reviewing the complete initial commit, especially the generated OpenAPI contract and bundled executable.

## Release checklist

1. Set `API_MCP_VERSION` to an existing api-mcp release tag.
2. Update the plugin manifest's base semantic version and user-facing documentation.
3. Run:

   ```bash
   ./scripts/validate
   API_MCP_SOURCE=/absolute/path/to/api-mcp VERSION=0.1.0 ./plugins/billy-plugin/scripts/build-macos-arm64
   ./scripts/validate
   ```

4. Test installation from a clean directory with a non-production Billy account. Verify first-use Keychain setup, read-only defaults, multiple profiles, policy expansion, and a separately approved write followed by a read-back.
5. Review the complete diff and merge it to `main`.
6. Create and push an annotated tag matching the manifest's base version:

   ```bash
   git tag -a v0.1.0 -m "Billy Plugin v0.1.0"
   git push origin v0.1.0
   ```

The GitHub release workflow checks out the pinned api-mcp tag, builds the executable with Go 1.27, imports the Developer ID certificate into an ephemeral Keychain, signs the executable with the hardened runtime, submits the release archive to Apple's notarization service, and publishes the archive and checksum.

## Verification

Download the published archive and checksum on another Mac:

```bash
shasum -a 256 -c SHA256SUMS
unzip billy-plugin_0.1.0_darwin-arm64.zip
codesign --verify --deep --strict --verbose=2 billy-plugin/plugins/billy-plugin/bin/darwin-arm64/billy-mcp
```

Install that extracted marketplace, restart the AI client, and test in a new task. A ZIP archive cannot carry a stapled notarization ticket; Gatekeeper retrieves the submitted ticket from Apple when the signed executable is first assessed online.
