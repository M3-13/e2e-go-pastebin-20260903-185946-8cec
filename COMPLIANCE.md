VERDICT: CHANGES_REQUESTED

## Prüfumfang und positives Ergebnis

Geprüft wurde der vollständige Stand des Go-Backends (reines Backend ohne Endnutzer-UI). Für ein solches Produkt sind Pflichttexte/Impressum/Cookie-Banner und Barrierefreiheit nicht einschlägig; die EU-KI-Verordnung ist mangels KI-Feature nicht einschlägig.

Positiv hervorzuheben:

- **Paste-IDs** werden mit `crypto/rand` erzeugt, 16 Zufallsbytes → 32 Hex-Zeichen → 128 Bit Entropie (`store.go`).
- **Request-Body-Limit** von 1 MiB über `http.MaxBytesReader` (`paste_create.go`).
- **Fehlerantworten** enthalten nur `{"error": ...}`, keine Stacktraces, Pfade oder Adressen (`response.go`).
- **Kein Logging** von Paste-Inhalten oder Client-IP-Adressen (`main.go` enthält nur `log.Fatal` bei Startfehlern; kein Access-Logging).
- **Standardablauf** bei `expires_in_seconds <= 0` beträgt 24 Stunden (`store.go`, `defaultExpiry`).
- **Abgelaufene Einträge** werden bei Zugriff über `Get`, `List` oder `Delete` entfernt und als 404 behandelt.
- **GET /pastes** liefert nur Metadaten ohne `content` (`paste_list.go`, `PasteMeta`).

Dennoch bestehen behebbare Lücken, insbesondere bei der DSGVO-Speicherbegrenzung, der Transportverschlüsselung und der CRA-konformen Härtung.

---

## 1. DSGVO

### Befund DSGVO-1 — Explizite Ablaufzeit unbegrenzt und potenzieller Integer-Overflow
**Schweregrad:** hoch

**Datei:** `paste_create.go`

**Problem:**  
`req.ExpiresInSeconds` kann beliebig große positive Werte annehmen. Die Umwandlung  
`time.Duration(req.ExpiresInSeconds) * time.Second` kann bei großen Werten überlaufen; außerdem gibt es keine Obergrenze. Ein Nutzer kann damit Pasts mit faktisch unbegrenzter Speicherdauer anlegen. Das verstößt gegen den Grundsatz der Speicherbegrenzung (Art. 5 Abs. 1 lit. e DSGVO) und gegen Privacy by Design / Datenminimierung (Art. 25 DSGVO). Der AC-13-Standardwert von maximal 24 Stunden greift nur, wenn `expires_in_seconds <= 0` ist.

**Konkrete Abhilfe:**  
In `paste_create.go` nach der Negativprüfung eine Obergrenze ergänzen, z. B.:

```go
const maxExpirySeconds = 30 * 24 * 60 * 60 // 30 Tage

if req.ExpiresInSeconds > maxExpirySeconds {
    writeError(w, http.StatusBadRequest, "expires_in_seconds exceeds maximum allowed lifetime")
    return
}
```

Dadurch wird zugleich der Overflow praktisch ausgeschlossen, weil der Multiplikand klein genug ist. Die Fehlerantwort bleibt als JSON-Fehler mit `400` kompatibel mit AC-02/AC-11.

---

### Befund DSGVO-2 — Abgelaufene Pasts werden nur bei Zugriff entfernt, nicht aktiv
**Schweregrad:** mittel

**Datei:** `store.go`, `main.go`

**Problem:**  
Abgelaufene Einträge werden erst gelöscht, wenn ihre ID über `GET /pastes/{id}` oder `DELETE /pastes/{id}` abgefragt wird bzw. `GET /pastes` sie wegräumt. Wenn niemand zugreift, bleiben die personenbezogenen Inhalte weiterhin im RAM gespeichert, obwohl die Speicherfrist abgelaufen ist. Das schwächt die tatsächliche Löschung nach Ablauf und entspricht nicht vollständig dem Grundsatz der Speicherbegrenzung.

**Konkrete Abhilfe:**  
In `store.go` eine Janitor-Methode ergänzen und in `main.go` starten:

```go
// store.go
func (s *Store) StartJanitor(interval time.Duration) {
    ticker := time.NewTicker(interval)
    go func() {
        for range ticker.C {
            s.List() // räumt abgelaufene Einträge ab
        }
    }()
}
```

In `main.go` nach `store := NewStore()`:

```go
store.StartJanitor(time.Hour)
```

Alternativ eine dedizierte interne Cleanup-Funktion verwenden, die abgelaufene Pasts ohne Nebenwirkungen entfernt. Wichtig: Der Janitor darf **keine** Paste-Inhalte oder IP-Adressen protokollieren.

---

### Befund DSGVO-3 / CRA — Transportverschlüsselung ist nicht erzwungen und nicht dokumentiert
**Schweregrad:** mittel

**Datei:** `main.go`, ggf. `README.md`

**Problem:**  
Der Server startet ausschließlich mit `http.ListenAndServe` und damit ohne TLS. Da Paste-Inhalte personenbezogene Daten enthalten können, ist die Übertragung ohne Transportverschlüsselung ein Risiko für Vertraulichkeit und Integrität (Art. 32 DSGVO; Sicherheit by design/default nach CRA). Zwar kann TLS auf einem vorgelagerten Reverse Proxy terminiert werden, aber das Produkt erzwingt dies nicht und dokumentiert es nicht sichtbar.

**Konkrete Abhilfe:**  
Entweder TLS direkt im Produkt unterstützen, z. B. in `main.go`:

```go
certFile := os.Getenv("TLS_CERT_FILE")
keyFile := os.Getenv("TLS_KEY_FILE")

server := &http.Server{
    Addr:              addr,
    Handler:           router,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      15 * time.Second,
    IdleTimeout:       60 * time.Second,
}

if certFile != "" && keyFile != "" {
    err = server.ListenAndServeTLS(certFile, keyFile)
} else {
    log.Println("WARNUNG: Kein TLS konfiguriert – nur hinter TLS-terminierendem Proxy betreiben")
    err = server.ListenAndServe()
}
```

Zusätzlich im `README.md` klarstellen: **„Der Betrieb ohne TLS-Terminierung ist für Produktion nicht zulässig.“** So bleibt die lokale Entwicklung ohne Zertifikat möglich, ohne dass das Produkt im Produktivbetrieb unverschlüsselt läuft.

---

### Befund DSGVO-4 — Fehlende `Cache-Control`-Direktive für sensible Antworten
**Schweregrad:** niedrig

**Datei:** `response.go`

**Problem:**  
Antworten mit Paste-Inhalten (`GET /pastes/{id}`, `POST /pastes`) werden nicht mit `Cache-Control: no-store` versehen. Zwischenspeicher (Browser, Proxys) könnten personenbezogene Inhalte ungewollt speichern.

**Konkrete Abhilfe:**  
In `response.go` in `writeJSON` ergänzen:

```go
w.Header().Set("Cache-Control", "no-store")
w.Header().Set("X-Content-Type-Options", "nosniff")
```

Diese Header sind mit der API-Funktion vollständig vereinbar und brechen keine Tests.

---

## 2. EU Cyber Resilience Act (CRA)

### Befund CRA-1 — HTTP-Server ohne Timeouts
**Schweregrad:** mittel

**Datei:** `main.go`

**Problem:**  
`http.ListenAndServe` nutzt die Default-Werte des `http.Server` ohne `ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout` oder `IdleTimeout`. Das macht den Dienst anfällig für Slowloris-/Ressourcenerschöpfungsangriffe. Sicherheit by design/default verlangt angemessene Schutzmaßnahmen.

**Konkrete Abhilfe:**  
`http.Server` mit Timeouts verwenden wie im Codebeispiel zu DSGVO-3. Die Timeouts müssen so gewählt werden, dass legitime Anfragen (Body-Limit 1 MiB) weiterhin funktionieren. Die Werte `ReadHeaderTimeout: 5s`, `ReadTimeout: 15s`, `WriteTimeout: 15s`, `IdleTimeout: 60s` sind für diese API realistisch.

---

### Befund CRA-2 — Security-Header fehlen
**Schweregrad:** niedrig

**Datei:** `response.go`

**Problem:**  
Für JSON-Antworten fehlt mindestens `X-Content-Type-Options: nosniff`. Darüber hinaus fehlt `Cache-Control: no-store` (siehe DSGVO-4). Dies ist ein einfacher, aber relevanter Baustein der Security-by-Default-Härtung.

**Konkrete Abhilfe:**  
Wie unter DSGVO-4 in `writeJSON` ergänzen. `DELETE /pastes/{id}` antwortet mit `204` und leerem Body; hier ist der Header weniger kritisch, aber eine zentrale Middleware in `newRouter` wäre sauberer.

---

### Befund CRA-3 — SBOM und dokumentierte Sicherheitseigenschaften fehlen
**Schweregrad:** mittel

**Datei:** `README.md` (existiert), neu: `SECURITY.md`, `SBOM.md` oder Abschnitt in `README.md`

**Problem:**  
Das Projekt hat ein `go.mod` ohne externe Abhängigkeiten, was die SBOM-Pflicht vereinfacht. Es gibt jedoch im sichtbaren Stand **keine SBOM, kein dokumentiertes Sicherheitskonzept und keinen Update-/Patch-Prozess**. Für ein Produkt mit digitalen Elementen sind das CRA-Pflichtbestandteile.

**Konkrete Abhilfe:**  
Im `README.md` oder in separaten Dateien ergänzen:

- **SBOM:** Go-Modul, Go-Version, Liste `direct dependencies: none`, Build-Angaben (`go build`). Beispiel:
  ```markdown
  ## SBOM
  - Modul: go-pastebin
  - Go-Version: (siehe go.mod)
  - Externe Abhängigkeiten: keine
  - Build: `go build .`
  ```
- **Sicherheitseigenschaften:** `crypto/rand` für 128-Bit-IDs, Body-Limit 1 MiB, Standardablauf 24 h, keine Protokollierung von Inhalten oder IP-Adressen, JSON-only-Fehler.
- **Update-/Patch-Prozess:** „Sicherheitsupdates werden über den regulären Build-/Release-Prozess bereitgestellt (neues Image/Binary + Rolling Deploy).“ Das beschreibt die Update-Fähigkeit des Produkts.

---

### Befund CRA-4 — Kein Rate-Limiting / Missbrauchsschutz
**Schweregrad:** niedrig

**Datei:** `main.go` oder neue Middleware

**Problem:**  
Die API hat keinen Schutz gegen massenhafte `POST`-Anfragen oder Brute-Force-Zugriffe. Auch wenn die IDs 128 Bit Entropie haben, kann ein ungebremster API-Zugang zu Missbrauch führen. Dies ist eine CRA-/Security-Empfehlung.

**Konkrete Abhilfe:**  
Eine einfache In-Memory-Rate-Limit-Middleware pro Client (z. B. Token-Bucket) ergänzen. Dabei **darf die Client-IP nicht protokolliert werden** und sollte nur flüchtig im RAM als Schlüssel verwendet werden, um AC-14 nicht zu verletzen. Ein Beispiel wäre ein Ratenlimit von z. B. 60 Requests/Minute pro Quell-IP für `POST /pastes`. Die konkrete Schwelle ist Betriebskonfiguration.

---

## 3. EU AI Act

**Nicht einschlägig.**  
Das Produkt enthält keine KI-Funktion; es gibt keine Modelle, keine automatisierte Entscheidungsfindung und keine KI-gestützte Inhaltsanalyse.

---

## 4. Pflichttexte & UI

**Nicht einschlägig.**  
Reines Backend ohne Endnutzer-UI. Es gibt keine Web-Oberfläche mit Impressumspflicht, Cookie-Banner, AGB oder Datenschutzerklärung **im Produkt**. Die Verantwortung für eine Datenschutzerklärung des Betreibers bleibt unberührt, ist aber kein Code-Befund.

---

## 5. Barrierefreiheit (WCAG/BITV/EAA)

**Nicht einschlägig.**  
Keine öffentliche Web-UI im Produkt. Die API selbst unterliegt nicht den Anforderungen an barrierefreie Benutzeroberflächen.

---

## Fazit

Der Kern der Anwendung ist solide: keine Protokollierung personenbezogener Daten, kryptografisch sichere IDs, Body-Limit, saubere JSON-Fehler und eine 24-Stunden-Standardablaufzeit. Die offenen Punkte sind behebbar und betreffen vor allem die **fehlende Obergrenze für die Ablaufzeit**, den **fehlenden aktiven Cleanup**, die **Transportverschlüsselung** sowie die **CRA-Härtung** (Timeouts, Security-Header, SBOM/Doku). Ein fundamentaler Verstoß, der eine Sperrung rechtfertigen würde, liegt nicht vor. Daher: `CHANGES_REQUESTED`.