VERDICT: CHANGES_REQUESTED

Die Prüfung der vollständig gemergten Go-Pastebin-REST-API ergibt keine hartkodierten Secrets, keine SQL-/Command-/Path-Injection, keine unsichere Deserialisierung und keine bekannten Dependency-Schwachstellen. Der Scanner hat für diesen Projekttyp keine Prüfung durchgeführt; das ist als Lücke zu dokumentieren, begründet aber allein keinen Befund. Die nachfolgenden Schwachstellen betreffen Härtung, Verfügbarkeit und Betriebskonfiguration.

## Sicherheitsprüfungsbericht

### 1. DoS durch unbegrenzte Paste-Anzahl (Speicherfüllung)
- **Schweregrad:** mittel
- **Betroffene Stelle:** `store.go` (`NewStore` / `Create`)
- **Beschreibung:** Der Store speichert Pasts unbegrenzt in einer In-Memory-Map. Ein Angreifer kann über `POST /pastes` beliebig viele gültige Paste-Anfragen mit bis zu 1 MiB Body senden, bis der Prozess den verfügbaren Arbeitsspeicher erschöpft (OOM-Kill oder Instabilität). Es existiert weder eine Grenze für die Gesamtzahl noch ein Ablauf der In-Memory-Daten bei Erreichen eines Limits.
- **Konkrete Lösung:** Ein konfigurierbares Limit für die maximale Anzahl aktiver Pasts einführen, z. B.:
  - `const maxPastes = 10000` (besser über Umgebungsvariable konfigurierbar).
  - In `Store.Create` vor dem Einfügen prüfen, ob `len(s.pastes) >= maxPastes`; bei Überschreitung einen definierten Fehler wie `errors.New("store full")` zurückgeben.
  - `CreateHandler` übersetzt diesen Fehler in eine JSON-Fehlerantwort, z. B. `503` mit `{"error":"storage full"}`.
  - Bestehende Tests sind dadurch nicht betroffen; ein neuer Test für das Limit ist sinnvoll.

### 2. Fehlende HTTP-Server-Timeouts (Slowloris / Ressourcenbindung)
- **Schweregrad:** mittel
- **Betroffene Stelle:** `main.go` (Aufruf `http.ListenAndServe(addr, router)`)
- **Beschreibung:** Der Server wird ohne explizite Timeouts gestartet. Dadurch können langsame Clients Verbindungen über lange Zeit offen halten (Slowloris) und Server-Ressourcen binden.
- **Konkrete Lösung:** Statt `http.ListenAndServe` einen `http.Server` mit Timeouts verwenden:
  ```go
  srv := &http.Server{
      Addr:              ":" + port,
      Handler:           router,
      ReadHeaderTimeout: 5 * time.Second,
      ReadTimeout:       10 * time.Second,
      WriteTimeout:      10 * time.Second,
      IdleTimeout:       120 * time.Second,
  }
  if err := srv.ListenAndServe(); err != nil {
      log.Fatal(err)
  }
  ```
  Die Tests mit `httptest.NewServer` sind nicht betroffen.

### 3. Fehlende Transportverschlüsselung
- **Schweregrad:** mittel
- **Betroffene Stelle:** `main.go` (Serverstart / Betrieb)
- **Beschreibung:** Der Server lauscht auf reinem HTTP. Paste-Inhalte und die zufälligen Paste-IDs können im Netzwerk passiv mitgelesen werden. Da die ID die einzige Zugriffskontrolle darstellt, ist ein Abfangen der ID gleichbedeutend mit dem vollständigen Zugriff auf den Paste.
- **Konkrete Lösung:** TLS aktivieren:
  - Entweder direkt `ListenAndServeTLS` mit konfigurierbaren Zertifikatspfaden verwenden,
  - oder im Deployment verbindlich einen TLS-Reverse-Proxy (z. B. nginx/caddy) vorsehen und dokumentieren.
  - Tests bleiben unberührt, da sie den Router direkt über `httptest.NewServer` testen.

### 4. Überlauf bei `expires_in_seconds` (int64-Multiplikation)
- **Schweregrad:** niedrig
- **Betroffene Stelle:** `paste_create.go` (`time.Duration(req.ExpiresInSeconds) * time.Second`)
- **Beschreibung:** Sehr große Werte von `expires_in_seconds` können bei der Multiplikation mit `time.Second` als `int64` überlaufen. Dadurch entsteht ein negativer `time.Duration`-Wert, den `Store.Create` anschließend als `<= 0` interpretiert und die Standardablaufzeit von 24 Stunden setzt. Das ist kein direkter Exploit, führt aber zu unerwartetem Verhalten und zeigt fehlende Eingabevalidierung.
- **Konkrete Lösung:** Eine Obergrenze festlegen und vor der Umwandlung prüfen, z. B.:
  ```go
  const maxExpirySeconds = 365 * 24 * 3600 // 1 Jahr
  if req.ExpiresInSeconds > maxExpirySeconds {
      writeError(w, http.StatusBadRequest, "expires_in_seconds exceeds maximum allowed value")
      return
  }
  ```
  Außerdem kann geprüft werden, ob `time.Duration(req.ExpiresInSeconds) > (math.MaxInt64 / time.Second)`, um den Überlauf zu verhindern. Bestehende Tests bleiben unverändert.

### 5. Fehlende Security-Header auf JSON-Antworten
- **Schweregrad:** niedrig
- **Betroffene Stelle:** `response.go` (`writeJSON`)
- **Beschreibung:** JSON-Antworten werden nur mit `Content-Type: application/json` ausgeliefert. Es fehlen Header wie `X-Content-Type-Options: nosniff`, um MIME-Sniffing durch Browser zu verhindern, sowie `Cache-Control: no-store`, um das Caching vertraulicher Paste-Inhalte zu unterbinden.
- **Konkrete Lösung:** In `writeJSON` vor dem `WriteHeader` ergänzen:
  ```go
  w.Header().Set("Content-Type", "application/json")
  w.Header().Set("X-Content-Type-Options", "nosniff")
  w.Header().Set("Cache-Control", "no-store")
  ```
  Die bestehenden Tests prüfen nur `Content-Type`, sind also kompatibel.

### 6. Öffentliche Metadaten-Liste leakt alle Paste-IDs
- **Schweregrad:** niedrig (Design-/Dokumentationsrisiko)
- **Betroffene Stelle:** `store.go` (`List`) / `paste_list.go`
- **Beschreibung:** `GET /pastes` liefert alle aktiven Paste-IDs. Da die ID die einzige Zugriffskontrolle ist, kann jeder, der die Liste abruft, anschließend jeden aktiven Paste über `GET /pastes/{id}` lesen und über `DELETE /pastes/{id}` löschen. Dies ist vermutlich als öffentlicher Pastebin beabsichtigt; für Nutzer, die die ID als geheim betrachten, stellt es jedoch eine Vertraulichkeits-/Integritätslücke dar.
- **Konkrete Lösung:** Da die bestehenden Tests (`paste_list_test.go`) ausdrücklich prüfen, dass IDs in der Metadatenliste enthalten sind, kann die Liste nicht ohne Bruch der Spec/Test geändert werden. Stattdessen:
  - In der API-Dokumentation (README) klarstellen, dass alle aktiven Pasts über `GET /pastes` öffentlich einsehbar sind.
  - Optional künftig eine Authentifizierung/Eigentümer-Zuordnung einführen, falls private Pasts unterstützt werden sollen.
  - Keine sofortige Codeänderung erforderlich, da das Produktverhalten durch AC-06 und bestehende Tests abgedeckt ist.

## Fazit
Es wurden keine kritischen oder hochriskanten Schwachstellen gefunden. Die vorhandenen Befunde rechtfertigen jedoch eine Überarbeitung (Härtung), insbesondere die Server-Timeouts, ein Speicherlimit gegen DoS und eine TLS-Absicherung im Produktivbetrieb. Daher lautet das Gesamturteil: **CHANGES_REQUESTED**.