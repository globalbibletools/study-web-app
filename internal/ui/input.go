package ui

import (
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type TextInputProps struct {
	Class string
}

func TextInput(props TextInputProps, children ...Node) Node {
	return Input(
		Class("text-input"),
		Group(children),
	)
}
