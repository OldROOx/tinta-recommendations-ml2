// Motor de búsqueda TF-IDF (para la entrega de minería de datos).
//
// Extiende tfcosine.go: en vez de comparar solo frecuencia de términos (TF),
// pondera cada palabra por qué tan "distintiva" es en la colección (IDF).
// Una palabra que aparece en TODOS los libros (ej. "libro", "historia") pesa
// poco; una que aparece en pocos (ej. "mitocondria") pesa mucho.
//
// TF-IDF(palabra, doc) = TF(palabra, doc) * IDF(palabra)
// IDF(palabra)         = log( N_documentos / (1 + docs_que_contienen_palabra) )
package main

import "math"

// construirIDF calcula el IDF de cada palabra a partir de la colección de
// documentos (en nuestro caso: los candidatos de Google Books).
func construirIDF(documentos []map[string]float64) map[string]float64 {
	n := float64(len(documentos))
	docFreq := make(map[string]float64) // en cuántos documentos aparece cada palabra

	for _, doc := range documentos {
		for palabra := range doc {
			docFreq[palabra]++
		}
	}

	idf := make(map[string]float64, len(docFreq))
	for palabra, df := range docFreq {
		idf[palabra] = math.Log(n / (1 + df))
	}
	return idf
}

// aplicarTFIDF convierte un vector TF en un vector TF-IDF usando el IDF ya
// calculado sobre la colección.
func aplicarTFIDF(tf map[string]float64, idf map[string]float64) map[string]float64 {
	tfidf := make(map[string]float64, len(tf))
	for palabra, frecuencia := range tf {
		tfidf[palabra] = frecuencia * idf[palabra]
	}
	return tfidf
}

// cosineTFIDF es cosineTF pero pensado semánticamente para vectores TF-IDF
// (la operación es la misma; se nombra distinto para que quede claro en el
// pipeline y en el video qué representa cada score).
func cosineTFIDF(a, b map[string]float64) float64 {
	return cosineTF(a, b)
}
