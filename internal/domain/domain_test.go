package domain

import (
	"testing"
	"time"
)

// B-3: volle Tage × Tagespreis + min(Reststunden × Stundenpreis, Tagespreis)
func TestPriceRule_Price(t *testing.T) {
	r := PriceRule{Type: TypeCity, HourlyCents: 600, DailyCents: 3500}
	cases := []struct{ hours, want int }{
		{0, 0},
		{1, 600},
		{3, 1800},
		{5, 3000},
		{6, 3500}, // 36 € → Tagesdeckel 35 €
		{8, 3500},
		{24, 3500},
		{25, 4100},
		{30, 7000}, // 35 + min(36, 35)
		{168, 7 * 3500},
	}
	for _, c := range cases {
		if got := r.Price(c.hours); got != c.want {
			t.Errorf("Price(%d) = %d, want %d", c.hours, got, c.want)
		}
	}
}

func TestPriceRule_Validate(t *testing.T) {
	bad := PriceRule{Type: TypeMTB, HourlyCents: 1000, DailyCents: 500}
	if bad.Validate() == nil {
		t.Error("Tagespreis < Stundenpreis sollte ungültig sein")
	}
}

// A-10
func TestRental_Status(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	mk := func(startOffset time.Duration, hours int) Rental {
		return Rental{Start: now.Add(startOffset), Hours: hours}
	}
	if s := mk(time.Hour, 2); s.Status(now) != StatusPlanned {
		t.Errorf("erwartet geplant, got %s", s.Status(now))
	}
	if s := mk(-time.Hour, 2); s.Status(now) != StatusActive {
		t.Errorf("erwartet aktiv, got %s", s.Status(now))
	}
	if s := mk(-3*time.Hour, 2); s.Status(now) != StatusOverdue {
		t.Errorf("erwartet überfällig, got %s", s.Status(now))
	}
	r := mk(-3*time.Hour, 2)
	r.ReturnedAt = &now
	if r.Status(now) != StatusReturned {
		t.Errorf("erwartet abgeschlossen, got %s", r.Status(now))
	}
}

func TestPerson_ValidateRequiresIDDocument(t *testing.T) {
	p := Person{FirstName: "Anna", LastName: "Huber"}
	if p.Validate() == nil {
		t.Error("Person ohne Ausweis sollte ungültig sein (A-6)")
	}
	p.IDDocument = IDDocument{Type: DocPassport, Number: "P1234567"}
	if err := p.Validate(); err != nil {
		t.Errorf("unerwarteter Fehler: %v", err)
	}
}

func TestIDDocument_AttachFileRejectsWrongType(t *testing.T) {
	var d IDDocument
	if d.AttachFile("x.exe", "application/octet-stream", []byte("MZ")) == nil {
		t.Error("nur JPG/PNG/PDF erlaubt (A-7)")
	}
}
