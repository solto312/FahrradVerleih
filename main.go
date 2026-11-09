// Fahrradverleih – Prototyp (Aufgabe 1 c). Siehe SPEC.md.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"fahrradverleih/internal/handler"
	"fahrradverleih/internal/store"
)

func main() {
	st := store.New()
	st.Seed(time.Now())

	mux := http.NewServeMux()
	handler.New(st).Routes(mux)
	mux.Handle("/", http.FileServer(http.Dir("web")))

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	log.Printf("Fahrradverleih läuft auf http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
