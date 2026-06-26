package ui

import (
	. "maragu.dev/gomponents"
	ds "maragu.dev/gomponents-datastar"
	. "maragu.dev/gomponents/html"
)

type ButtonProps struct {
	Class   string
	OnClick string
}

func Btn(props ButtonProps, children ...Node) Node {
	return Button(
		Class("btn "+props.Class),
		If(len(props.OnClick) > 0, ds.On("click", props.OnClick)),
		Group(children),
	)
}
