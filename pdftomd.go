// PDF -> Markdown.
//
// Convierte el PDF subido por el usuario a texto markdown simple. El
// resultado se le pasa al mismo pipeline que ya existía (extractor de tema,
// perfil, Google Books, TF-coseno, K-Means). Nada de eso cambia.
package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

// pdfToMarkdown lee un PDF y regresa su contenido como texto markdown básico.
// No intenta reconstruir formato exacto (encabezados, tablas); el objetivo es
// tener texto limpio y legible para el extractor de tema, no un documento
// visualmente fiel.
func pdfToMarkdown(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("no se pudo abrir el PDF: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	totalPaginas := r.NumPage()

	for i := 1; i <= totalPaginas; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		texto, err := page.GetPlainText(nil)
		if err != nil {
			continue // página problemática: se omite, no se detiene todo
		}
		buf.WriteString(texto)
		buf.WriteString("\n\n")
	}

	if buf.Len() == 0 {
		return "", fmt.Errorf("el PDF no tiene texto extraíble (¿está escaneado como imagen?)")
	}

	return limpiarComoMarkdown(buf.String()), nil
}

var (
	espaciosMultiples = regexp.MustCompile(`[ \t]{2,}`)
	saltosMultiples   = regexp.MustCompile(`\n{3,}`)
)

// limpiarComoMarkdown normaliza el texto crudo extraído del PDF: junta
// espacios sobrantes y colapsa saltos de línea excesivos. Resultado: un
// markdown plano (sin encabezados especiales), suficiente para el RAG/tema.
func limpiarComoMarkdown(texto string) string {
	texto = espaciosMultiples.ReplaceAllString(texto, " ")
	texto = saltosMultiples.ReplaceAllString(texto, "\n\n")
	return strings.TrimSpace(texto)
}
