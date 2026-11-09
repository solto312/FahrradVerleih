package domain

import (
	"net/mail"
	"strings"
)

type IDDocType string

const (
	DocIDCard   IDDocType = "Personalausweis"
	DocPassport IDDocType = "Reisepass"
	DocLicense  IDDocType = "Führerschein"
)

var IDDocTypes = []IDDocType{DocIDCard, DocPassport, DocLicense}

func (t IDDocType) Valid() bool {
	for _, v := range IDDocTypes {
		if v == t {
			return true
		}
	}
	return false
}

// MaxIDFileBytes begrenzt die Größe eines Ausweis-Scans (A-7).
const MaxIDFileBytes = 5 << 20

// AllowedIDFileTypes sind die akzeptierten Content-Types für Ausweis-Scans.
var AllowedIDFileTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"application/pdf": true,
}

// IDDocument: Ausweis einer Person (A-6, A-7). Die Datei liegt im Speicher.
type IDDocument struct {
	Type        IDDocType `json:"type"`
	Number      string    `json:"number"`
	FileName    string    `json:"fileName"`
	HasFile     bool      `json:"hasFile"`
	ContentType string    `json:"-"`
	Data        []byte    `json:"-"`
}

// AttachFile setzt den hochgeladenen Scan.
func (d *IDDocument) AttachFile(name, contentType string, data []byte) error {
	if len(data) == 0 {
		return invalid("Datei ist leer")
	}
	if len(data) > MaxIDFileBytes {
		return invalid("Datei ist größer als 5 MB")
	}
	if !AllowedIDFileTypes[contentType] {
		return invalid("nur JPG, PNG oder PDF erlaubt")
	}
	d.FileName, d.ContentType, d.Data, d.HasFile = name, contentType, data, true
	return nil
}

type Person struct {
	ID         int        `json:"id"`
	FirstName  string     `json:"firstName"`
	LastName   string     `json:"lastName"`
	Email      string     `json:"email"`
	Phone      string     `json:"phone"`
	IDDocument IDDocument `json:"idDocument"`
}

func (p *Person) Normalize() {
	p.FirstName = strings.TrimSpace(p.FirstName)
	p.LastName = strings.TrimSpace(p.LastName)
	p.Email = strings.TrimSpace(p.Email)
	p.Phone = strings.TrimSpace(p.Phone)
	p.IDDocument.Number = strings.ToUpper(strings.TrimSpace(p.IDDocument.Number))
}

// Validate prüft Pflichtfelder inkl. Ausweis (A-5, A-6).
func (p *Person) Validate() error {
	switch {
	case p.FirstName == "":
		return invalid("Vorname fehlt")
	case p.LastName == "":
		return invalid("Nachname fehlt")
	case p.Email != "" && !validEmail(p.Email):
		return invalid("E-Mail-Adresse ist ungültig")
	case !p.IDDocument.Type.Valid():
		return invalid("Ausweistyp fehlt oder ist unbekannt")
	case len(p.IDDocument.Number) < 4:
		return invalid("Ausweisnummer fehlt (mind. 4 Zeichen)")
	}
	return nil
}

func (p *Person) FullName() string { return p.FirstName + " " + p.LastName }

func validEmail(s string) bool {
	a, err := mail.ParseAddress(s)
	return err == nil && a.Address == s
}
