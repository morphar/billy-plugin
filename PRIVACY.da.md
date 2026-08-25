# Privatliv

**Dansk** | [English](PRIVACY.md)

Billy Plugin er fuldt lokal middleware. Det driver ingen hostet tjeneste og indsamler hverken analyse, telemetri, legitimationsoplysninger eller regnskabsdata til pluginets forfatter.

## Dataflow

- Den lokale proces `billy-mcp` læser tokenet til den valgte Billy-profil fra macOS Nøglering eller, under udvikling, fra miljøvariablen `BILLY_ACCESS_TOKEN`.
- Den sender forespørgsler direkte til Billy API'et over HTTPS.
- Den returnerer det ønskede API-svar til AI-klienten gennem den lokale MCP-forbindelse.
- Profilnavne og værktøjspolitikker gemmes i den aktuelle brugers Application Support-mappe. Tokens gemmes separat i macOS Nøglering.

Værktøjsresultater kan indeholde personoplysninger, finansielle oplysninger eller kundedata. Resultatet er synligt for den AI-klient, der håndterer samtalen, og er omfattet af klientens konfiguration og privatlivsvilkår. Billys behandling af API-forespørgsler er omfattet af Billys egne vilkår og privatlivspolitik.

Pluginet logger ikke bevidst legitimationsværdier. Præcise konfigurerede legitimationsværdier erstattes i svar og fejl fra API'et, før de returneres gennem MCP. Brugeren bør alligevel give den mindst mulige praktiske Billy-adgang og kun aktivere de nødvendige værktøjer.

Sletning af en profil sletter ikke dens element i Nøglering eller dens politikfil, medmindre disse sletninger vælges eksplicit. Adskillelsen er bevidst og forebygger utilsigtet tab af legitimationsoplysninger eller konfiguration.
