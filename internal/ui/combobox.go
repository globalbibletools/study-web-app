package ui

import (
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type ComboboxInputOption struct {
	Name  string
	Value string
}

type ComboboxInputProps struct {
	Class   string
	Value 	string
	Options []ComboboxInputOption
}

func ComboboxInput(props ComboboxInputProps, children ...Node) Node {
	return Select(
		Class("text-input"),
		Group(children),
		Map(props.Options, func(option ComboboxInputOption) Node {
			return Option(
				Value(option.Value),
				Text(option.Name),
				If(option.Value == props.Value, Selected()),
			)
		}),
	)
}
