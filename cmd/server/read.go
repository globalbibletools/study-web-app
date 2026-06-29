package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/starfederation/datastar-go/datastar"
	. "maragu.dev/gomponents"
	ds "maragu.dev/gomponents-datastar"
	. "maragu.dev/gomponents/html"

	"gbtreader/internal/ui"
)

type Signals struct {
	Reference string `json:"reference"`
}

var dbpool *sqlitex.Pool

type Reference struct {
	book    uint
	chapter uint
}

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

func parseReference(reference string) Reference {
	chapterStartIndex := strings.IndexAny(reference, "0123456789")
	if chapterStartIndex < 0 {
		return Reference{
			book:    1,
			chapter: 1,
		}
	}

	matches := fuzzy.RankFindNormalizedFold(strings.TrimSpace(reference[0:chapterStartIndex]), books)
	sort.Sort(matches)

	if len(matches) == 0 {
		return Reference{
			book:    1,
			chapter: 1,
		}
	}

	book := matches[0].OriginalIndex + 1
	chapter, err := strconv.ParseUint(reference[chapterStartIndex:], 10, 32)
	if err != nil {
		return Reference{
			book:    1,
			chapter: 1,
		}
	}

	return Reference{
		book:    uint(book),
		chapter: uint(chapter),
	}
}

func parseReferenceCode(reference string) Reference {
	splitIndex := strings.IndexRune(reference, '.')
	if splitIndex < 0 {
		book := uint(1 + slices.Index(booksCodes, reference))
		if book <= 0 {
			book = 1
		}

		return Reference{
			book:    book,
			chapter: 1,
		}
	}

	book := uint(1 + slices.Index(booksCodes, reference[0:splitIndex]))

	if book <= 0 {
		return Reference{
			book:    1,
			chapter: 1,
		}
	}

	chapter, err := strconv.ParseUint(reference[splitIndex+1:], 10, 32)
	if err != nil {
		return Reference{
			book:    book,
			chapter: 1,
		}
	}

	return Reference{
		book:    uint(book),
		chapter: uint(chapter),
	}
}

func formatReferenceCode(reference Reference) string {
	return booksCodes[reference.book-1] + "." + strconv.FormatUint(uint64(reference.chapter), 10)
}

func formatReference(reference Reference) string {
	return books[reference.book-1] + " " + strconv.FormatUint(uint64(reference.chapter), 10)
}

func main() {
	con, err := sqlite.OpenConn("file:export.db", 0)
	if err != nil {
		log.Fatal(err)
	}

	dbpool, err = sqlitex.Open("file::memory:?cache=shared", 0, 10)
	if err != nil {
		log.Fatal(err)
	}

	targetCon := dbpool.Get(context.Background())
	backup, err := con.BackupInit("", "", targetCon)
	if err != nil {
		log.Fatal(err)
	}
	if err := backup.Step(-1); err != nil {
		log.Fatal(err)
	}
	if err := backup.Finish(); err != nil {
		log.Fatal(err)
	}
	dbpool.Put(targetCon)
	_ = con.Close()

	log.Println("database loaded")

	fileServer := http.FileServer(http.Dir("./web"))
	http.Handle("/static/", http.StripPrefix("/static/", fileServer))
	http.Handle("/favicon.ico", fileServer)

	http.HandleFunc("/next", func(w http.ResponseWriter, r *http.Request) {
		var signals Signals
		if err := datastar.ReadSignals(r, &signals); err != nil {
			http.Error(w, "Server Error", http.StatusInternalServerError)
		}

		reference := parseReferenceCode(signals.Reference)
		reference.chapter += 1

		chapterData, err := getChapterData(r.Context(), reference)
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

		sse.MarshalAndPatchSignals(Signals{Reference: formatReferenceCode(reference)})
	})

	http.HandleFunc("/prev", func(w http.ResponseWriter, r *http.Request) {
		var signals Signals
		if err := datastar.ReadSignals(r, &signals); err != nil {
			http.Error(w, "Server Error", http.StatusInternalServerError)
		}

		reference := parseReferenceCode(signals.Reference)
		reference.chapter -= 1

		chapterData, err := getChapterData(r.Context(), reference)
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

		sse.MarshalAndPatchSignals(Signals{Reference: formatReferenceCode(reference)})
	})

	http.HandleFunc("/reference", func(w http.ResponseWriter, r *http.Request) {
		reference := parseReference(strings.ReplaceAll(r.URL.Query().Get("reference"), "+", " "))

		chapterData, err := getChapterData(r.Context(), reference)
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

		sse.MarshalAndPatchSignals(Signals{Reference: formatReferenceCode(reference)})
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reference := parseReferenceCode(r.URL.Query().Get("reference"))

		chapterData, err := getChapterData(r.Context(), reference)
		if err != nil {
			http.Error(w, "Server Error", http.StatusInternalServerError)
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
	BookName  string
	Verses    []VerseData
}

func getChapterData(context context.Context, reference Reference) (ChapterData, error) {
	conn := dbpool.Get(context)
	if conn == nil {
		return ChapterData{}, fmt.Errorf("failed to get database connection")
	}
	defer dbpool.Put(conn)

	stmt := conn.Prep(`
		select verse.number as verse, word.id, word.text, phrase.gloss from verse
		join word on word.verse_id = verse.id
		join phrase_word on word.id = phrase_word.word_id
		join phrase on phrase.id = phrase_word.phrase_id
		where book_id = $bookId
		and chapter = $chapter
		and phrase.language_id = (select id from language where code = $langCode)
		order by verse.id, word.id
	`)
	stmt.SetInt64("$bookId", int64(reference.book))
	stmt.SetInt64("$chapter", int64(reference.chapter))
	stmt.SetText("$langCode", "eng")

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
			id:    stmt.GetText("id"),
			text:  stmt.GetText("text"),
			gloss: stmt.GetText("gloss"),
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
		Reference: reference,
		BookName:  stmt.GetText("name"),
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
			Value(formatReference(data.Reference)),
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
			"reference": formatReferenceCode(data.Reference),
		}),
		ds.Effect(`
			const url = new URL(window.location);
			$reference ? url.searchParams.set('reference', $reference) : url.searchParams.delete('reference');
			window.history.replaceState({}, '', url);
		`),
		toolbar(data),
		pageContent(data),
	)
}
