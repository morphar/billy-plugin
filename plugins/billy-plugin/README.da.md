# Billy Plugin

**Dansk** | [English](README.md)

> [!WARNING]
> **Beta / AI-genereret kode:** Dette er eksperimentel betasoftware, og en stor del af koden, dokumentationen og Billy OpenAPI-kontrakten er genereret med hjælp fra AI under menneskelig styring og gennemgang. Automatiske tests, kodesignering og notarization er ingen garanti for korrekt bogføring. Test med en Billy-konto, der ikke bruges i produktion, behold skriveværktøjer deaktiveret, medmindre de er nødvendige, og kontrollér selv alle regnskabsresultater og ændringer.

Billy Plugin er et lokalt regnskabsplugin til ChatGPT-skrivebordsappen, Codex og andre MCP-klienter som Claude. Det indeholder en Billy MCP-server, en sikker værktøjskonfiguration og en skill til arbejdsgange inden for bogføring. MCP-programmet kører lokalt og kalder Billy direkte; der findes intet hostet mellemled.

Dette er et uafhængigt og uofficielt projekt. Det er ikke tilknyttet, godkendt eller sponsoreret af Billy.

## Understøttet platform

Den første pakke understøtter macOS på Apple Silicon. Andre platforme kræver deres eget forudbyggede program og en adapter til sikker opbevaring af legitimationsoplysninger, før de kan betegnes som understøttede.

Det program, der ligger i kildekoderepositoriet, er en ad hoc-signeret udviklingsversion. En offentlig macOS-release skal bygges igen fra committed kildekode, signeres med et Apple Developer ID, notariseres og ledsages af en opdateret `SHA256SUMS`-fil.

## Første opsætning af legitimationsoplysninger

MCP'en kan starte uden et gemt Billy-token. Ved det første Billy API-kald for en valgt profil åbner den en lokal macOS-dialog med skjult indtastning, beder om profilens token to gange, gemmer det direkte i Nøglering og fortsætter derefter den ønskede handling. ChatGPT modtager kun status for opsætningen; tokenet er aldrig et MCP-værktøjsargument eller -resultat.

Det medfølgende værktøj `api_mcp_setup_credential` kan åbne den samme dialog direkte, mens `api_mcp_credential_status` kun oplyser, om der findes et element i Nøglering.

Manuel opsætning er fortsat mulig for profilen `default` fra pluginmappen:

```bash
./scripts/setup-macos
./scripts/credential-status-macos
```

På macOS kan du i stedet dobbeltklikke på `Setup Billy.command` i Finder og følge instruktionerne, hvor indtastningen ikke vises.

Begge opsætningsmetoder læser legitimationsoplysningen to gange uden at vise den og gemmer den som et ikke-synkroniseret, enhedsspecifikt element i macOS Nøglering. Legitimationsoplysningen accepteres aldrig som et kommandolinjeargument eller MCP-værktøjsinput.

`BILLY_ACCESS_TOKEN` kan fortsat bruges som en miljøvariabel, der tilsidesætter Nøglering under udvikling. Hvis variablen ikke er sat, bruger MCP'en Nøglering.

### Godkendelse af adgang til macOS Nøglering

Dialogen til indtastning af tokenet ovenfor styres af Billy Plugin. macOS kan desuden vise sin egen godkendelsesdialog, når programmet `billy-mcp` forsøger at læse et eksisterende element i Nøglering. Det sker typisk ved første adgang og kan ske igen efter installation af en opdatering eller en anden ændring af programmets kodeidentitet. macOS kan vise mere end én godkendelsesdialog, mens adgangen tildeles.

Kontrollér dialogen, før du godkender en officiel Billy Plugin-release:

- programmet, der anmoder om adgang, er `billy-mcp`
- elementet i Nøglering er `com.morphar.billy-plugin`
- elementet ligger i nøgleringen `login`

Vælg **Tillad altid**, hvis oplysningerne stemmer, og det signerede plugin fremover skal kunne læse tokenet uden at spørge hver gang. Indtast adgangskoden til nøgleringen `login`—normalt adgangskoden til Mac-computeren, **ikke** Billy-tokenet. **Tillad** gælder kun det aktuelle forsøg og kan medføre et nyt spørgsmål senere. **Afvis** forhindrer læsning af legitimationsoplysningen; der sendes ingen forespørgsel til Billy API'et.

Hvis adgangen blev afvist, kan klienten vise macOS Nøglering-fejlen `-25293`. Prøv handlingen igen, og godkend den dialog, hvor oplysningerne stemmer. Hvis dialogerne eller fejlen fortsætter efter **Tillad altid**, skal du afslutte og åbne AI-klienten igen og begynde en ny opgave. En opgave, der blev åbnet før installationen eller opdateringen, kan stadig bruge den tidligere MCP-proces. En senere signeret release kan med rette spørge igen, fordi macOS vurderer det opdaterede program særskilt.

## Værktøjskonfiguration

Ved første start opretter MCP'en automatisk denne lokale brugerkonfiguration:

`~/Library/Application Support/Billy Plugin/config.json`

```json
{
  "version": 1,
  "enabledFeatures": ["read-only"],
  "enabledEndpoints": [],
  "disabledEndpoints": []
}
```

Standarden er afvisning som udgangspunkt: kun Billy `GET`-endpoints er tilgængelige. Ingen endpoints til oprettelse, opdatering eller sletning er aktiveret. Filen indeholder kun værktøjsrettigheder—aldrig Billy-tokenet—og oprettes med adgang kun for ejeren.

MCP'en tilbyder fire konfigurationsværktøjer:

- `get_tool_configuration` viser den valgte profils konfiguration og filsti.
- `list_tool_features` forklarer alle navngivne funktioner, risikoniveauer og endpointgrupper for den valgte profil.
- `search_available_tools` søger efter og forklarer enkelte endpoints og oplyser, om de er aktiveret for den valgte profil.
- `configure_tools` tilføjer eller fjerner navngivne funktioner eller enkelte endpointselektorer for den valgte profil, gemmer den lokale brugerfil atomisk og opdaterer MCP-værktøjslisten med det samme.

`configure_tools` er det eneste værktøj, der ændrer konfigurationen. Den medfølgende accounting-skill kræver fortsat, at assistenten forklarer den præcise adgangsændring. Uafhængigt heraf åbner MCP'en en lokal macOS-dialog, som viser den valgte profil, den ønskede ændring og den resulterende politik, før noget gemmes. Klienten eller modellen kan ikke omgå dialogen med et bekræftelsesargument. Ændring af værktøjsadgang godkender ikke en bogføringsændring; konkrete ændringer kræver deres egen lokale godkendelse og efterfølgende kontrol.

Eksempler på forespørgsler:

- “Forklar hvilke Billy-funktioner der er tilgængelige.”
- “Hvilke værktøjer skal du bruge for at afstemme en banklinje?”
- “Aktivér værktøjer til bankafstemning for `default`.”
- “Deaktivér sletning af banklinjematches.”
- “Vis min nuværende Billy-værktøjskonfiguration.”

### Medfølgende funktionsgrupper

| Funktion | Risiko | Giver adgang til |
| --- | --- | --- |
| `read-only` | Læsning | Alle Billy `GET`-endpoints |
| `bank-payments` | Skrivning | Oprettelse og opdatering af bankbetalinger |
| `purchase-bookkeeping` | Skrivning | Oprettelse og opdatering af regninger, regningslinjer og tilknyttede bilag |
| `bank-reconciliation` | Skrivning | Oprettelse, opdatering og sletning af banklinjematches og deres emnetilknytninger |
| `duplicate-bank-line-cleanup` | Skrivning | Sletning af præcise banklinjer gennem den beskyttede preview/commit-arbejdsgang |
| `daybook-corrections` | Skrivning | Oprettelse og opdatering af kassekladdeposteringer og -linjer |
| `sales-tax-settlement` | Skrivning | Opdatering af momsafregning og momsbetalinger i Billy; indsender aldrig til Skattestyrelsen |

En individuel selektor kan være et MCP-værktøjsnavn som `delete_bank_line_match`, et OpenAPI-operation-ID, en metode og sti som `DELETE /bankLineMatches/{bankLineMatchId}`, en tagselektor som `tag:BankLineMatches` eller et wildcard. Et element i `disabledEndpoints` har forrang for aktiverede funktioner og endpoints.

Du kan redigere den lokale JSON-fil manuelt, men ændringer gennem MCP'en er sikrere, fordi de valideres mod den medfølgende Billy OpenAPI-kontrakt og træder i kraft med det samme. Genstart MCP-klienten efter manuel redigering. Redigér ikke konfigurationen i den installerede plugin-cache; disse filer bliver erstattet ved opgraderinger.

## Arbejdsgange til regnskabskontrol

Pluginet tilføjer faste værktøjer til de kontroller, der ellers kræver mange rå API-kald:

- `api_mcp_diagnostics` identificerer den kørende version, OpenAPI-kontrakten, profiler, tilgængelighed af legitimationsoplysninger, svargrænsen og de aktive funktionspolitikker.
- `diagnose_billy_connection` kontrollerer en valgt profil med en ufarlig læsning af Billy-organisationen.
- `review_billy_bank_account`, `review_billy_document_coverage`, `review_billy_vat_period` og `review_billy_foreign_currency_purchase` returnerer struktureret dokumentation, paginationens fuldstændighed, dataenes sluttidspunkt og eksplicitte API-begrænsninger.
- `preview_billy_bank_line_cleanup` genlæser præcise kandidater og returnerer et deterministisk digest uden at skrive.
- `commit_billy_bank_line_cleanup` afviser ændrede, godkendte eller nyligt tilknyttede mål, beder om én lokal godkendelse til hele den præcise batch, kontrollerer den beskyttede tilstand igen umiddelbart efter godkendelsen og verificerer hver idempotent sletning. Billys dokumenterede slette-endpoint har ingen betinget skriveforudsætning, så kontrollen indsnævrer, men kan ikke fjerne, risikoen for en samtidig ekstern ændring mellem sidste kontrol og sletning.

Et tomt API-resultat behandles ikke som fravær, medmindre alle relevante sider er læst. En banklinjestatus som `booked` behandles ikke som bevis på afstemning uden et godkendt match og dets emnetilknytninger. Billys dokumenterede API viser ikke alle tilstande fra webappens bilagsindbakke, så arbejdsgangen oplyser begrænsningen i stedet for at påstå, at en fil, som kan ses i browseren, ikke findes.

## Flere Billy-konti

Pluginet understøtter flere lokale Billy-kontoprofiler. Hver profil har:

- et stabilt ID med små bogstaver og et læsevenligt navn
- et særskilt token gemt under sin egen konto i macOS Nøglering
- en særskilt værktøjspolitik, der som standard kun tillader læsning
- håndhævelse af rettigheder på serversiden før ethvert Billy HTTP-kald

Den første profil oprettes automatisk som `default`. Den genbruger det eksisterende Nøglering-element `com.morphar.billy-plugin` / `default` og den eksisterende værktøjspolitik i `config.json`. En opgradering kopierer, viser eller nulstiller derfor ikke det nuværende token eller de nuværende indstillinger.

Profilmetadata gemmes i:

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

Registeret indeholder aldrig tokens. Tokens til nye profiler bruger kontonavne i Nøglering som `profile:company-b`, og deres politikker ligger i `~/Library/Application Support/Billy Plugin/profiles/company-b/tools.json`.

Kontoværktøjer:

- `list_api_profiles` viser profiler, tilgængelighed af legitimationsoplysninger, stier til politikker og antal aktiverede endpoints uden at læse tokenværdier ind i chatten.
- `add_api_profile` opretter en profil og åbner straks den sikre dialog til tokenet. Et nyt ID starter med skrivebeskyttet adgang. Genbrug af et ID med en bevaret politik afvises, medmindre `restoreToolConfiguration: true` anmodes eksplicit og godkendes i den lokale dialog.
- `rename_api_profile` ændrer kun det læsevenlige navn.
- `remove_api_profile` fjerner en profil. Separate tilvalg styrer permanent sletning af dens element i Nøglering og dens politikfil. Der skal altid være mindst én profil.
- `api_mcp_credential_status` og `api_mcp_setup_credential` arbejder på den valgte profil.

Eksempler på forespørgsler:

- “Tilføj en Billy-konto med navnet Firma B og ID'et `firma-b`.”
- “Vis mine Billy-konti og deres aktiverede funktioner.”
- “Aktivér kun bankafstemning for `firma-b`.”
- “Omdøb `default` til Hovedfirma.”
- “Fjern `firma-b`, men behold token og indstillinger i Nøglering.”

Der findes bevidst ingen foranderlig global aktiv konto. Når der findes mere end én profil, kræver Billy API-værktøjerne profilens ID. Værktøjsresultater angiver også den anvendte profil. Dermed kan én chat eller bogføringshandling ikke ubemærket skifte konto for en anden handling.

MCP-værktøjslisten deles mellem profilerne. Hvis et skriveværktøj er aktiveret for én profil, er det synligt globalt, men serveren afviser det for alle profiler, hvis egen politik ikke tillader det—før en legitimationsoplysning læses, eller en HTTP-forespørgsel sendes.

Alle Billy-forespørgsler, der opretter, opdaterer eller sletter, valideres mod den medfølgende OpenAPI-operation og kontrolleres mod den valgte profils aktuelle politik. Derefter vises de i en lokal macOS-dialog med de præcise argumenter og Billy-destinationen. Billy-legitimationsoplysninger læses først efter godkendelsen. En beskyttet workflow-batch viser alle præcise elementer i én godkendelse og kan ikke udføre et kald uden for det godkendte sæt. Skrivebeskyttede Billy-kald er ikke interaktive bortset fra den første opsætning eller godkendelse i Nøglering.

Ændringer gennem profil- og konfigurationsværktøjerne opdaterer den aktuelle MCP-proces med det samme. Genstart en anden MCP-klient, der allerede kører, efter ændringer af profiler eller politikker fra ChatGPT, Codex, Claude eller ved manuel filredigering, så klienten genindlæser registeret.

## Privatliv og sikkerhed

Pluginet har ingen hostet tjeneste, telemetri eller analyse. Forespørgsler, der er nødvendige for at løse brugerens regnskabsopgave, sendes direkte fra den lokale MCP-proces til Billy API'et. Værktøjsresultater, som modellen kan se, kan indeholde regnskabsdata, så brugeren skal også tage højde for den valgte AI-klients privatlivsindstillinger og vilkår.

Billy-tokens gemmes i macOS Nøglering og vises ikke som MCP-input eller -resultater. De fulde grænser og rapporteringsprocessen findes i repositoriets [PRIVACY.da.md](../../PRIVACY.da.md) og [SECURITY.da.md](../../SECURITY.da.md).

## Licens

MIT. Licensen og tredjepartsmeddelelserne følger med pluginet.
