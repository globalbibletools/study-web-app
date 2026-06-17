package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

func main() {
	fmt.Println("Hello, World!")

	http.HandleFunc("/endpoint", func(w http.ResponseWriter, r *http.Request) {
		sse := datastar.NewSSE(w, r)

		sse.PatchElementTempl(
			apologize(),
		)

		time.Sleep(1 * time.Second)

		sse.PatchElements(
			`<div id="hal">Waiting for an order...</div>`,
		)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		component := hello()
		component.Render(r.Context(), w)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
