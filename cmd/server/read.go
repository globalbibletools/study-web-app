package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/starfederation/datastar-go/datastar"
)

type Signals struct {
	Reference string `json:"reference"`
	LangCode  string `json:"lang"`
}

var dbpool *pgxpool.Pool

func main() {
	var err error

	dbpool, err = pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	fileServer := http.FileServer(http.Dir("./web"))
	http.Handle("/static/", http.StripPrefix("/static/", fileServer))
	http.Handle("/favicon.ico", fileServer)

	http.HandleFunc("/next", func(w http.ResponseWriter, r *http.Request) {
		var signals Signals
		if err := datastar.ReadSignals(r, &signals); err != nil {
			http.Error(w, "Server Error", http.StatusInternalServerError)
		}

		reference := ParseReferenceCode(signals.Reference)
		reference.chapter += 1

		chapterData, err := GetChapterData(r.Context(), reference, signals.LangCode)
		if err != nil {
			http.Error(w, "Server Error", http.StatusInternalServerError)
		}

		sse := datastar.NewSSE(w, r)

		sse.PatchElementGostar(
			PageContent(chapterData),
			datastar.WithMode(datastar.ElementPatchModeReplace),
		)
		sse.PatchElementGostar(
			Toolbar(chapterData),
		)

		sse.MarshalAndPatchSignals(Signals{Reference: reference.FormatAsCode(), LangCode: signals.LangCode})
	})

	http.HandleFunc("/prev", func(w http.ResponseWriter, r *http.Request) {
		var signals Signals
		if err := datastar.ReadSignals(r, &signals); err != nil {
			http.Error(w, "Server Error", http.StatusInternalServerError)
		}

		reference := ParseReferenceCode(signals.Reference)
		reference.chapter -= 1

		chapterData, err := GetChapterData(r.Context(), reference, signals.LangCode)
		if err != nil {
			http.Error(w, "Server Error", http.StatusInternalServerError)
		}

		sse := datastar.NewSSE(w, r)

		sse.PatchElementGostar(
			PageContent(chapterData),
			datastar.WithMode(datastar.ElementPatchModeReplace),
		)
		sse.PatchElementGostar(
			Toolbar(chapterData),
		)

		sse.MarshalAndPatchSignals(Signals{Reference: reference.FormatAsCode(), LangCode: signals.LangCode})
	})

	http.HandleFunc("/reference", func(w http.ResponseWriter, r *http.Request) {
		reference := ParseReference(strings.ReplaceAll(r.URL.Query().Get("reference"), "+", " "))
		langCode := r.URL.Query().Get("lang")
		if langCode == "" {
			langCode = "eng"
		}

		chapterData, err := GetChapterData(r.Context(), reference, langCode)
		if err != nil {
			http.Error(w, "Server Error", http.StatusInternalServerError)
		}

		sse := datastar.NewSSE(w, r)

		sse.PatchElementGostar(
			PageContent(chapterData),
			datastar.WithMode(datastar.ElementPatchModeReplace),
		)
		sse.PatchElementGostar(
			Toolbar(chapterData),
		)

		sse.MarshalAndPatchSignals(Signals{Reference: reference.FormatAsCode(), LangCode: langCode})
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reference := ParseReferenceCode(r.URL.Query().Get("reference"))
		langCode := r.URL.Query().Get("lang")
		if langCode == "" {
			langCode = "eng"
		}

		chapterData, err := GetChapterData(r.Context(), reference, langCode)
		if err != nil {
			log.Printf("Error: %s\n", err)
			http.Error(w, "Server Error", http.StatusInternalServerError)
			return
		}

		_ = ReadPage(chapterData).Render(w)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
