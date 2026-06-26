package ui

import (
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

func Layout(
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
