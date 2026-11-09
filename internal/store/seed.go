package store

import (
	"strings"
	"time"

	"fahrradverleih/internal/domain"
)

// Seed füllt den Store mit Demo-Daten (fiktive Personen).
func (s *Store) Seed(now time.Time) {
	bikes := []domain.Bike{
		{Brand: "KTM", Model: "Life Joy", Type: domain.TypeCity, FrameSize: "M", Condition: domain.CondGood},
		{Brand: "KTM", Model: "Life Joy", Type: domain.TypeCity, FrameSize: "S", Condition: domain.CondGood},
		{Brand: "Gazelle", Model: "Esprit", Type: domain.TypeCity, FrameSize: "L", Condition: domain.CondUsed},
		{Brand: "Gazelle", Model: "Esprit", Type: domain.TypeCity, FrameSize: "M", Condition: domain.CondNew},
		{Brand: "Cube", Model: "Nature EXC", Type: domain.TypeTrekking, FrameSize: "M", Condition: domain.CondGood},
		{Brand: "Cube", Model: "Nature EXC", Type: domain.TypeTrekking, FrameSize: "L", Condition: domain.CondGood},
		{Brand: "Kalkhoff", Model: "Endeavour 1.B", Type: domain.TypeTrekking, FrameSize: "S", Condition: domain.CondGood},
		{Brand: "Trek", Model: "FX 2", Type: domain.TypeTrekking, FrameSize: "XL", Condition: domain.CondUsed},
		{Brand: "Specialized", Model: "Rockhopper", Type: domain.TypeMTB, FrameSize: "M", Condition: domain.CondGood},
		{Brand: "Canyon", Model: "Grand Canyon 7", Type: domain.TypeMTB, FrameSize: "L", Condition: domain.CondGood},
		{Brand: "Scott", Model: "Aspect 950", Type: domain.TypeMTB, FrameSize: "S", Condition: domain.CondNew},
		{Brand: "Giant", Model: "Talon 2", Type: domain.TypeMTB, FrameSize: "M", Condition: domain.CondUsed},
		{Brand: "Haibike", Model: "Trekking 6", Type: domain.TypeEBike, FrameSize: "M", Condition: domain.CondGood},
		{Brand: "Haibike", Model: "AllMtn 3", Type: domain.TypeEBike, FrameSize: "L", Condition: domain.CondNew},
		{Brand: "Riese & Müller", Model: "Charger4", Type: domain.TypeEBike, FrameSize: "M", Condition: domain.CondGood},
		{Brand: "Cube", Model: "Kathmandu Hybrid", Type: domain.TypeEBike, FrameSize: "S", Condition: domain.CondGood},
		{Brand: "Woom", Model: "4", Type: domain.TypeKinder, FrameSize: "20 Zoll", Condition: domain.CondGood},
		{Brand: "Woom", Model: "3", Type: domain.TypeKinder, FrameSize: "16 Zoll", Condition: domain.CondGood},
		{Brand: "Puky", Model: "Cyke 24", Type: domain.TypeKinder, FrameSize: "24 Zoll", Condition: domain.CondUsed},
		{Brand: "Trek", Model: "Precaliber 20", Type: domain.TypeKinder, FrameSize: "20 Zoll", Condition: domain.CondNew},
	}
	for _, b := range bikes {
		_, _ = s.CreateBike(b)
	}

	people := []struct{ first, last, doc, nr string }{
		{"Anna", "Huber", "Personalausweis", "L01X00T47"},
		{"Lukas", "Gruber", "Reisepass", "P1234567"},
		{"Sophie", "Wagner", "Führerschein", "FS884512"},
		{"Felix", "Bauer", "Personalausweis", "T22000129"},
		{"Lea", "Moser", "Reisepass", "U7766554"},
		{"Jonas", "Steiner", "Personalausweis", "M88231XY"},
		{"Marie", "Hofer", "Führerschein", "FS100233"},
		{"David", "Berger", "Reisepass", "P9988776"},
		{"Laura", "Fuchs", "Personalausweis", "K55442211"},
		{"Tobias", "Leitner", "Führerschein", "FS776120"},
	}
	for _, p := range people {
		_, _ = s.CreatePerson(domain.Person{
			FirstName: p.first, LastName: p.last,
			Email:      strings.ToLower(p.first+"."+p.last) + "@example.com",
			Phone:      "+43 660 0000000",
			IDDocument: domain.IDDocument{Type: domain.IDDocType(p.doc), Number: p.nr},
		})
	}

	h := time.Hour
	start := now.Truncate(h)
	_, _ = s.CreateRental(domain.Rental{PersonID: 1, ItemIDs: []int{4}, Start: start.Add(-2 * h), Hours: 4}, now)
	_, _ = s.CreateRental(domain.Rental{PersonID: 2, ItemIDs: []int{10}, Start: start.Add(-26 * h), Hours: 48}, now)
	_, _ = s.CreateRental(domain.Rental{PersonID: 3, ItemIDs: []int{16}, Start: start.Add(-5 * h), Hours: 3}, now)   // überfällig
	_, _ = s.CreateRental(domain.Rental{PersonID: 4, ItemIDs: []int{1, 2}, Start: start.Add(24 * h), Hours: 8}, now) // geplant
}
