# Udgivelse af Billy Plugin

**Dansk** | [English](RELEASING.md)

Billy Plugin udgives personligt fra `morphar/billy-plugin`. Officielle betaversioner er fuldt lokale pakker: Arkivet indeholder pluginet, accounting-skillen, konfigurationen, OpenAPI-kontrakten og et forudbygget program til macOS på Apple Silicon.

## Engangsopsætning

1. Udgiv `cortexium-io/api-mcp`. `API_MCP_VERSION` skal angive dets releasetag, og `API_MCP_COMMIT` skal indeholde det fulde commit-hash, som tagget peger på.
2. Tilmeld dig Apple Developer Program, og anskaf et **Developer ID Application**-certifikat. Et Apple Development-certifikat er ikke tilstrækkeligt til offentlig distribution.
3. Opret et GitHub Actions-miljø med navnet `release`, begræns deployments til tags, der matcher `v*`, og tilføj følgende **environment secrets**. Kræv en betroet reviewer, når repositoriets synlighed og GitHub-abonnement understøtter reviewers for miljøer:

   - `APPLE_DEVELOPER_ID_P12`: base64-kodet eksport af Developer ID Application-certifikatet og den private nøgle
   - `APPLE_DEVELOPER_ID_PASSWORD`: adgangskoden, der blev brugt ved eksport af `.p12`-filen
   - `APPLE_NOTARY_APPLE_ID`: det Apple ID, der bruges til notarization
   - `APPLE_NOTARY_TEAM_ID`: Apple Developer Team ID
   - `APPLE_NOTARY_PASSWORD`: en app-specifik adgangskode til Apple ID'et

   Behold ikke kopier som repository secrets. Et workflow på en ref, der ikke er betroet, kan anmode om repository secrets uden at passere gennem det beskyttede miljø.

4. Beskyt standardbranchen, kræv valideringsworkflowet, og tilføj et tag-ruleset, som begrænser oprettelse og sletning af tags, der matcher `v*`, til betroede vedligeholdere. På GitHub-abonnementer, som ikke understøtter påkrævede reviewers for et privat repository, er det beskyttede tag-ruleset og miljøets `v*`-begrænsning obligatoriske. Aktivér reviewer-godkendelse, når repositoriet gøres offentligt.
5. Gør først GitHub-repositoriet offentligt efter gennemgang af hele det første commit, især den genererede OpenAPI-kontrakt og det medfølgende program.

## Tjekliste til en release

1. Sæt `API_MCP_VERSION` til et eksisterende api-mcp-releasetag og `API_MCP_COMMIT` til det fulde commit-hash, som tagget peger på. Kør `GOTOOLCHAIN=go1.27.0 go mod tidy`, efter at tagget er udgivet, så `go.sum` registrerer det udgivne modul, og kontrollér, at `go.mod` ikke indeholder et lokalt `replace`-direktiv. Releaseworkflowet kontrollerer versions-/commit-parret, før api-mcp-kildekode hentes eller køres.
2. Opdatér pluginmanifestets grundversion efter SemVer og den brugervendte dokumentation.
3. Kør:

   ```bash
   ./scripts/validate
   API_MCP_SOURCE=/absolut/sti/til/api-mcp VERSION=0.1.0-beta.3 ./plugins/billy-plugin/scripts/build-macos-arm64
   ./scripts/validate
   ```

4. Test installation fra en ren mappe med en Billy-konto, der ikke bruges i produktion. Kontrollér første opsætning og godkendelse i Nøglering, skrivebeskyttet standardadgang, flere profiler, udvidelse af værktøjspolitikken og en separat godkendt skrivehandling efterfulgt af kontrol ved læsning.
5. Gennemgå hele diffen, og merge den til den beskyttede `main`-branch. Kontrollér, at valideringen for releasecommittet lykkedes.
6. Kontrollér, at advarslen om beta og AI-genereret kode stadig står tydeligt i README-filen i roden, den pakkede plugin-README, manifestet og releasenoterne.
7. Opret og push et annoteret tag, som matcher manifestets prerelease-version:

   ```bash
   git tag -a v0.1.0-beta.3 -m "Billy Plugin v0.1.0-beta.3"
   git push origin v0.1.0-beta.3
   ```

8. Gennemgå og godkend deployment til miljøet `release`, hvis reviewer-godkendelse er konfigureret, og kontrollér derefter, at workflowet lykkes.

GitHub-releaseworkflowet accepterer kun SemVer-tags, hvis commit'et findes i standardbranchen. Det kontrollerer, at det fastlåste api-mcp-tag peger på præcis `API_MCP_COMMIT`, før kildekoden køres, bygger programmet med Go 1.27, importerer Developer ID-certifikatet i en midlertidig Nøglering, signerer programmet med hardened runtime, indsender releasearkivet til Apples notarization-tjeneste og udgiver arkivet og kontrolsummen. Det beskyttede miljø og tag-rulesettet er fortsat nødvendige, fordi kontroller inde i et workflow ikke kan erstatte tillidsgrænser, som håndhæves af GitHub.

## Kontrol

Hent det udgivne arkiv og kontrolsummen på en anden Mac:

```bash
shasum -a 256 -c SHA256SUMS
unzip billy-plugin_0.1.0-beta.3_darwin-arm64.zip
codesign --verify --deep --strict --verbose=2 billy-plugin/plugins/billy-plugin/bin/darwin-arm64/billy-mcp
```

Installér den udpakkede marketplace, genstart AI-klienten, og test i en ny opgave. Kontrollér både dialogen til indtastning af et manglende Billy-token og macOS-dialogen, der giver `billy-mcp` adgang til elementet `com.morphar.billy-plugin` i nøgleringen `login`. Et ZIP-arkiv kan ikke indeholde en fastgjort notarization-ticket; Gatekeeper henter den indsendte ticket fra Apple, første gang det signerede program kontrolleres online.
