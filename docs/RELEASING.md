# Releasing Billy Plugin

[Dansk](RELEASING.da.md) | **English**

Billy Plugin is released personally from `morphar/billy-plugin`. Official beta releases are fully local packages: the archive contains the plugin, accounting skill, configuration, OpenAPI contract, and a prebuilt macOS Apple Silicon executable.

## One-time setup

1. Publish `cortexium-io/api-mcp`. `API_MCP_VERSION` must name its release tag and `API_MCP_COMMIT` must contain the full commit hash to which that tag resolves.
2. Enrol in the Apple Developer Program and obtain a **Developer ID Application** certificate. An Apple Development certificate is not sufficient for public distribution.
3. Create a GitHub Actions environment named `release`, restrict deployments to tags matching `v*`, and add these **environment secrets** there. Require a trusted reviewer when the repository visibility and GitHub plan support environment reviewers:

   - `APPLE_DEVELOPER_ID_P12`: base64-encoded export of the Developer ID Application certificate and private key
   - `APPLE_DEVELOPER_ID_PASSWORD`: password used when exporting the `.p12`
   - `APPLE_NOTARY_APPLE_ID`: Apple ID used for notarization
   - `APPLE_NOTARY_TEAM_ID`: Apple Developer Team ID
   - `APPLE_NOTARY_PASSWORD`: app-specific password for the Apple ID

   Do not retain copies as repository secrets. A workflow on an untrusted ref can request repository secrets without passing through the protected environment.

4. Protect the default branch, require the validation workflow, and add a tag ruleset restricting creation and deletion of tags matching `v*` to trusted maintainers. On GitHub plans that do not support required reviewers for a private repository, the protected tag ruleset and the environment's `v*` deployment restriction are mandatory; enable the reviewer gate when making the repository public.
5. Make the GitHub repository public only after reviewing the complete initial commit, especially the generated OpenAPI contract and bundled executable.

## Release checklist

1. Set `API_MCP_VERSION` to an existing api-mcp release tag and `API_MCP_COMMIT` to the full commit hash that the tag resolves to. The release workflow verifies the pair before checking out or executing api-mcp source.
2. Update the plugin manifest's base semantic version and user-facing documentation.
3. Run:

   ```bash
   ./scripts/validate
   API_MCP_SOURCE=/absolute/path/to/api-mcp VERSION=0.1.0-beta.2 ./plugins/billy-plugin/scripts/build-macos-arm64
   ./scripts/validate
   ```

4. Test installation from a clean directory with a non-production Billy account. Verify first-use Keychain setup, read-only defaults, multiple profiles, policy expansion, and a separately approved write followed by a read-back.
5. Review the complete diff and merge it to protected `main`. Confirm the release commit's validation run passed.
6. Confirm the beta and AI-generated-code warning remains prominent in the root README, packaged plugin README, manifest, and release notes.
7. Create and push an annotated tag matching the manifest's prerelease version:

   ```bash
   git tag -a v0.1.0-beta.1 -m "Billy Plugin v0.1.0-beta.1"
   git push origin v0.1.0-beta.1
   ```

8. Review and approve the `release` environment deployment when a reviewer gate is configured, then confirm the workflow succeeds.

The GitHub release workflow accepts only semantic-version tags whose commit is contained in the default branch. It verifies that the pinned api-mcp tag resolves to the exact `API_MCP_COMMIT` before executing that source, builds the executable with Go 1.27, imports the Developer ID certificate into an ephemeral Keychain, signs the executable with the hardened runtime, submits the release archive to Apple's notarization service, and publishes the archive and checksum. The protected environment and tag ruleset remain required because checks inside a workflow are not a substitute for GitHub-enforced trust boundaries.

## Verification

Download the published archive and checksum on another Mac:

```bash
shasum -a 256 -c SHA256SUMS
unzip billy-plugin_0.1.0-beta.1_darwin-arm64.zip
codesign --verify --deep --strict --verbose=2 billy-plugin/plugins/billy-plugin/bin/darwin-arm64/billy-mcp
```

Install that extracted marketplace, restart the AI client, and test in a new task. A ZIP archive cannot carry a stapled notarization ticket; Gatekeeper retrieves the submitted ticket from Apple when the signed executable is first assessed online.
