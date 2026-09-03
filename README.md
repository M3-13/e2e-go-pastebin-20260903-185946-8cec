# go-pastebin

Eine kleine, in-memory Pastebin-REST-API in Go, ausschließlich mit der Standardbibliothek (`net/http`) gebaut. Pasts werden per `POST /pastes` angelegt, per ID abgerufen und gelöscht, und `GET /pastes` liefert eine Metadatenliste ohne Inhalt. Der Store ist Mutex-geschützt, abgelaufene Pasts werden als 404 behandelt, und alle Fehler kommen als sauberes JSON.

## Tech Stack

- **Sprache**: Go (≥ 1.22)
- **HTTP**: `net/http` (Standardbibliothek, Go-1.22-Routing-Patterns)
- **Tests**: `testing` + `net/http/httptest`
- **Modul**: `go.mod` ohne externe Abhängigkeiten

## Installation / Start

Voraussetzung: Go ≥ 1.22.

```sh
go run .
```

Der Server startet auf Port `8080` (überschreibbar über die Umgebungsvariable `PORT`):

```sh
# PowerShell
$env:PORT="9090"; go run .

# Linux/macOS
PORT=9090 go run .
```

Testen:

```sh
go test ./...
```

## Endpunkte

| Methode | Pfad           | Beschreibung                                              |
| ------- | -------------- | --------------------------------------------------------- |
| GET     | `/health`      | Health-Check, antwortet `200 {"status":"ok"}`              |
| POST    | `/pastes`      | Legt einen Paste an                                        |
| GET     | `/pastes`      | Metadatenliste aller nicht abgelaufenen Pasts              |
| GET     | `/pastes/{id}` | Ruft einen Paste per ID ab                                 |
| DELETE  | `/pastes/{id}` | Löscht einen Paste per ID                                  |

### POST /pastes

Request-Body (JSON):

```json
{
  "content": "Hello, world",
  "language": "text",
  "expires_in_seconds": 3600
}
```

- `content` (string, Pflicht, nicht leer): der Paste-Inhalt.
- `language` (string, optional): die Sprache des Inhalts.
- `expires_in_seconds` (int64, optional, ≥ 0): Ablaufzeit in Sekunden; `0` oder fehlend = Standardablauf von 24 Stunden.

Antworten:

- `201` mit dem vollständigen Paste (inkl. zufälliger `id`).
- `400` bei fehlendem/leerem `content`, ungültigem JSON oder negativem Wert.
- `413` bei Request-Body größer 1 MiB.

### GET /pastes/{id}

- `200` mit dem vollständigen Paste.
- `404` bei unbekannter oder abgelaufener ID.

### DELETE /pastes/{id}

- `204` ohne Body bei Erfolg.
- `404` bei unbekannter oder abgelaufener ID.

### Fehlerformat

Alle Fehler kommen als JSON der Form `{"error":"msg"}` – ohne interne Details, Stacktraces, Dateipfade oder Speicheradressen.

## Features

- In-Memory-Speicher mit `sync.RWMutex` geschützt.
- Kryptographisch sichere Paste-IDs (128 Bit Entropie, `crypto/rand`).
- Automatischer Ablauf: Pasts ohne `expires_in_seconds` laufen nach 24 Stunden ab.
- Abgelaufene Pasts verhalten sich wie gelöscht (404) und werden bei Zugriff entfernt.
- Keine Protokollierung von Paste-Inhalten oder Client-IP-Adressen.
