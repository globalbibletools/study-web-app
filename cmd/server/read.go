package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/starfederation/datastar-go/datastar"
	. "maragu.dev/gomponents"
	ds "maragu.dev/gomponents-datastar"
	. "maragu.dev/gomponents/html"

	"gbtreader/internal/ui"
)

type Signals struct {
	Reference string `json:"reference"`
	LangCode  string `json:"lang"`
}

var dbpool *pgxpool.Pool

var books = []string{
	"Genesis",
	"Exodus",
	"Leviticus",
}

var booksCodes = []string{
	"Gen",
	"Exo",
	"Lev",
}

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

		chapterData, err := getChapterData(r.Context(), reference, signals.LangCode)
		if err != nil {
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

		sse.MarshalAndPatchSignals(Signals{Reference: reference.FormatAsCode(), LangCode: signals.LangCode})
	})

	http.HandleFunc("/prev", func(w http.ResponseWriter, r *http.Request) {
		var signals Signals
		if err := datastar.ReadSignals(r, &signals); err != nil {
			http.Error(w, "Server Error", http.StatusInternalServerError)
		}

		reference := ParseReferenceCode(signals.Reference)
		reference.chapter -= 1

		chapterData, err := getChapterData(r.Context(), reference, signals.LangCode)
		if err != nil {
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

		sse.MarshalAndPatchSignals(Signals{Reference: reference.FormatAsCode(), LangCode: signals.LangCode})
	})

	http.HandleFunc("/reference", func(w http.ResponseWriter, r *http.Request) {
		var signals Signals
		if err := datastar.ReadSignals(r, &signals); err != nil {
			http.Error(w, "Server Error", http.StatusInternalServerError)
		}

		reference := ParseReference(strings.ReplaceAll(r.URL.Query().Get("reference"), "+", " "))

		chapterData, err := getChapterData(r.Context(), reference, signals.LangCode)
		if err != nil {
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

		sse.MarshalAndPatchSignals(Signals{Reference: reference.FormatAsCode(), LangCode: signals.LangCode})
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reference := ParseReferenceCode(r.URL.Query().Get("reference"))
		langCode := r.URL.Query().Get("lang")
		if langCode == "" {
			langCode = "eng"
		}

		chapterData, err := getChapterData(r.Context(), reference, langCode)
		if err != nil {
			log.Printf("Error: %s\n", err)
			http.Error(w, "Server Error", http.StatusInternalServerError)
			return
		}

		_ = read(chapterData).Render(w)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

type WordData struct {
	id    string
	text  string
	gloss string
}

type VerseData struct {
	verseNumber uint
	words       []WordData
}

type ChapterData struct {
	Reference Reference
	LangCode  string
	BookName  string
	Verses    []VerseData
}

func getChapterData(context context.Context, reference Reference, langCode string) (ChapterData, error) {
	rows, _ := dbpool.Query(context, `
		select
			verse.number as verse,
			word.id,
			word.text,
			gloss.gloss
		from verse
		join word on word.verse_id = verse.id
		join phrase_word on word.id = phrase_word.word_id
		join phrase on phrase.id = phrase_word.phrase_id
		join gloss on gloss.phrase_id = phrase.id
		where verse.book_id = $1
			and verse.chapter = $2
			and phrase.language_id = (select id from language where code = $3)
			and phrase.deleted_at is null
			and gloss.state = 'APPROVED'
		order by verse.id, word.id
	`, reference.book, reference.chapter, langCode,
	)

	var verses []VerseData
	var verseNumber uint = 0
	var words []WordData

	var word WordData
	var nextVerseNumber uint
	_, err := pgx.ForEachRow(rows, []any{&nextVerseNumber, &word.id, &word.text, &word.gloss}, func() error {
		if nextVerseNumber != verseNumber {
			if verseNumber > 0 {
				verses = append(verses, VerseData{
					verseNumber,
					words,
				})
				words = []WordData{}
			}
			verseNumber = nextVerseNumber
		}

		words = append(words, word)

		return nil
	})
	if err != nil {
		return ChapterData{}, err
	}

	if len(words) > 0 {
		verses = append(verses, VerseData{
			verseNumber,
			words,
		})
	}

	type BookRow struct {
		Name string
	}

	rows, _ = dbpool.Query(context, `
		select name from book
		where id = $1
	`, reference.book,
	)
	book, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[BookRow])
	if err != nil {
		return ChapterData{}, err
	}

	return ChapterData{
		Reference: reference,
		LangCode:  langCode,
		BookName:  book.Name,
		Verses:    verses,
	}, nil
}

func pageContent(data ChapterData) Node {
	return Div(
		ID("content"),
		Div(
			Class("reading-content"),
			If(data.Reference.book < 40, Dir("rtl")),
			If(data.Reference.book >= 40, Dir("ltr")),
			P(
				Map(data.Verses, func(verse VerseData) Node {
					verseNumberStr := strconv.FormatUint(uint64(verse.verseNumber), 10)
					return Span(
						Span(
							Class("verse-number"),
							Text(verseNumberStr),
						),
						Text(" "),
						Map(verse.words, func(word WordData) Node {
							return Group([]Node{
								Button(
									Class("gloss-popover-anchor"),
									PopoverTarget("gloss-"+word.id),
									PopoverTargetAction("show"),
									TabIndex("-1"),
									Text(word.text),
								),
								Span(
									ID("gloss-"+word.id),
									Class("gloss-popover"),
									Popover(),
									Text(word.gloss),
								),
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
		ui.TextInput(
			ui.TextInputProps{},
			Name("reference"),
			Value(data.Reference.Format()),
		),
		Div(
			Class("chapter-input-actions"),
			ui.Btn(
				ui.ButtonProps{
					OnClick: "@get('/prev')",
				},
				ui.Icon(ui.IconProps{
					Icon: "arrow-up",
				}),
			),
			ui.Btn(
				ui.ButtonProps{
					OnClick: "@get('/next')",
				},
				ui.Icon(ui.IconProps{
					Icon: "arrow-down",
				}),
			),
		),
	)
}

func toolbar(data ChapterData) Node {
	return Div(
		ID("toolbar"),
		Form(
			ds.On("submit", "@get('/reference', {contentType: 'form'})"),
			chapterInput(data),
		),
	)
}

func read(data ChapterData) Node {
	return ui.Layout(
		"/static/css/read.css",
		ds.Signals(map[string]any{
			"reference": data.Reference.FormatAsCode(),
			"lang":      data.LangCode,
		}),
		ds.Effect(`
			const url = new URL(window.location);
			$reference ? url.searchParams.set('reference', $reference) : url.searchParams.delete('reference');
			$lang ? url.searchParams.set('lang', $lang) : url.searchParams.delete('lang');
			window.history.replaceState({}, '', url);
		`),
		toolbar(data),
		pageContent(data),
	)
}
