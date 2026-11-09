package store

import (
	"errors"
	"sync"
	"testing"
	"time"

	"fahrradverleih/internal/domain"
)

var now = time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

func fixture(t *testing.T) (*Store, domain.Bike, domain.Person) {
	t.Helper()
	s := New()
	bk, err := s.CreateBike(domain.Bike{Brand: "KTM", Model: "Life Joy", Type: domain.TypeCity, FrameSize: "M", Condition: domain.CondGood})
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.CreatePerson(domain.Person{FirstName: "Anna", LastName: "Huber",
		IDDocument: domain.IDDocument{Type: domain.DocIDCard, Number: "L01X00T47"}})
	if err != nil {
		t.Fatal(err)
	}
	return s, bk, p
}

// A-9: keine Doppelbuchung
func TestCreateRental_RejectsOverlap(t *testing.T) {
	s, bk, p := fixture(t)
	if _, err := s.CreateRental(domain.Rental{PersonID: p.ID, ItemIDs: []int{bk.ID}, Start: now, Hours: 4}, now); err != nil {
		t.Fatal(err)
	}
	_, err := s.CreateRental(domain.Rental{PersonID: p.ID, ItemIDs: []int{bk.ID}, Start: now.Add(3 * time.Hour), Hours: 2}, now)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("erwartet Konflikt, got %v", err)
	}
	// direkt anschließend ist erlaubt
	if _, err := s.CreateRental(domain.Rental{PersonID: p.ID, ItemIDs: []int{bk.ID}, Start: now.Add(4 * time.Hour), Hours: 2}, now); err != nil {
		t.Fatalf("anschließende Ausleihe sollte gehen: %v", err)
	}
}

// A-4: defekte Fahrräder sind nicht verleihbar
func TestCreateRental_RejectsDefect(t *testing.T) {
	s, bk, p := fixture(t)
	bk.Condition = domain.CondDefect
	if _, err := s.UpdateBike(bk.ID, bk); err != nil {
		t.Fatal(err)
	}
	_, err := s.CreateRental(domain.Rental{PersonID: p.ID, ItemIDs: []int{bk.ID}, Start: now, Hours: 1}, now)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("erwartet Konflikt, got %v", err)
	}
	if len(s.AvailableBikes(now, 1, now)) != 0 {
		t.Error("defektes Fahrrad darf nicht als verfügbar gelten")
	}
}

// B-3/B-4: Preis wird bei Buchung festgehalten
func TestCreateRental_PriceSnapshot(t *testing.T) {
	s, bk, p := fixture(t)
	r, err := s.CreateRental(domain.Rental{PersonID: p.ID, ItemIDs: []int{bk.ID}, Start: now, Hours: 3}, now)
	if err != nil {
		t.Fatal(err)
	}
	if r.PriceCents != 1200 {
		t.Errorf("Preis = %d, want 1200", r.PriceCents)
	}
	if _, err := s.SetPrices([]domain.PriceRule{{Type: domain.TypeCity, HourlyCents: 1000, DailyCents: 5000}}); err != nil {
		t.Fatal(err)
	}
	if got := s.ListRentals(now)[0].PriceCents; got != 1200 {
		t.Errorf("gebuchter Preis darf sich nicht ändern, got %d", got)
	}
}

// B-6: Überziehung wird nachberechnet
func TestReturnRental_LateRecalculates(t *testing.T) {
	s, bk, p := fixture(t)
	r, _ := s.CreateRental(domain.Rental{PersonID: p.ID, ItemIDs: []int{bk.ID}, Start: now, Hours: 2}, now)
	out, err := s.ReturnRental(r.ID, now.Add(4*time.Hour+10*time.Minute)) // 5 angefangene Stunden
	if err != nil {
		t.Fatal(err)
	}
	if out.FinalPriceCents != 1800 {
		t.Errorf("Endpreis = %d, want 1800 (Tagesdeckel)", out.FinalPriceCents)
	}
	if out.Status != domain.StatusReturned {
		t.Errorf("Status = %s", out.Status)
	}
}

func TestReturnRental_EarlyKeepsPrice(t *testing.T) {
	s, bk, p := fixture(t)
	r, _ := s.CreateRental(domain.Rental{PersonID: p.ID, ItemIDs: []int{bk.ID}, Start: now, Hours: 5}, now)
	out, _ := s.ReturnRental(r.ID, now.Add(time.Hour))
	if out.FinalPriceCents != r.PriceCents {
		t.Errorf("frühe Rückgabe: %d != %d", out.FinalPriceCents, r.PriceCents)
	}
	// nach Rückgabe ist das Fahrrad wieder frei
	if len(s.AvailableBikes(now.Add(2*time.Hour), 1, now.Add(time.Hour))) != 1 {
		t.Error("Fahrrad sollte nach Rückgabe frei sein")
	}
}

// A-4/A-5: Löschen mit offener Ausleihe → Konflikt
func TestDelete_WithOpenRental(t *testing.T) {
	s, bk, p := fixture(t)
	_, _ = s.CreateRental(domain.Rental{PersonID: p.ID, ItemIDs: []int{bk.ID}, Start: now, Hours: 2}, now)
	if err := s.DeleteBike(bk.ID); !errors.Is(err, ErrConflict) {
		t.Errorf("Fahrrad löschen: erwartet Konflikt, got %v", err)
	}
	if err := s.DeletePerson(p.ID); !errors.Is(err, ErrConflict) {
		t.Errorf("Person löschen: erwartet Konflikt, got %v", err)
	}
}

func TestCancelRental_OnlyPlanned(t *testing.T) {
	s, bk, p := fixture(t)
	active, _ := s.CreateRental(domain.Rental{PersonID: p.ID, ItemIDs: []int{bk.ID}, Start: now, Hours: 1}, now)
	if err := s.CancelRental(active.ID, now); !errors.Is(err, ErrConflict) {
		t.Errorf("aktive Ausleihe darf nicht storniert werden, got %v", err)
	}
	planned, _ := s.CreateRental(domain.Rental{PersonID: p.ID, ItemIDs: []int{bk.ID}, Start: now.Add(24 * time.Hour), Hours: 1}, now)
	if err := s.CancelRental(planned.ID, now); err != nil {
		t.Errorf("geplante Ausleihe stornieren: %v", err)
	}
}

func TestStats(t *testing.T) {
	s := New()
	s.Seed(now)
	st := s.Stats(now)
	if st.TotalItems != 20 || st.Persons != 10 {
		t.Fatalf("Seed: %+v", st)
	}
	if st.AvailableNow+st.RentedNow+st.Defective != st.TotalItems {
		t.Errorf("Summen passen nicht: %+v", st)
	}
	if st.OverdueRentals != 1 || st.PlannedRentals != 1 || st.ActiveRentals != 2 {
		t.Errorf("Status-Zählung: %+v", st)
	}
}

// Thread-Safety: parallele Buchungen desselben Bike → genau eine gewinnt.
func TestCreateRental_Concurrent(t *testing.T) {
	s, bk, p := fixture(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.CreateRental(domain.Rental{PersonID: p.ID, ItemIDs: []int{bk.ID}, Start: now, Hours: 2}, now); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if ok != 1 {
		t.Errorf("erfolgreiche Buchungen = %d, want 1", ok)
	}
}
