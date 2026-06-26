package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"crawshaw.io/sqlite/sqlitex"
	"github.com/starfederation/datastar-go/datastar"
	. "maragu.dev/gomponents"
	ds "maragu.dev/gomponents-datastar"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

type Signals struct {
	Chapter string `json:"chapter"`
}

var dbpool *sqlitex.Pool

func main() {
	var err error
	dbpool, err = sqlitex.Open("file:export.db", 0, 10)
	if err != nil {
		log.Fatal(err)
	}

	fileServer := http.FileServer(http.Dir("./web"))
	http.Handle("/static/", http.StripPrefix("/static/", fileServer))

	http.HandleFunc("/chapter/{chapterNumber}", func(w http.ResponseWriter, r *http.Request) {
		chapter, err := strconv.ParseUint(r.PathValue("chapterNumber"), 10, 32)
		if err != nil {
			log.Printf("Error: %v\n", err)
			return
		}

		chapterData, err := getChapterData(r.Context(), uint(chapter))
		if err != nil {
			log.Printf("Error: %v\n", err)
			http.Error(w, "Server Error", http.StatusInternalServerError)
		}

		sse := datastar.NewSSE(w, r)

		sse.PatchElementGostar(
			pageContent(chapterData),
			datastar.WithMode(datastar.ElementPatchModeReplace),
		)
		sse.PatchElementGostar(
			toolbar(chapterData),
		)

		chapterStr := strconv.FormatUint(uint64(chapter), 10)
		sse.MarshalAndPatchSignals(Signals{Chapter: chapterStr})
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		chapterData, err := getChapterData(r.Context(), 1)
		if err != nil {
			log.Printf("Error: %v\n", err)
			http.Error(w, "Server Error", http.StatusInternalServerError)
		}

		_ = read(chapterData).Render(w)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

type WordData struct {
	id   string
	text string
}

type VerseData struct {
	verseNumber uint
	words       []WordData
}

type ChapterData struct {
	BookName string
	Chapter  uint
	Verses   []VerseData
}

func getChapterData(context context.Context, chapter uint) (ChapterData, error) {
	conn := dbpool.Get(context)
	if conn == nil {
		return ChapterData{}, fmt.Errorf("failed to get database connection")
	}
	defer dbpool.Put(conn)

	stmt := conn.Prep(`
		select verse.number as verse, word.id, word.text from verse
		join word on word.verse_id = verse.id
		where book_id = $bookId
		and chapter = $chapter
		order by verse.id, word.id
	`)
	stmt.SetInt64("$bookId", 1)
	stmt.SetInt64("$chapter", int64(chapter))

	var verses []VerseData
	var verseNumber uint = 0
	var words []WordData
	for {
		if hasRow, err := stmt.Step(); err != nil {
			return ChapterData{}, err
		} else if !hasRow {
			break
		}

		newVerseNumber := uint(stmt.GetInt64("verse"))
		if newVerseNumber != verseNumber {
			if verseNumber > 0 {
				verses = append(verses, VerseData{
					verseNumber,
					words,
				})
				words = []WordData{}
			}
			verseNumber = newVerseNumber
		}

		words = append(words, WordData{
			id:   stmt.GetText("id"),
			text: stmt.GetText("text"),
		})
	}

	if len(words) > 0 {
		verses = append(verses, VerseData{
			verseNumber,
			words,
		})
	}

	stmt = conn.Prep(`
		select name from book
		where id = $bookId
	`)
	stmt.SetInt64("$bookId", 1)

	if hasRow, err := stmt.Step(); err != nil {
		return ChapterData{}, err
	} else if !hasRow {
		return ChapterData{}, nil
	}
	stmt.Step()

	return ChapterData{
		BookName: stmt.GetText("name"),
		Chapter:  chapter,
		Verses:   verses,
	}, nil
}

func pageContent(data ChapterData) Node {
	return Div(
		ID("content"),
		Div(
			Class("reading-content"),
			If(data.Chapter < 40, Dir("rtl")),
			If(data.Chapter >= 40, Dir("ltr")),
			P(
				Map(data.Verses, func(verse VerseData) Node {
					verseNumberStr := strconv.FormatUint(uint64(verse.verseNumber), 10)
					return Span(
						Span(
							Class("verse-number"),
							Text(verseNumberStr),
							Text(" "),
						),
						Map(verse.words, func(word WordData) Node {
							return Group([]Node{
								Span(Text(word.text)),
								If(!strings.HasSuffix(word.text, "־"), Text(" ")),
							})
						}),
					)
				}),
			),
		),
	)
}

func chapterInput(data ChapterData) Node {
	return Div(
		ID("chapter-input"),
		Class("chapter-input"),
		Input(
			Class("text-input"),
			ds.Bind("chapter"),
		),
		Div(
			Class("chapter-input-actions"),
			Button(
				ds.On("click", fmt.Sprintf("@get('/chapter/%d')", data.Chapter-1)),
				Text("P"),
			),
			Button(
				ds.On("click", fmt.Sprintf("@get('/chapter/%d')", data.Chapter+1)),
				Text("N"),
			),
		),
	)
}

func toolbar(data ChapterData) Node {
	return Div(
		ID("toolbar"),
		chapterInput(data),
	)
}

func read(data ChapterData) Node {
	pageStr := strconv.FormatUint(uint64(data.Chapter), 10)

	return layout(
		"/static/css/read.css",
		ds.Signals(map[string]any{
			"chapter": pageStr,
		}),
		toolbar(data),
		pageContent(data),
	)
}

func layout(
	styles string,
	children ...Node,
) Node {
	return HTML5(HTML5Props{
		Title:    "Global Bible Tools",
		Language: "en",
		Head: []Node{
			Script(Type("module"), Src("/static/scripts/datastar.js")),
			Link(Rel("stylesheet"), Href("/static/css/reset.css")),
			Link(Rel("stylesheet"), Href("/static/css/system.css")),
			Link(Rel("stylesheet"), Href("/static/css/components.css")),
			Link(Rel("preload"), Href("/static/fonts/SBL_Hbrw.woff2"), As("font"), Type("font/woff2"), CrossOrigin("")),
			Link(Rel("preload"), Href("/static/fonts/SBL_grk.woff2"), As("font"), Type("font/woff2"), CrossOrigin("")),
			Link(Rel("preload"), Href("/static/fonts/noto-sans-latin.woff2"), As("font"), Type("font/woff2"), CrossOrigin("")),
			If(len(styles) > 0, Link(Rel("stylesheet"), Href(styles))),
		},
		Body: children,
	})
}
