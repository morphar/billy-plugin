# Sikkerhedspolitik

**Dansk** | [English](SECURITY.md)

## Understøttede versioner

Sikkerhedsrettelser foretages i den senest udgivne version. Den første understøttede platform er macOS på Apple Silicon.

## Rapportering af en sårbarhed

Rapportér mistænkte sårbarheder privat gennem repositoriets formular til GitHub Security Advisories. Opret ikke et offentligt issue om eksponering af tokens, adgang til Nøglering, omgåelse af godkendelse eller værktøjspolitik, brud på profiladskillelse, usikker opdateringsadfærd eller eksponering af private regnskabsdata.

Angiv den berørte pluginversion, macOS-version, trin til reproduktion og den forventede konsekvens. Fjern organisations-ID'er, regnskabsdata, tokens, eksporter fra Nøglering og andre private oplysninger før indsendelse. Du bør modtage en kvittering inden for syv dage.

## Sikkerhedsgrænser

- MCP-programmet og konfigurationen kører lokalt; der findes intet mellemled drevet af pluginprojektet.
- macOS Nøglering forhindrer tokenet i at blive en del af pluginets konfiguration eller et værktøjsargument, som modellen kan se. Nøglering beskytter ikke mod en kompromitteret lokal brugerkonto eller et kompromitteret program.
- Den medfølgende politik starter med skrivebeskyttet adgang og kontrollerer den valgte profil og OpenAPI-deklarerede input, før en legitimationsoplysning hentes, eller en forespørgsel sendes til Billy.
- Ændringer af værktøjspolitikker og profiler samt alle Billy-forespørgsler, der ikke kun læser, kræver en lokal macOS-godkendelse efter validering og før legitimationsoplysninger eller sideeffekter. Serveren håndhæver godkendelsen, og den kan ikke leveres som et MCP-argument af en model.
- Aktivering af et skriveværktøj gør kun den pågældende type forespørgsel teknisk tilgængelig. Hver konkret bogføringsændring kræver stadig sin egen godkendelse.
- Værktøjsresultater indeholder aktuelle Billy-data og er synlige for den valgte AI-klient.
- Officielle releasearkiver er signerede, indsendt til Apple til notarization og udgivet med SHA-256-kontrolsummer. Programmer fra kildekode-checkoutet er udviklingsversioner og må ikke præsenteres som officielle releases.
