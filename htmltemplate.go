package xtremecore

import (
	"bytes"
	"html/template"
	"log"
	"os"
)

func MailHTMLTemplate(path string, vars interface{}) string {
	var buf bytes.Buffer

	baseDir, _ := os.Getwd()

	tmpl, err := template.ParseFiles(baseDir + "/internal/pkg/layout/email/" + path)
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}
	tmpl.Execute(&buf, vars)
	return buf.String()
}

func PDFHTMLTemplate(path string, vars interface{}) bytes.Buffer {
	var buf bytes.Buffer

	baseDir, _ := os.Getwd()

	tmpl, err := template.ParseFiles(baseDir + "/internal/pkg/layout/pdf/" + path)
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}
	tmpl.Execute(&buf, vars)
	return buf
}
