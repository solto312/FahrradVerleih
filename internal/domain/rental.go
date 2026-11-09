package domain

import (
	"math"
	"time"
)

type RentalStatus string

const (
	StatusPlanned  RentalStatus = "geplant"
	StatusActive   RentalStatus = "aktiv"
	StatusOverdue  RentalStatus = "überfällig"
	StatusReturned RentalStatus = "abgeschlossen"
)

// Rental ist eine Ausleihe einer Person über 1..n Gegenstände (A-8).
type Rental struct {
	ID              int        `json:"id"`
	PersonID        int        `json:"personId"`
	ItemIDs         []int      `json:"itemIds"`
	Start           time.Time  `json:"start"`
	Hours           int        `json:"hours"`
	PriceCents      int        `json:"priceCents"`
	FinalPriceCents int        `json:"finalPriceCents"`
	ReturnedAt      *time.Time `json:"returnedAt"`
	CreatedAt       time.Time  `json:"createdAt"`
}

// RentalView ergänzt berechnete Felder für die API.
type RentalView struct {
	Rental
	End    time.Time    `json:"end"`
	Status RentalStatus `json:"status"`
}

func (r *Rental) Validate() error {
	switch {
	case r.PersonID <= 0:
		return invalid("Person fehlt")
	case len(r.ItemIDs) == 0:
		return invalid("mindestens ein Fahrrad muss gewählt sein")
	case r.Hours < MinHours || r.Hours > MaxHours:
		return invalid("Dauer muss zwischen 1 Stunde und 14 Tagen liegen")
	case r.Start.IsZero():
		return invalid("Verleihbeginn fehlt")
	}
	seen := map[int]bool{}
	for _, id := range r.ItemIDs {
		if seen[id] {
			return invalid("Fahrrad doppelt ausgewählt")
		}
		seen[id] = true
	}
	return nil
}

func (r *Rental) End() time.Time { return r.Start.Add(time.Duration(r.Hours) * time.Hour) }

// Status wird aus der Zeit abgeleitet (A-10).
func (r *Rental) Status(now time.Time) RentalStatus {
	switch {
	case r.ReturnedAt != nil:
		return StatusReturned
	case now.Before(r.Start):
		return StatusPlanned
	case now.After(r.End()):
		return StatusOverdue
	default:
		return StatusActive
	}
}

// Open: noch nicht zurückgegeben (geplant, aktiv oder überfällig).
func (r *Rental) Open() bool { return r.ReturnedAt == nil }

// blockedUntil: bis wann die Gegenstände belegt sind. Überfällige blockieren
// bis zur Rückgabe.
func (r *Rental) blockedUntil(now time.Time) time.Time {
	if r.ReturnedAt != nil {
		return *r.ReturnedAt
	}
	if now.After(r.End()) {
		return time.Unix(math.MaxInt32, 0)
	}
	return r.End()
}

// Overlaps prüft, ob der Zeitraum [start, end) mit dieser Ausleihe kollidiert (A-9).
func (r *Rental) Overlaps(start, end, now time.Time) bool {
	return start.Before(r.blockedUntil(now)) && r.Start.Before(end)
}

func (r *Rental) Contains(itemID int) bool {
	for _, id := range r.ItemIDs {
		if id == itemID {
			return true
		}
	}
	return false
}

// ActualHours: tatsächlich angefangene Stunden bis t.
func (r *Rental) ActualHours(t time.Time) int {
	h := int(math.Ceil(t.Sub(r.Start).Hours()))
	return max(h, 1)
}

func (r *Rental) View(now time.Time) RentalView {
	return RentalView{Rental: *r, End: r.End(), Status: r.Status(now)}
}
