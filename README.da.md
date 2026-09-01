# Billy Plugin

**Dansk** | [English](README.md)

> [!WARNING]
> **Beta / AI-genereret kode:** Dette er eksperimentel betasoftware, og en stor del af koden, dokumentationen og Billy OpenAPI-kontrakten er genereret med hjælp fra AI under menneskelig styring og gennemgang. Automatiske tests, kodesignering og notarization er ingen garanti for korrekt bogføring. Test med en Billy-konto, der ikke bruges i produktion, behold skriveværktøjer deaktiveret, medmindre de er nødvendige, og kontrollér selv alle regnskabsresultater og ændringer.

Billy Plugin samler en fuldt lokal Billy MCP-server og en skill til arbejdsgange inden for bogføring til ChatGPT-skrivebordsappen og Codex.

Pluginet bruger ikke et hostet mellemled. Det selvstændige Go-program kører på brugerens computer, læser Billy-legitimationsoplysninger fra operativsystemets lokale, sikre lager, kalder Billy direkte og fjerner den præcise legitimationsværdi fra svar og fejl, som modellen kan se.

Dette er et uafhængigt og uofficielt projekt. Det er ikke tilknyttet, godkendt eller sponsoreret af Billy.

## Nuværende betamål

Den første betaversion understøtter macOS på Apple Silicon. Brugeren behøver ikke at have Go eller Node.js installeret.

Programmet i kildekoderepositoriet er en ad hoc-signeret udviklingsversion. Officielle GitHub-releasearkiver bygges fra tagget api-mcp-kildekode, signeres med et Apple Developer ID, indsendes til Apples notarization-tjeneste og udgives med kontrolsummer.

## Installér en release

Hent arkivet til macOS på Apple Silicon og filen `SHA256SUMS` fra GitHub-releasen. Kontrollér arkivet, pak det ud, og tilføj derefter den udpakkede mappe som en lokal marketplace:

```bash
shasum -a 256 -c SHA256SUMS
codex plugin marketplace add /absolut/sti/til/billy-plugin
codex plugin add billy-plugin@billy
```

Genstart ChatGPT-skrivebordsappen eller Codex, og begynd en ny opgave, så den medfølgende skill og MCP-værktøjerne bliver indlæst. Go og Node.js er ikke nødvendige.

### Første adgang til Nøglering

Der kan blive vist to forskellige typer sikre dialoger. Ingen af dem gør Billy-tokenet synligt for AI-klienten:

1. Hvis der ikke er gemt et token, åbner Billy Plugin sin egen dialog med skjult indtastning og beder om Billy-tokenet to gange, før det gemmes i macOS Nøglering.
2. Når `billy-mcp` læser det gemte element første gang, efter en opdatering eller efter en ændring af programmets kodeidentitet, kan macOS vise en eller flere dialoger om adgang til Nøglering.

Ved en officiel release skal du kontrollere, at programmet er `billy-mcp`, at elementet i Nøglering er `com.morphar.billy-plugin`, og at nøgleringen er `login`. Vælg **Tillad altid** for at undgå et spørgsmål ved hver læsning, og indtast adgangskoden til Mac-computerens `login`-nøglering—normalt din adgangskode til Mac-computeren, **ikke** Billy-tokenet. Hvis du vælger **Afvis**, kan legitimationsoplysningen ikke læses, og der sendes ingen forespørgsel til Billy.

Efter installation eller opdatering skal du genstarte AI-klienten og begynde en ny opgave. En opgave, som allerede var åben, kan stadig være forbundet til den tidligere MCP-proces. Se den pakkede [pluginvejledning](plugins/billy-plugin/README.da.md#første-opsætning-af-legitimationsoplysninger) for fejlfinding.

## Installér fra dette checkout

Installér pluginet, genstart ChatGPT-skrivebordsappen, og begynd en ny opgave, så den medfølgende skill og MCP-værktøjerne bliver indlæst. Det første kald til Billy API'et åbner en lokal macOS-dialog med skjult indtastning, hvis legitimationsoplysningen ikke allerede findes i Nøglering.

Manuel opsætning og statuskontrol er fortsat mulig for profilen `default`:

```bash
./plugins/billy-plugin/scripts/setup-macos
./plugins/billy-plugin/scripts/credential-status-macos
```

På macOS kan du alternativt dobbeltklikke på `plugins/billy-plugin/Setup Billy.command` i Finder.

Tilføj repositoriet som en lokal marketplace, og installér pluginet:

```bash
codex plugin marketplace add /absolut/sti/til/billy-plugin
codex plugin add billy-plugin@billy
```

## Sikker værktøjskonfiguration

Ved første MCP-start oprettes `~/Library/Application Support/Billy Plugin/config.json` automatisk med kun funktionen `read-only` aktiveret. Ingen endpoints til oprettelse, opdatering eller sletning er tilgængelige som standard.

ChatGPT, Codex eller Claude kan forklare og administrere denne politik gennem `get_tool_configuration`, `list_tool_features`, `search_available_tools` og det eneste konfigurationsændrende værktøj, `configure_tools`. Konfigurationsændringer valideres mod den medfølgende OpenAPI-kontrakt, hvorefter en lokal macOS-dialog viser den valgte konto, den ønskede ændring og den resulterende politik. MCP'en gemmer og anvender kun ændringen efter lokal godkendelse; en model kan ikke godkende den med et argument.

Flere Billy-konti administreres af én lokal MCP-proces. `list_api_profiles`, `add_api_profile`, `rename_api_profile` og `remove_api_profile` administrerer et lokalt register uden tokens. Profilændringer kræver den samme lokale godkendelse. Hver profil har sit eget element i Nøglering og sin egen værktøjspolitik, der som standard kun tillader læsning. API-kald kræver en eksplicit profil, når der findes mere end én konto. Eksisterende legitimationsoplysninger og indstillinger for `default` migreres på stedet.

Alle forespørgsler til Billy, der opretter, opdaterer eller sletter, valideres og kontrolleres mod den valgte profils politik. Derefter viser en lokal macOS-dialog den præcise ændring, før tokenet læses, og før Billy kontaktes. Skrivebeskyttede Billy-kald viser ikke denne dialog. Opsætning af legitimationsoplysninger bruger fortsat en særskilt dialog med skjult indtastning.

Pluginmetadata er på engelsk, fordi det nuværende manifest kun har ét beskrivelsesfelt og ingen sprogspecifikke varianter. De medfølgende startprompts og dokumentationen indeholder danske eksempler, så danske forespørgsler fungerer naturligt i ChatGPT, Codex og Claude.

## Repositoriets opbygning

- `.agents/plugins/marketplace.json` gør repositoriet tilgængeligt som marketplacen `billy`.
- `plugins/billy-plugin/.codex-plugin/plugin.json` definerer det plugin, der kan installeres.
- `plugins/billy-plugin/.mcp.json` starter den lokale MCP over stdio.
- `plugins/billy-plugin/skills/accounting` indeholder arbejdsgangen og sikkerhedspolitikken for bogføring.
- `cmd/billy-mcp` og `internal/billyworkflow` indeholder de faste review- og beskyttede cleanup-værktøjer oven på api-mcp.
- `plugins/billy-plugin/config/billy.json` definerer serverens uforanderlige lokale indstillinger. `config/features.json` forklarer de understøttede funktionsgrupper, mens den aktive brugerpolitik ligger uden for plugin-cachen.
- `plugins/billy-plugin/evals` indeholder sessionsbaserede regnskabsregressioner, som fortsat skal være dækket af skillen eller eksekverbare tests.
- `plugins/billy-plugin/bin` indeholder releaseprogrammer, aldrig legitimationsoplysninger.

## Build for vedligeholdere

Den fælles MCP-motor ligger i øjeblikket i et parallelt `api-mcp`-checkout. Dets rene `HEAD` skal svare til det fulde hash i `API_MCP_COMMIT`. Buildet kompilerer den Billy-specifikke kommando i dette repository mod præcis den fastlåste motor-commit:

```bash
./plugins/billy-plugin/scripts/build-macos-arm64
```

Sæt `API_MCP_SOURCE=/absolut/sti/til/api-mcp`, hvis checkoutet ligger et andet sted. Buildet indlejrer hverken Billy-tokenet eller konfigurationen.

Krav til releases og opsætning af signeringshemmeligheder er dokumenteret i [docs/RELEASING.da.md](docs/RELEASING.da.md). Sikkerhed og grænserne for lokale data er dokumenteret i [SECURITY.da.md](SECURITY.da.md) og [PRIVACY.da.md](PRIVACY.da.md).

## Licens

MIT. Se [LICENSE](LICENSE).
