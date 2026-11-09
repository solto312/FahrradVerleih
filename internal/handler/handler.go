package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fahrradverleih/internal/domain"
	"fahrradverleih/internal/store"
)

// API verbindet HTTP-Routen mit dem Store.
type API struct {
	Store *store.Store
	Now   func() time.Time // austauschbar für Tests
}

func New(s *store.Store) *API { return &API{Store: s, Now: time.Now} }

// Routes registriert alle Endpunkte aus SPEC.md §4.
func (a *API) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/stats", a.stats)
	mux.HandleFunc("GET /api/meta", a.meta)

	mux.HandleFunc("GET /api/bikes", a.listBikes)
	mux.HandleFunc("POST /api/bikes", a.createBike)
	mux.HandleFunc("GET /api/bikes/available", a.availableBikes)
	mux.HandleFunc("PUT /api/bikes/{id}", a.updateBike)
	mux.HandleFunc("DELETE /api/bikes/{id}", a.deleteBike)

	mux.HandleFunc("GET /api/persons", a.listPersons)
	mux.HandleFunc("POST /api/persons", a.createPerson)
	mux.HandleFunc("PUT /api/persons/{id}", a.updatePerson)
	mux.HandleFunc("DELETE /api/persons/{id}", a.deletePerson)
	mux.HandleFunc("POST /api/persons/{id}/id-document", a.uploadIDFile)
	mux.HandleFunc("GET /api/persons/{id}/id-document", a.getIDFile)

	mux.HandleFunc("GET /api/rentals", a.listRentals)
	mux.HandleFunc("POST /api/rentals", a.createRental)
	mux.HandleFunc("GET /api/rentals/quote", a.quote)
	mux.HandleFunc("POST /api/rentals/{id}/return", a.returnRental)
	mux.HandleFunc("DELETE /api/rentals/{id}", a.cancelRental)

	mux.HandleFunc("GET /api/prices", a.getPrices)
	mux.HandleFunc("PUT /api/prices", a.setPrices)
}

// ---------- Hilfsfunktionen ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, store.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, store.ErrConflict):
		status = http.StatusConflict
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func readJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%w: JSON konnte nicht gelesen werden", domain.ErrInvalid)
	}
	return nil
}

func pathID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%w: ungültige ID", domain.ErrInvalid)
	}
	return id, nil
}

func queryHours(r *http.Request) (int, error) {
	h, err := strconv.Atoi(r.URL.Query().Get("hours"))
	if err != nil {
		return 0, fmt.Errorf("%w: Dauer (hours) fehlt", domain.ErrInvalid)
	}
	return h, nil
}

// ---------- Dashboard ----------

func (a *API) stats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.Store.Stats(a.Now()))
}

// meta liefert Auswahllisten für das Frontend (Typen, Zustände, Ausweistypen).
func (a *API) meta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"itemTypes":  domain.ItemTypes,
		"conditions": domain.Conditions,
		"docTypes":   domain.IDDocTypes,
		"maxHours":   domain.MaxHours,
		"frameSizes": domain.FrameSizes,
	})
}

// ---------- Bike ----------

func (a *API) listBikes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.Store.ListBikes())
}

func (a *API) createBike(w http.ResponseWriter, r *http.Request) {
	var bk domain.Bike
	if err := readJSON(r, &bk); err != nil {
		writeErr(w, err)
		return
	}
	out, err := a.Store.CreateBike(bk)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *API) updateBike(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	var bk domain.Bike
	if err := readJSON(r, &bk); err != nil {
		writeErr(w, err)
		return
	}
	out, err := a.Store.UpdateBike(id, bk)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) deleteBike(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err == nil {
		err = a.Store.DeleteBike(id)
	}
	if err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) availableBikes(w http.ResponseWriter, r *http.Request) {
	start, err := time.Parse(time.RFC3339, r.URL.Query().Get("start"))
	if err != nil {
		writeErr(w, fmt.Errorf("%w: Verleihbeginn (start) im Format RFC3339 erwartet", domain.ErrInvalid))
		return
	}
	hours, err := queryHours(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	out := a.Store.AvailableBikes(start, hours, a.Now())
	if out == nil {
		out = []domain.Bike{}
	}
	writeJSON(w, http.StatusOK, out)
}

// ---------- Personen ----------

func (a *API) listPersons(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.Store.ListPersons())
}

func (a *API) createPerson(w http.ResponseWriter, r *http.Request) {
	var p domain.Person
	if err := readJSON(r, &p); err != nil {
		writeErr(w, err)
		return
	}
	out, err := a.Store.CreatePerson(p)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *API) updatePerson(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	var p domain.Person
	if err := readJSON(r, &p); err != nil {
		writeErr(w, err)
		return
	}
	out, err := a.Store.UpdatePerson(id, p)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) deletePerson(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err == nil {
		err = a.Store.DeletePerson(id)
	}
	if err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// uploadIDFile nimmt einen Ausweis-Scan als multipart-Feld "file" entgegen (A-7).
func (a *API) uploadIDFile(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, domain.MaxIDFileBytes+(1<<20))
	f, hdr, err := r.FormFile("file")
	if err != nil {
		writeErr(w, fmt.Errorf("%w: keine Datei oder Datei zu groß (max. 5 MB)", domain.ErrInvalid))
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, domain.MaxIDFileBytes+1))
	if err != nil {
		writeErr(w, fmt.Errorf("Datei lesen: %w", err))
		return
	}
	// Typ anhand des Inhalts bestimmen, nicht anhand des Dateinamens.
	ctype := strings.Split(http.DetectContentType(data), ";")[0]
	out, err := a.Store.SetIDFile(id, hdr.Filename, ctype, data)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) getIDFile(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	doc, err := a.Store.IDFile(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", doc.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", doc.FileName))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(doc.Data)
}

// ---------- Ausleihen ----------

func (a *API) listRentals(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.Store.ListRentals(a.Now()))
}

func (a *API) createRental(w http.ResponseWriter, r *http.Request) {
	var in domain.Rental
	if err := readJSON(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	out, err := a.Store.CreateRental(in, a.Now())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *API) quote(w http.ResponseWriter, r *http.Request) {
	hours, err := queryHours(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	var ids []int
	for _, part := range strings.Split(r.URL.Query().Get("itemIds"), ",") {
		if part = strings.TrimSpace(part); part == "" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err != nil {
			writeErr(w, fmt.Errorf("%w: ungültige Fahrrad-ID", domain.ErrInvalid))
			return
		}
		ids = append(ids, id)
	}
	total, err := a.Store.Quote(ids, hours)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"priceCents": total, "hours": hours})
}

func (a *API) returnRental(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	out, err := a.Store.ReturnRental(id, a.Now())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) cancelRental(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err == nil {
		err = a.Store.CancelRental(id, a.Now())
	}
	if err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Preise ----------

func (a *API) getPrices(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.Store.Prices())
}

func (a *API) setPrices(w http.ResponseWriter, r *http.Request) {
	var rules []domain.PriceRule
	if err := readJSON(r, &rules); err != nil {
		writeErr(w, err)
		return
	}
	out, err := a.Store.SetPrices(rules)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
