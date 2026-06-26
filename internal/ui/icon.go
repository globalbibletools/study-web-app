package ui

import (
	"strconv"

	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type IconProps struct {
	Class      string
	Icon       string
	FixedWidth bool
	Size       string
}

var iconSizeMap = map[string]float64{
	"xs":  0.75,
	"sm":  0.875,
	"md":  1,
	"lg":  1.25,
	"xl":  1.5,
	"2xl": 2,
}

func Icon(props IconProps) Node {
	size, exists := iconSizeMap[props.Size]
	if !exists {
		size = iconSizeMap["md"]
	}

	heightStyle := "height: " + strconv.FormatFloat(size, 'f', 2, 64) + "em;"
	widthStyle := "width: " + strconv.FormatFloat(size*1.25, 'f', 2, 64) + "em;"

	return SVG(
		Class("icon "+props.Class),
		Aria("hidden", "true"),
		Style(heightStyle+" "+widthStyle),
		Rawf(`<use href="/static/img/icons.svg#%s" />`, props.Icon),
	)
}
