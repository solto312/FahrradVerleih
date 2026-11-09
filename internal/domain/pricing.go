package domain

const (
	MinHours = 1
	MaxHours = 14 * 24 // B-1
)

// PriceRule: stundenweiser Preis mit Tagesdeckel pro Typ (B-2). Beträge in Cent (B-7).
type PriceRule struct {
	Type        ItemType `json:"type"`
	HourlyCents int      `json:"hourlyCents"`
	DailyCents  int      `json:"dailyCents"`
}

func (r *PriceRule) Validate() error {
	switch {
	case !r.Type.Valid():
		return invalid("unbekannter Typ")
	case r.HourlyCents <= 0 || r.DailyCents <= 0:
		return invalid("Preise müssen größer als 0 sein")
	case r.DailyCents < r.HourlyCents:
		return invalid("Tagespreis darf nicht kleiner als der Stundenpreis sein")
	}
	return nil
}

// Price berechnet den Preis für die angegebene Stundenzahl (B-3):
// volle Tage × Tagespreis + min(Reststunden × Stundenpreis, Tagespreis).
func (r PriceRule) Price(hours int) int {
	if hours <= 0 {
		return 0
	}
	days, rest := hours/24, hours%24
	return days*r.DailyCents + min(rest*r.HourlyCents, r.DailyCents)
}

// DefaultPrices ist die Start-Preisliste.
func DefaultPrices() []PriceRule {
	return []PriceRule{
		{TypeCity, 400, 1800},
		{TypeTrekking, 500, 2200},
		{TypeMTB, 700, 3500},
		{TypeEBike, 1000, 4500},
		{TypeKinder, 250, 1000},
	}
}
