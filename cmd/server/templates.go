package main

import (
	"strconv"
	"strings"

	. "maragu.dev/gomponents"
	ds "maragu.dev/gomponents-datastar"
	. "maragu.dev/gomponents/html"

	"gbtreader/internal/ui"
)

func PageContent(data ChapterData) Node {
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

func ChapterInput(data ChapterData) Node {
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

func LanguageInput(data ChapterData) Node {
	return ui.ComboboxInput(
		ui.ComboboxInputProps{
			Value: data.LangCode,
			Options: []ui.ComboboxInputOption{
				{Value: "eng", Name: "English"},
				{Value: "spa", Name: "Spanish"},
				{Value: "hin", Name: "Hindi"},
			},
		},
		Name("lang"),
		ds.On("change", "@get('/reference', {contentType: 'form'})"),
	)
}

func Toolbar(data ChapterData) Node {
	return Div(
		ID("toolbar"),
		Form(
			ds.On("submit", "@get('/reference', {contentType: 'form'})"),
			ChapterInput(data),
			LanguageInput(data),
		),
	)
}

func ReadPage(data ChapterData) Node {
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
		Toolbar(data),
		PageContent(data),
	)
}
