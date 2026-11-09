# Fahrradverleih – Prototyp (Aufgabe 1 c)

Nur Prototyp. Priorität: schnell verständlich, leicht änderbar.
**Spec-Driven Development:** `SPEC.md` ist die Quelle der Wahrheit. Erst die Spec
ändern, dann Code und Tests. Anforderungs-IDs (`A-x`, `B-x`, `C-x`) in Kommentaren/Tests nennen.

## Stack
- Backend: Go 1.22+ (nur Standardbibliothek, `net/http` mit Method-Routing)
- DB: In-Memory (Maps + `sync.RWMutex`), Demo-Daten beim Start
- Frontend: reines HTML/CSS/JS in `web/` (kein Build-Schritt)
- Versionierung: lokales Git

## Commands
- Starten: `go run .` → http://localhost:8080 (Port via `PORT=9000 go run .`)
- Tests: `go test ./...` (mit Race-Detector: `go test -race ./...`)
- Format: `gofmt -w .`
- Vet: `go vet ./...`

## Architektur
- `main.go` → Einstieg, Store anlegen + seeden, Routing, Static-Files aus `web/`
- `internal/domain/` → Kern-Structs + Methoden: `Bike`, `Person`/`IDDocument`, `Rental`, `PriceRule`
  - Validierung (`Validate()`), Statusberechnung (`Rental.Status`), Preisformel (`PriceRule.Price`)
- `internal/store/` → thread-sicherer In-Memory-Store; Verfügbarkeitsprüfung und Buchung im selben Lock
- `internal/handler/` → HTTP-Handler (JSON), mappen Fehler auf 400/404/409
- `web/config.js` → domänenspezifische Texte/Felder; `web/app.js` ist generisch

## Herkunft
- Abgeleitet aus dem Ski-Verleih (a + b). Architektur identisch, Domäne `Ski` → `Bike`.
- Fahrradspezifisch: `internal/domain/bike.go`, `internal/store/seed.go`, Preise in `domain.DefaultPrices`, `web/config.js`.

## Konventionen
- Methoden an Structs statt freier Funktionen, wo es Sinn ergibt
- Fehler mit `fmt.Errorf("...: %w", err)` wrappen; `domain.ErrInvalid`, `store.ErrNotFound`, `store.ErrConflict`
- Geld immer in Cent (`int`), nie `float64`
- Zeit im Backend immer `time.Time` (RFC3339 über die API), `now` wird injiziert → testbar
- Keine globalen Variablen außer Konstanten/Listen in `domain`
- Keine neuen Dependencies ohne Nachfrage
- Keine Secrets im Code
- Fehlermeldungen auf Deutsch (werden im UI angezeigt)

## Frontend
- Minimales JS, kein Framework; jede Seite ist eine `render*()`-Funktion (Hash-Routing)
- API-Calls nur gegen `/api/*`, nie direkt gegen Store-Strukturen
- Nutzertext immer über `esc()` in HTML einsetzen

## Regeln
- Keine generierten Dateien von Hand editieren
- Commit-Messages: kurz und imperativ
- Bei Unklarheit fragen statt raten
- Prototyp-Fokus: lieber einfach und klar als elegant und komplex
