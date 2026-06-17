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
		sse.PatchElements(
			`<div id="hal">I’m sorry, Dave. I’m afraid I can’t do that.</div>`,
		)

		time.Sleep(1 * time.Second)

		sse.PatchElements(
			`<div id="hal">Waiting for an order...</div>`,
		)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		fmt.Fprintf(w, `
<DOCTYPE html>
<html>
	<head>
		<script type="module" src="https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.2/bundles/datastar.js"></script>
	</head>
	<body>
		<button data-on:click="@get('/endpoint')">
			Open the pod bay doors, HAL.
		</button>
		<div id="hal"></div>
	</body>
</html>
`)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
