package main

import (
	"context"

	"github.com/jackc/pgx/v5"
)

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

func GetChapterData(context context.Context, reference Reference, langCode string) (ChapterData, error) {
	rows, _ := dbpool.Query(context, `
		select
			verse.number as verse,
			word.id,
			word.text,
			coalesce(gloss.gloss, "") as gloss,
		from verse
		join word on word.verse_id = verse.id
		left join lateral (
			select gloss.gloss from phrase_word 
			left join phrase on phrase.id = phrase_word.phrase_id
			left join gloss on gloss.phrase_id = phrase.id
			where word.id = phrase_word.word_id
				and phrase.language_id = (select id from language where code = $3)
				and phrase.deleted_at is null
				and gloss.state = 'APPROVED'
		) as gloss on true
		where verse.book_id = $1
			and verse.chapter = $2
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
