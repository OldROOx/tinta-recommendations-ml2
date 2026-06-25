package main

import "sort"

// domainKeywords: palabras representativas por dominio (los 9 del catálogo).
var domainKeywords = map[string][]string{
	"fisica":       {"fuerza", "energia", "masa", "velocidad", "particula", "onda", "campo", "atomo"},
	"matematicas":  {"ecuacion", "funcion", "derivada", "integral", "algebra", "calculo", "teorema", "vector"},
	"quimica":      {"molecula", "reaccion", "enlace", "compuesto", "elemento", "acido", "solucion"},
	"biologia":     {"celula", "organismo", "especie", "tejido", "proteina", "evolucion", "membrana"},
	"medicina":     {"hueso", "musculo", "organo", "sangre", "nervio", "anatomia", "enfermedad", "paciente"},
	"historia":     {"guerra", "imperio", "revolucion", "siglo", "civilizacion", "antiguo", "batalla"},
	"literatura":   {"novela", "poema", "personaje", "narrador", "verso", "relato", "metafora"},
	"programacion": {"funcion", "variable", "codigo", "algoritmo", "datos", "objeto", "memoria"},
	"idiomas":      {"verbo", "gramatica", "vocabulario", "pronunciacion", "frase", "idioma", "traduccion"},
}

// extractKeywords regresa las top-N palabras por frecuencia del texto del libro.
// (caja "Extractor de tema" del diagrama)
func extractKeywords(text string, topN int) []string {
	tf := termFreq(text)
	type kv struct {
		word  string
		count float64
	}
	arr := make([]kv, 0, len(tf))
	for w, c := range tf {
		arr = append(arr, kv{w, c})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].count > arr[j].count })
	if len(arr) > topN {
		arr = arr[:topN]
	}
	out := make([]string, len(arr))
	for i, k := range arr {
		out[i] = k.word
	}
	return out
}

// detectDomain elige el dominio con más coincidencias de palabras clave.
func detectDomain(text string) string {
	tf := termFreq(text)
	best, bestScore := "general", 0
	for dom, kws := range domainKeywords {
		score := 0
		for _, kw := range kws {
			if tf[kw] > 0 {
				score++
			}
		}
		if score > bestScore {
			bestScore, best = score, dom
		}
	}
	return best
}
