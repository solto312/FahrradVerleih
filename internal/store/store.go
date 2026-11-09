package store

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"fahrradverleih/internal/domain"
)

var (
	ErrNotFound = errors.New("nicht gefunden")
	ErrConflict = errors.New("Konflikt")
)

// Stats sind die Dashboard-Kennzahlen (A-1).
type Stats struct {
	TotalItems     int `json:"totalItems"`
	AvailableNow   int `json:"availableNow"`
	RentedNow      int `json:"rentedNow"`
	Defective      int `json:"defective"`
	Persons        int `json:"persons"`
	ActiveRentals  int `json:"activeRentals"`
	OverdueRentals int `json:"overdueRentals"`
	PlannedRentals int `json:"plannedRentals"`
}

// Store ist der thread-sichere In-Memory-Speicher.
type Store struct {
	mu      sync.RWMutex
	bikes   map[int]domain.Bike
	persons map[int]domain.Person
	rentals map[int]domain.Rental
	prices  map[domain.ItemType]domain.PriceRule
	nextID  map[string]int
}

func New() *Store {
	s := &Store{
		bikes:   map[int]domain.Bike{},
		persons: map[int]domain.Person{},
		rentals: map[int]domain.Rental{},
		prices:  map[domain.ItemType]domain.PriceRule{},
		nextID:  map[string]int{},
	}
	for _, p := range domain.DefaultPrices() {
		s.prices[p.Type] = p
	}
	return s
}

func (s *Store) newID(kind string) int {
	s.nextID[kind]++
	return s.nextID[kind]
}

func sortedValues[T any](m map[int]T) []T {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	out := make([]T, 0, len(m))
	for _, k := range keys {
		out = append(out, m[k])
	}
	return out
}

// ---------- Bike (A-4) ----------

func (s *Store) ListBikes() []domain.Bike {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedValues(s.bikes)
}

func (s *Store) CreateBike(bk domain.Bike) (domain.Bike, error) {
	bk.Normalize()
	if err := bk.Validate(); err != nil {
		return domain.Bike{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	bk.ID = s.newID("bike")
	s.bikes[bk.ID] = bk
	return bk, nil
}

func (s *Store) UpdateBike(id int, bk domain.Bike) (domain.Bike, error) {
	bk.Normalize()
	if err := bk.Validate(); err != nil {
		return domain.Bike{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.bikes[id]; !ok {
		return domain.Bike{}, fmt.Errorf("Fahrrad %d: %w", id, ErrNotFound)
	}
	bk.ID = id
	s.bikes[id] = bk
	return bk, nil
}

func (s *Store) DeleteBike(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.bikes[id]; !ok {
		return fmt.Errorf("Fahrrad %d: %w", id, ErrNotFound)
	}
	for _, r := range s.rentals {
		if r.Open() && r.Contains(id) {
			return fmt.Errorf("%w: Fahrrad ist in einer offenen Ausleihe (#%d)", ErrConflict, r.ID)
		}
	}
	delete(s.bikes, id)
	return nil
}

// AvailableBikes liefert verleihbare Bike, die im Zeitraum frei sind (A-9).
func (s *Store) AvailableBikes(start time.Time, hours int, now time.Time) []domain.Bike {
	s.mu.RLock()
	defer s.mu.RUnlock()
	end := start.Add(time.Duration(hours) * time.Hour)
	var out []domain.Bike
	for _, bk := range sortedValues(s.bikes) {
		if bk.Rentable() && s.isFree(bk.ID, start, end, now, 0) {
			out = append(out, bk)
		}
	}
	return out
}

// isFree: Aufrufer hält den Lock. ignoreRental erlaubt das Ausschließen einer Ausleihe.
func (s *Store) isFree(itemID int, start, end, now time.Time, ignoreRental int) bool {
	for _, r := range s.rentals {
		if r.ID != ignoreRental && r.Contains(itemID) && r.Overlaps(start, end, now) {
			return false
		}
	}
	return true
}

// ---------- Personen (A-5, A-6, A-7) ----------

func (s *Store) ListPersons() []domain.Person {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedValues(s.persons)
}

func (s *Store) CreatePerson(p domain.Person) (domain.Person, error) {
	p.Normalize()
	if err := p.Validate(); err != nil {
		return domain.Person{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = s.newID("person")
	p.IDDocument.FileName, p.IDDocument.HasFile, p.IDDocument.Data = "", false, nil
	s.persons[p.ID] = p
	return p, nil
}

// UpdatePerson ändert Stammdaten; ein hochgeladener Scan bleibt erhalten.
func (s *Store) UpdatePerson(id int, p domain.Person) (domain.Person, error) {
	p.Normalize()
	if err := p.Validate(); err != nil {
		return domain.Person{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.persons[id]
	if !ok {
		return domain.Person{}, fmt.Errorf("Person %d: %w", id, ErrNotFound)
	}
	p.ID = id
	doc := old.IDDocument
	doc.Type, doc.Number = p.IDDocument.Type, p.IDDocument.Number
	p.IDDocument = doc
	s.persons[id] = p
	return p, nil
}

func (s *Store) DeletePerson(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.persons[id]; !ok {
		return fmt.Errorf("Person %d: %w", id, ErrNotFound)
	}
	for _, r := range s.rentals {
		if r.Open() && r.PersonID == id {
			return fmt.Errorf("%w: Person hat eine offene Ausleihe (#%d)", ErrConflict, r.ID)
		}
	}
	delete(s.persons, id)
	return nil
}

func (s *Store) SetIDFile(id int, name, contentType string, data []byte) (domain.Person, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.persons[id]
	if !ok {
		return domain.Person{}, fmt.Errorf("Person %d: %w", id, ErrNotFound)
	}
	if err := p.IDDocument.AttachFile(name, contentType, data); err != nil {
		return domain.Person{}, err
	}
	s.persons[id] = p
	return p, nil
}

func (s *Store) IDFile(id int) (domain.IDDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.persons[id]
	if !ok || !p.IDDocument.HasFile {
		return domain.IDDocument{}, fmt.Errorf("Ausweis-Datei: %w", ErrNotFound)
	}
	return p.IDDocument, nil
}

// ---------- Preise (B-2, B-4) ----------

func (s *Store) Prices() []domain.PriceRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.PriceRule, 0, len(domain.ItemTypes))
	for _, t := range domain.ItemTypes {
		out = append(out, s.prices[t])
	}
	return out
}

func (s *Store) SetPrices(rules []domain.PriceRule) ([]domain.PriceRule, error) {
	for i := range rules {
		if err := rules[i].Validate(); err != nil {
			return nil, err
		}
	}
	s.mu.Lock()
	for _, r := range rules {
		s.prices[r.Type] = r
	}
	s.mu.Unlock()
	return s.Prices(), nil
}

// quote: Aufrufer hält den Lock.
func (s *Store) quote(itemIDs []int, hours int) (int, error) {
	total := 0
	for _, id := range itemIDs {
		bk, ok := s.bikes[id]
		if !ok {
			return 0, fmt.Errorf("Fahrrad %d: %w", id, ErrNotFound)
		}
		total += s.prices[bk.Type].Price(hours)
	}
	return total, nil
}

// Quote berechnet den Preis vor dem Buchen (B-5).
func (s *Store) Quote(itemIDs []int, hours int) (int, error) {
	if hours < domain.MinHours || hours > domain.MaxHours {
		return 0, fmt.Errorf("%w: Dauer muss zwischen 1 Stunde und 14 Tagen liegen", domain.ErrInvalid)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.quote(itemIDs, hours)
}

// ---------- Ausleihen (A-8 … A-11, B-6) ----------

func (s *Store) ListRentals(now time.Time) []domain.RentalView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.RentalView, 0, len(s.rentals))
	for _, r := range sortedValues(s.rentals) {
		out = append(out, r.View(now))
	}
	return out
}

func (s *Store) CreateRental(r domain.Rental, now time.Time) (domain.RentalView, error) {
	if err := r.Validate(); err != nil {
		return domain.RentalView{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.persons[r.PersonID]; !ok {
		return domain.RentalView{}, fmt.Errorf("Person %d: %w", r.PersonID, ErrNotFound)
	}
	end := r.End()
	for _, id := range r.ItemIDs {
		bk, ok := s.bikes[id]
		if !ok {
			return domain.RentalView{}, fmt.Errorf("Fahrrad %d: %w", id, ErrNotFound)
		}
		if !bk.Rentable() {
			return domain.RentalView{}, fmt.Errorf("%w: %s %s ist defekt", ErrConflict, bk.Brand, bk.Model)
		}
		if !s.isFree(id, r.Start, end, now, 0) {
			return domain.RentalView{}, fmt.Errorf("%w: %s %s ist im Zeitraum bereits verliehen", ErrConflict, bk.Brand, bk.Model)
		}
	}
	price, err := s.quote(r.ItemIDs, r.Hours)
	if err != nil {
		return domain.RentalView{}, err
	}
	r.ID = s.newID("rental")
	r.ItemIDs = append([]int(nil), r.ItemIDs...)
	r.PriceCents, r.FinalPriceCents = price, price
	r.ReturnedAt, r.CreatedAt = nil, now
	s.rentals[r.ID] = r
	return r.View(now), nil
}

// ReturnRental schließt eine aktive/überfällige Ausleihe ab und berechnet
// bei Überziehung nach (B-6).
func (s *Store) ReturnRental(id int, now time.Time) (domain.RentalView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rentals[id]
	if !ok {
		return domain.RentalView{}, fmt.Errorf("Ausleihe %d: %w", id, ErrNotFound)
	}
	switch r.Status(now) {
	case domain.StatusReturned:
		return domain.RentalView{}, fmt.Errorf("%w: Ausleihe ist bereits abgeschlossen", ErrConflict)
	case domain.StatusPlanned:
		return domain.RentalView{}, fmt.Errorf("%w: Ausleihe hat noch nicht begonnen – bitte stornieren", ErrConflict)
	}
	if actual := r.ActualHours(now); actual > r.Hours {
		if p, err := s.quote(r.ItemIDs, actual); err == nil {
			r.FinalPriceCents = max(r.PriceCents, p)
		}
	}
	t := now
	r.ReturnedAt = &t
	s.rentals[id] = r
	return r.View(now), nil
}

// CancelRental storniert eine geplante Ausleihe (A-11).
func (s *Store) CancelRental(id int, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rentals[id]
	if !ok {
		return fmt.Errorf("Ausleihe %d: %w", id, ErrNotFound)
	}
	if r.Status(now) != domain.StatusPlanned {
		return fmt.Errorf("%w: nur geplante Ausleihen können storniert werden", ErrConflict)
	}
	delete(s.rentals, id)
	return nil
}

// ---------- Dashboard (A-1) ----------

func (s *Store) Stats(now time.Time) Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := Stats{TotalItems: len(s.bikes), Persons: len(s.persons)}
	rented := map[int]bool{}
	for _, r := range s.rentals {
		switch r.Status(now) {
		case domain.StatusActive:
			st.ActiveRentals++
		case domain.StatusOverdue:
			st.OverdueRentals++
		case domain.StatusPlanned:
			st.PlannedRentals++
			continue
		default:
			continue
		}
		for _, id := range r.ItemIDs {
			rented[id] = true
		}
	}
	for id, bk := range s.bikes {
		switch {
		case rented[id]:
			st.RentedNow++
		case !bk.Rentable():
			st.Defective++
		default:
			st.AvailableNow++
		}
	}
	return st
}
