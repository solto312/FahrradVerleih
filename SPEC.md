# Spezifikation: Fahrradverleih (Aufgabe 1 c)

Spec-Driven Development: Diese Datei ist die Quelle der Wahrheit. Code und Tests
werden aus ihr abgeleitet. Jede Anforderung hat eine ID (z. B. `A-3`), die in
Tests und Commits referenziert werden kann.

## 1. Ziel

Ein Prototyp für einen Fahrradverleih: Mitarbeitende verwalten Fahrräder und Kund:innen,
legen Ausleihen mit Beginn und Dauer an und sehen auf einem Dashboard sofort,
was verfügbar, verliehen oder überfällig ist.

## 2. Anforderungen

### Teil c) Fahrradverleih statt Ski-Verleih (5 Punkte)

Der Fahrradverleih übernimmt **alle Anforderungen aus a) und b)** des Ski-Verleihs
und überträgt sie auf Fahrräder. Fachliche Unterschiede:

| ID  | Anforderung |
|-----|-------------|
| C-1 | Verliehen werden **Fahrräder** (Marke, Modell, Typ, **Rahmengröße**, Zustand) statt Ski. |
| C-2 | Fahrrad-Typen: **City, Trekking, Mountainbike, E-Bike, Kinder** – jeder mit eigenem Stunden- und Tagespreis. |
| C-3 | Rahmengrößen: XS, S, M, L, XL sowie für Kinderräder 16, 20 und 24 Zoll (Auswahlliste, `GET /api/meta`). |
| C-4 | Stundenweise Ausleihe ist der Normalfall (Stadtrundfahrt 2 h, Tagestour, Wochenende) – Schnellauswahl wie B-1. |
| C-5 | API-Ressource heißt `/api/bikes`; alle übrigen Endpunkte bleiben gleich. |
| C-6 | Alle UI-Texte sprechen von Fahrrädern; domänenspezifische Texte stehen zentral in `web/config.js`. |

### Übernommen aus a) Verleih mit Dashboard

| ID  | Anforderung |
|-----|-------------|
| A-1 | **Dashboard** als Startseite: Kennzahlen *verfügbare Fahrräder*, *ausgeliehene Fahrräder*, *Fahrräder gesamt*, *registrierte Personen*, *überfällige* und *geplante* Ausleihen. |
| A-2 | Dashboard zeigt eine Liste **offener Ausleihen** (aktiv, überfällig, geplant) mit Rückgabe-Button; überfällige sind rot markiert. |
| A-3 | **Schnellzugriff** auf: Fahrräder verwalten, Personen verwalten, Neue Ausleihe. |
| A-4 | **Fahrrad-Verwaltung**: Fahrräder anlegen, bearbeiten, löschen (Marke, Modell, Typ, Rahmengröße, Zustand). Defekte Fahrräder sind nicht verleihbar. Fahrräder mit offener Ausleihe können nicht gelöscht werden. |
| A-5 | **Personenverwaltung**: Person anlegen, bearbeiten, löschen (Vorname, Nachname, E-Mail, Telefon). Personen mit offener Ausleihe können nicht gelöscht werden. |
| A-6 | **Ausweis eingeben**: Ausweistyp (Personalausweis, Reisepass, Führerschein) und Ausweisnummer sind Pflichtfelder einer Person. |
| A-7 | **Ausweis hochladen**: Pro Person kann ein Ausweis-Scan (JPG, PNG oder PDF, max. 5 MB) hochgeladen, angesehen und ersetzt werden. |
| A-8 | **Ausleihe anlegen** mit **Verleihbeginn** (Datum + Uhrzeit, Standard: jetzt) und **Verleihdauer**. Eine Ausleihe enthält eine Person und 1..n Fahrräder. |
| A-9 | Es werden nur Fahrräder angeboten, die im gewählten Zeitraum **frei** sind (keine Überschneidung mit anderen Ausleihen, nicht defekt). Doppelbuchungen werden vom Backend abgelehnt (HTTP 409). |
| A-10 | Status einer Ausleihe wird aus der Zeit berechnet: `geplant` (Beginn in der Zukunft), `aktiv`, `überfällig` (Ende überschritten, nicht zurückgegeben), `abgeschlossen`. |
| A-11 | **Rückgabe** aktiver/überfälliger Ausleihen; **Stornieren** geplanter Ausleihen. |
| A-12 | **Benutzerfreundlichkeit**: Suche/Filter in Listen, Rückmeldung per Toast, Bestätigung vor dem Löschen, verständliche deutsche Fehlermeldungen, Preis wird vor dem Buchen live angezeigt, responsive Layout. |

### Übernommen aus b) Flexibilisierung der Preise – stundenweise

| ID  | Anforderung |
|-----|-------------|
| B-1 | Die Verleihdauer wird in **Stunden** angegeben (min. 1 h, max. 14 Tage = 336 h). Schnellauswahl: 1 h, 2 h, 4 h, 1 Tag, 2 Tage, 3 Tage, 1 Woche. |
| B-2 | Preise sind **pro Fahrrad-Typ** konfigurierbar: **Stundenpreis** und **Tagespreis** (Tagesdeckel). |
| B-3 | Preisformel pro Fahrrad: `volle Tage × Tagespreis + min(Reststunden × Stundenpreis, Tagespreis)`. Damit ist eine lange Ausleihe nie teurer als die nächste volle Tagesanzahl. |
| B-4 | Preise sind im UI unter **„Preise“** änderbar; Änderungen gelten für neue Ausleihen. Bestehende Ausleihen behalten ihren gebuchten Preis. |
| B-5 | **Live-Preisvorschau** beim Anlegen einer Ausleihe (`GET /api/rentals/quote`). |
| B-6 | **Überziehung**: Wird nach dem geplanten Ende zurückgegeben, wird nach tatsächlich angefangenen Stunden neu berechnet (Endpreis = max(gebuchter Preis, Preis der tatsächlichen Dauer)). Frühere Rückgabe wird nicht erstattet. |
| B-7 | Geldbeträge werden intern in **Cent (int)** gespeichert, um Rundungsfehler zu vermeiden. |

Beispiel (City: 4 €/h, 18 €/Tag): 2 h = 8 €, 6 h = 18 € (Deckel, 24 € > 18 €), 26 h = 18 € + 8 € = 26 €.

## 3. Datenmodell (In-Memory)

```
Bike       { id, brand, model, type, frameSize, condition }
           type ∈ City | Trekking | Mountainbike | E-Bike | Kinder
           frameSize ∈ XS | S | M | L | XL | 16 Zoll | 20 Zoll | 24 Zoll
           condition ∈ Neu | Gut | Gebraucht | Defekt
Person     { id, firstName, lastName, email, phone,
             idDocument { type, number, fileName, hasFile } }
Rental     { id, personId, itemIds[], start, hours, end, status,
             priceCents, finalPriceCents, returnedAt, createdAt }
PriceRule  { type, hourlyCents, dailyCents }
Stats      { totalItems, availableNow, rentedNow, defective, persons,
             activeRentals, overdueRentals, plannedRentals }
```

## 4. REST-API (JSON)

| Methode | Pfad | Zweck |
|---|---|---|
| GET | `/api/stats` | Dashboard-Kennzahlen (A-1) |
| GET, POST | `/api/bikes` | Fahrräder auflisten / anlegen (A-4) |
| PUT, DELETE | `/api/bikes/{id}` | Fahrrad ändern / löschen |
| GET | `/api/bikes/available?start=RFC3339&hours=N` | freie Fahrräder im Zeitraum (A-9) |
| GET, POST | `/api/persons` | Personen auflisten / anlegen (A-5, A-6) |
| PUT, DELETE | `/api/persons/{id}` | Person ändern / löschen |
| POST | `/api/persons/{id}/id-document` | Ausweis hochladen, multipart Feld `file` (A-7) |
| GET | `/api/persons/{id}/id-document` | Ausweis-Datei abrufen |
| GET, POST | `/api/rentals` | Ausleihen auflisten / anlegen (A-8) |
| POST | `/api/rentals/{id}/return` | Rückgabe (A-11, B-6) |
| DELETE | `/api/rentals/{id}` | geplante Ausleihe stornieren (A-11) |
| GET | `/api/rentals/quote?itemIds=1,2&hours=N` | Preisvorschau (B-5) |
| GET, PUT | `/api/prices` | Preisliste lesen / ändern (B-2, B-4) |

Fehler: `{"error": "<deutsche Meldung>"}` mit 400 (ungültig), 404 (nicht
gefunden), 409 (Konflikt, z. B. Doppelbuchung).

## 5. Akzeptanzkriterien (werden durch `go test ./...` geprüft)

- Preisformel B-3 inkl. Tagesdeckel und Mehrtagesberechnung.
- Überschneidende Ausleihe desselben Fahrrads wird abgelehnt (A-9).
- Defektes Fahrrad kann nicht verliehen werden (A-4).
- Status-Berechnung geplant/aktiv/überfällig/abgeschlossen (A-10).
- Überziehung wird nachberechnet (B-6).
- Löschen von Fahrrad/Person mit offener Ausleihe → Konflikt (A-4, A-5).

## 6. Nicht-Ziele

Keine Persistenz (Neustart = Demo-Daten), kein Login, keine Zahlungsabwicklung.
