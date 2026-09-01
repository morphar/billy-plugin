# Bidrag

**Dansk** | [English](CONTRIBUTING.md)

Tak, fordi du hjælper med at forbedre Billy Plugin.

Opret et GitHub-issue før større ændringer af adfærd, rettighedsmodel, pakning eller offentlige kontrakter. Rapportér sikkerhedsproblemer privat som beskrevet i [SECURITY.da.md](SECURITY.da.md).

Bidrag skal bevare disse garantier:

- ingen Billy-legitimationsoplysninger i pluginkonfiguration, logfiler, MCP-input eller MCP-resultater
- kun skrivebeskyttet værktøjsadgang ved første start og for hver ny profil
- eksplicit valg af profil, når der findes mere end én konto
- håndhævelse af værktøjspolitikken på serversiden før læsning af legitimationsoplysninger eller HTTP-kald
- eksplicit brugergodkendelse før udvidelse af værktøjsadgang eller ændring af konkret bogføring

Kør `./scripts/validate` fra repositoriets rod, før du åbner en pull request. I et almindeligt checkout bruger valideringen det udgivne `api-mcp`-modul. Ved koordinerede, endnu ikke udgivne motorændringer skal du sætte `API_MCP_SOURCE=/absolut/sti/til/api-mcp`; valideringen kræver da præcis det rene commit, der står i `API_MCP_COMMIT`, uden et permanent lokalt `replace`-direktiv. Ændringer af MCP-motoren hører til i `cortexium-io/api-mcp`; opdatér først `API_MCP_VERSION`, efter at den pågældende motorversion er udgivet.

Ved at indsende et bidrag accepterer du, at det licenseres under repositoriets MIT License, og bekræfter, at du har ret til at stille bidraget til rådighed under disse vilkår.
