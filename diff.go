package main

import (
	"bytes"
	diffLib "github.com/sergi/go-diff/diffmatchpatch"
	"html"
	"strings"
)

var replacer *strings.Replacer

func init() {
	replacer = strings.NewReplacer("\r", "", "\n", "&#8626;<br>")
}

func diffHTML(text1, text2 string) string {
	dmp := diffLib.New()
	diffs := dmp.DiffMain(text1, text2, false)

	var buff bytes.Buffer
	for _, diff := range diffs {
		text := replacer.Replace(html.EscapeString(diff.Text))
		switch diff.Type {
		case diffLib.DiffInsert:
			_, _ = buff.WriteString("<span class='webix_docmanager_diff_insert'>")
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("</span>")
		case diffLib.DiffDelete:
			_, _ = buff.WriteString("<span class='webix_docmanager_diff_remove'>")
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("</span>")
		case diffLib.DiffEqual:
			_, _ = buff.WriteString("<span>")
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("</span>")
		}
	}

	return buff.String()
}
