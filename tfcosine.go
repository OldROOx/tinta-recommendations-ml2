package main

import (
	"math"
	"regexp"
	"strings"
)

// wordRe: palabras de 4+ letras (incluye acentos y ñ).
var wordRe = regexp.MustCompile(`[a-záéíóúñ]{4,}`)

// stopwords mínimas en español.
var stopwords = map[string]bool{
	"para": true, "como": true, "este": true, "esta": true, "estos": true,
	"estas": true, "pero": true, "porque": true, "cuando": true, "donde": true,
	"sobre": true, "entre": true, "todos": true, "todas": true, "desde": true,
	"hasta": true, "tambien": true, "los": true, "las": true, "una": true,
	"con": true, "por": true, "del": true, "sus": true, "mas": true, "son": true,
}

// termFreq construye el vector de frecuencia de términos de un texto.
func termFreq(text string) map[string]float64 {
	tf := make(map[string]float64)
	for _, w := range wordRe.FindAllString(strings.ToLower(text), -1) {
		if stopwords[w] {
			continue
		}
		tf[w]++
	}
	return tf
}

// cosineTF calcula la similitud coseno entre dos vectores TF.
func cosineTF(a, b map[string]float64) float64 {
	var dot, na, nb float64
	for _, v := range a {
		na += v * v
	}
	for w, v := range b {
		nb += v * v
		if av, ok := a[w]; ok {
			dot += av * v
		}
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
