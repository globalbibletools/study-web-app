package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/starfederation/datastar-go/datastar"

	. "maragu.dev/gomponents"
	ds "maragu.dev/gomponents-datastar"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

func main() {
	fmt.Println("Hello, World!")

	http.HandleFunc("/endpoint", func(w http.ResponseWriter, r *http.Request) {
		sse := datastar.NewSSE(w, r)

		_ = sse.PatchElementGostar(apologize())

		time.Sleep(1 * time.Second)

		_ = sse.PatchElementGostar(waiting())
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_ = hello().Render(w)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func apologize() Node {
	return Div(ID("hal"), Text("I'm sorry, Dave. I'm afraid I can't do that"))
}

func waiting() Node {
	return Div(ID("hal"), Text("Waiting for an order..."))
}

func hello() Node {
	return layout(
		Button(ds.On("click", "@get('/endpoint')"), Text("Open the pod bay doors. Alan.")),
		Div(ID("hal")),
	)
}

func layout(children ...Node) Node {
	return HTML5(HTML5Props{
		Title:    "Global Bible Tools",
		Language: "en",
		Head: []Node{
			Script(Type("module"), Src("https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.2/bundles/datastar.js")),
		},
		Body: children,
	})
}
