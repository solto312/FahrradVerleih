# Fahrradverleih

Prototyp eines Fahrradverleihs mit Dashboard – Aufgabe 1 **c)**,
entwickelt nach Spec-Driven Development (siehe [`SPEC.md`](SPEC.md)).

## Schnellstart

Voraussetzung: Go ≥ 1.22

```bash
go run .
# → http://localhost:8080
```

Beim Start werden Demo-Daten geladen (20 Fahrräder, 10 fiktive Personen, 4 Ausleihen).
Alle Daten liegen im Speicher – ein Neustart setzt alles zurück.

```bash
go test ./...      # Akzeptanztests aus SPEC.md §5
```

## Funktionen

Alle Funktionen aus Aufgabe a) und b) des Ski-Verleihs, übertragen auf Fahrräder:

- **Dashboard**: verfügbare / ausgeliehene / gesamte Fahrräder, registrierte Personen,
  überfällige und geplante Ausleihen, Liste offener Ausleihen mit Rückgabe-Button
- **Fahrrad-Verwaltung**: Marke, Modell, Typ (City, Trekking, Mountainbike, E-Bike, Kinder),
  Rahmengröße (XS–XL, 16/20/24 Zoll), Zustand; Suche, Typ-Filter, Live-Status, „Verleihen“-Button
- **Personen** mit **Ausweis** (Typ + Nummer, Scan-Upload per Drag & Drop, JPG/PNG/PDF ≤ 5 MB)
- **Neue Ausleihe**: Person → **Verleihbeginn** + **Dauer in Stunden** (1 h … 1 Woche) →
  freie Fahrräder auswählen, Live-Preis; keine Doppelbuchungen
- **Preise** pro Fahrrad-Typ: Stundenpreis + Tagespreis (Deckel), im UI änderbar,
  Nachberechnung bei verspäteter Rückgabe

| Typ | Stunde | Tag (Deckel) |
|---|---|---|
| City | 4 € | 18 € |
| Trekking | 5 € | 22 € |
| Mountainbike | 7 € | 35 € |
| E-Bike | 10 € | 45 € |
| Kinder | 2,50 € | 10 € |

## Unterschied zum Ski-Verleih

| | Ski-Verleih | Fahrradverleih |
|---|---|---|
| Domain-Struct | `Ski` (Länge in cm) | `Bike` (Rahmengröße) |
| Typen | Allround, Carving, Freeride, Touren, Kinder | City, Trekking, Mountainbike, E-Bike, Kinder |
| API | `/api/skis` | `/api/bikes` |
| UI-Texte | `web/config.js` | `web/config.js` |

Store, Handler, Preislogik, Ausleih- und Personenlogik sind unverändert übernommen.

## Projektstruktur

```
main.go                  Einstieg, Routing, Static-Server
internal/domain/         Bike, Person, Rental, PriceRule (+ Tests)
internal/store/          thread-sicherer In-Memory-Store, Demo-Daten (+ Tests)
internal/handler/        REST-API (JSON)
web/                     index.html, style.css, config.js, app.js
SPEC.md                  Spezifikation (Anforderungen, API, Akzeptanzkriterien)
CLAUDE.md                Arbeitsanweisungen für Claude Code
```

## API (Auszug)

| Methode | Pfad | Zweck |
|---|---|---|
| GET | `/api/stats` | Dashboard-Kennzahlen |
| GET/POST/PUT/DELETE | `/api/bikes[/{id}]` | Fahrrad-CRUD |
| GET | `/api/bikes/available?start=…&hours=…` | freie Fahrräder |
| GET/POST/PUT/DELETE | `/api/persons[/{id}]` | Personen-CRUD |
| POST/GET | `/api/persons/{id}/id-document` | Ausweis hoch-/herunterladen |
| GET/POST | `/api/rentals` | Ausleihen |
| POST | `/api/rentals/{id}/return` | Rückgabe |
| GET | `/api/rentals/quote?itemIds=…&hours=…` | Preisvorschau |
| GET/PUT | `/api/prices` | Preisliste |

Vollständig in [`SPEC.md`](SPEC.md) §4.
