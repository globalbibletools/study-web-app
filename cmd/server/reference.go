package main

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

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

func ParseReference(reference string) Reference {
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

func ParseReferenceCode(reference string) Reference {
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

func (r Reference) FormatAsCode() string {
	return booksCodes[r.book-1] + "." + strconv.FormatUint(uint64(r.chapter), 10)
}

func (r Reference) Format() string {
	return books[r.book-1] + " " + strconv.FormatUint(uint64(r.chapter), 10)
}
