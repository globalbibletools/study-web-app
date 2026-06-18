package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	. "maragu.dev/gomponents"
	ds "maragu.dev/gomponents-datastar"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

type Signals struct {
	Chapter string `json:"chapter"`
}

func main() {
	http.HandleFunc("/chapter/{chapterNumber}", func(w http.ResponseWriter, r *http.Request) {
		chapter, err := strconv.ParseUint(r.PathValue("chapterNumber"), 10, 32)
		if err != nil {
			log.Printf("Error: %v\n", err)
			return
		}

		if err := read(uint(chapter)).Render(w); err != nil {
			log.Printf("Error: %v\n", err)
			return
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_ = read(1).Render(w)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func pageContent(page string) Node {
	return Div(ID("content"), Text("Chapter "), Text(page))
}

func chapterInput() Node {
	return Input(ID("chapter-input"), ds.Bind("chapter"))
}

func read(page uint) Node {
	pageStr := strconv.FormatUint(uint64(page), 10)

	return layout(
		Div(
			ds.Signals(map[string]any{
				"chapter": pageStr,
			}),
			chapterInput(),
			Button(
				ds.On("click", fmt.Sprintf("@get('/chapter/%d')", page-1)),
				Text("Prev Chapter"),
			),
			Button(
				ds.On("click", fmt.Sprintf("@get('/chapter/%d')", page+1)),
				Text("Next Chapter"),
			),
		),
		pageContent(pageStr),
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
