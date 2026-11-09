package domain

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalid markiert Validierungsfehler (→ HTTP 400).
var ErrInvalid = errors.New("ungültige Eingabe")

func invalid(msg string) error { return fmt.Errorf("%w: %s", ErrInvalid, msg) }

// ItemType ist die Kategorie eines Fahrrads; Preise hängen am Typ (B-2).
type ItemType string

const (
	TypeCity     ItemType = "City"
	TypeTrekking ItemType = "Trekking"
	TypeMTB      ItemType = "Mountainbike"
	TypeEBike    ItemType = "E-Bike"
	TypeKinder   ItemType = "Kinder"
)

// ItemTypes listet alle gültigen Typen in Anzeige-Reihenfolge.
var ItemTypes = []ItemType{TypeCity, TypeTrekking, TypeMTB, TypeEBike, TypeKinder}

func (t ItemType) Valid() bool {
	for _, v := range ItemTypes {
		if v == t {
			return true
		}
	}
	return false
}

// FrameSizes: Rahmengrößen (Kinderräder nach Laufradgröße).
var FrameSizes = []string{"XS", "S", "M", "L", "XL", "16 Zoll", "20 Zoll", "24 Zoll"}

func validFrameSize(s string) bool {
	for _, v := range FrameSizes {
		if v == s {
			return true
		}
	}
	return false
}

type Condition string

const (
	CondNew    Condition = "Neu"
	CondGood   Condition = "Gut"
	CondUsed   Condition = "Gebraucht"
	CondDefect Condition = "Defekt"
)

var Conditions = []Condition{CondNew, CondGood, CondUsed, CondDefect}

func (c Condition) Valid() bool {
	for _, v := range Conditions {
		if v == c {
			return true
		}
	}
	return false
}

// Bike ist ein verleihbares Fahrrad.
type Bike struct {
	ID        int       `json:"id"`
	Brand     string    `json:"brand"`
	Model     string    `json:"model"`
	Type      ItemType  `json:"type"`
	FrameSize string    `json:"frameSize"`
	Condition Condition `json:"condition"`
}

// Normalize trimmt Texteingaben.
func (b *Bike) Normalize() {
	b.Brand = strings.TrimSpace(b.Brand)
	b.Model = strings.TrimSpace(b.Model)
	b.FrameSize = strings.TrimSpace(b.FrameSize)
}

// Validate prüft die Pflichtfelder (A-4).
func (b *Bike) Validate() error {
	switch {
	case b.Brand == "":
		return invalid("Marke fehlt")
	case b.Model == "":
		return invalid("Modell fehlt")
	case !b.Type.Valid():
		return invalid("unbekannter Fahrrad-Typ")
	case !validFrameSize(b.FrameSize):
		return invalid("unbekannte Rahmengröße")
	case !b.Condition.Valid():
		return invalid("unbekannter Zustand")
	}
	return nil
}

// Rentable: defekte Fahrräder werden nicht verliehen (A-4).
func (b *Bike) Rentable() bool { return b.Condition != CondDefect }
