package common

import (
	"regexp"
	"strings"
)

var backslash = regexp.MustCompile(`\\(\S)`)
var heading = regexp.MustCompile(`(?m)^(#{1,6} )`)
var orderedList = regexp.MustCompile(`(?m)^(\W* {0,3})(\d+)\. `)
var unorderedList = regexp.MustCompile(`(?m)^([^\\\w]*)[*+-] `)
var horizontalDivider = regexp.MustCompile(`(?m)^([-*_] *){3,}$`)
var blockquote = regexp.MustCompile(`(?m)^(\W* {0,3})> `)
var link = regexp.MustCompile(`[\[\]]`)

var replacer = strings.NewReplacer(
	`*`, `\*`,
	`_`, `\_`,
	"`", "\\`",
)

func Escape(text string) string {
	// Escape backslash escapes!
	text = backslash.ReplaceAllString(text, `\\$1`)

	// Escape headings
	text = heading.ReplaceAllString(text, `\$1`)

	// Escape hr
	text = horizontalDivider.ReplaceAllStringFunc(text, func(t string) string {
		if strings.Contains(t, "-") {
			return strings.Replace(t, "-", `\-`, 3)
		} else if strings.Contains(t, "_") {
			return strings.Replace(t, "_", `\_`, 3)
		}
		return strings.Replace(t, "*", `\*`, 3)
	})

	// Escape ol bullet points
	text = orderedList.ReplaceAllString(text, `$1$2\. `)

	// Escape ul bullet points
	text = unorderedList.ReplaceAllStringFunc(text, func(t string) string {
		return regexp.MustCompile(`([*+-])`).ReplaceAllString(t, `\$1`)
	})

	// Escape blockquote indents
	text = blockquote.ReplaceAllString(text, `$1\> `)

	// Escape em/strong *
	// Escape em/strong _
	// Escape code _
	text = replacer.Replace(text)

	// Escape link brackets
	text = link.ReplaceAllString(text, `\$&`)

	return text
}
