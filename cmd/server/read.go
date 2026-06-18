package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/starfederation/datastar-go/datastar"

	. "maragu.dev/gomponents"
	ds "maragu.dev/gomponents-datastar"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

type Signals struct {
	Chapter string `json:"chapter"`
}

func main() {
	http.HandleFunc("/endpoint", func(w http.ResponseWriter, r *http.Request) {
		sse := datastar.NewSSE(w, r)

		signals := &Signals{}
		if err := datastar.ReadSignals(r, signals); err != nil {
			log.Printf("Error: %v\n", err)
			return
		}

		log.Printf("Reading chapter: %s\n", signals.Chapter)

		chapter, err := strconv.ParseUint(signals.Chapter, 10, 32)
		if err != nil {
			log.Printf("Error: %v\n", err)
			return
		}

		chapter += 1
		signals.Chapter = strconv.FormatUint(chapter, 10)

		if err := sse.PatchElementGostar(pageContent(signals.Chapter)); err != nil {
			log.Printf("Error: %v\n", err)
			return
		}
		if err := sse.MarshalAndPatchSignals(signals); err != nil {
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
				ds.On("click", "@get('/endpoint')"),
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
