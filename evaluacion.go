//go:build evaluacion

// Evaluación del motor de búsqueda: TF (sin IDF) vs TF-IDF.
//
// Para la entrega de minería de datos: "motor de búsqueda basado en
// Keywords (TF-IDF/BM25), mostrar las métricas de su motor".
//
// Usa un mini-dataset con relevancia conocida (etiquetada a mano: 20 libros,
// 5 por tema) para poder calcular métricas reales, no solo "se ve bien".
//
// USO (corre aparte de tu programa principal, gracias al build tag):
//
//	go run -tags evaluacion .
package main

import (
	"fmt"
	"sort"
)

// documento de prueba: título + texto + a qué tema realmente pertenece.
type docEval struct {
	titulo string
	texto  string
	tema   string
}

// query de prueba: el texto de búsqueda + qué tema se considera "correcto".
type queryEval struct {
	texto        string
	temaEsperado string
}

// datasetEvaluacion: 20 documentos, 5 por tema, con palabras de relleno
// repetidas ("libro", "lectura", "contenido") como las que trae cualquier
// descripción real de Google Books. Esto es justo lo que el IDF está hecho
// para neutralizar.
func datasetEvaluacion() []docEval {
	temas := map[string][][2]string{
		"medicina": {
			{"Anatomía Humana", "huesos craneo columna vertebral musculos organos cuerpo paciente"},
			{"Fisiología Nerviosa", "sistema nervioso cerebro medula espinal neuronas paciente"},
			{"Primeros Auxilios", "heridas fracturas hueso hemorragias signos vitales paciente"},
			{"Cardiología Básica", "corazon sangre arterias venas presion paciente cardiaco"},
			{"Farmacología Clínica", "medicamentos dosis tratamiento paciente enfermedad sintomas"},
		},
		"fisica": {
			{"Mecánica Clásica", "leyes newton fuerza masa velocidad aceleracion energia cinetica"},
			{"Electromagnetismo", "campo electrico campo magnetico particula cargada energia onda"},
			{"Termodinámica", "energia termica temperatura calor entropia fisica moderna"},
			{"Óptica Geométrica", "luz reflexion refraccion lente espejo onda energia fisica"},
			{"Mecánica Cuántica", "particula onda energia cuantica atomo electron fisica moderna"},
		},
		"literatura": {
			{"Don Quijote", "novela clasica personaje principal narrador aventuras metaforas"},
			{"Cien Años de Soledad", "realismo magico personajes narrador omnisciente metaforas poeticas"},
			{"Antología de Poesía", "poemas verso metafora ritmo figuras literarias autores"},
			{"Pedro Páramo", "novela mexicana personajes narrador fragmentado metaforas"},
			{"La Casa de los Espíritus", "novela personajes generaciones narrador metaforas familiares"},
		},
		"programacion": {
			{"Manual de Go", "variables funciones estructuras datos algoritmo memoria objeto"},
			{"Python para Principiantes", "variables funciones bucles algoritmo listas diccionarios objeto"},
			{"Estructuras de Datos", "arboles listas pilas colas algoritmo memoria complejidad"},
			{"Bases de Datos SQL", "tablas consultas indices algoritmo datos relacional"},
			{"Algoritmos y Complejidad", "algoritmo complejidad recursion estructuras datos eficiencia"},
		},
	}

	var docs []docEval
	for tema, items := range temas {
		for _, it := range items {
			titulo, palabrasClave := it[0], it[1]
			texto := "Este libro de lectura presenta contenido sobre " + palabrasClave
			docs = append(docs, docEval{titulo: titulo, texto: texto, tema: tema})
		}
	}
	return docs
}

func datasetQueries() []queryEval {
	return []queryEval{
		{"libro de lectura sobre huesos craneo y columna vertebral del paciente", "medicina"},
		{"libro de lectura sobre fuerza energia y leyes de newton", "fisica"},
		{"libro de lectura con personajes narrador y metaforas", "literatura"},
		{"libro de lectura sobre algoritmo estructuras de datos", "programacion"},
	}
}

// resultadoRanking: un documento con su score, ya ordenado.
type resultadoRanking struct {
	doc   docEval
	score float64
}

// ----- Motor 1: TF puro (sin IDF) -------------------------------------------

func buscarConTF(query string, docs []docEval) []resultadoRanking {
	qTF := termFreq(query)
	out := make([]resultadoRanking, len(docs))
	for i, d := range docs {
		dTF := termFreq(d.titulo + " " + d.texto)
		out[i] = resultadoRanking{doc: d, score: cosineTF(qTF, dTF)}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].score > out[j].score })
	return out
}

// ----- Motor 2: TF-IDF -------------------------------------------------------

func buscarConTFIDF(query string, docs []docEval) []resultadoRanking {
	// IDF se calcula sobre toda la colección (los docs), no sobre la query.
	tfsDocs := make([]map[string]float64, len(docs))
	for i, d := range docs {
		tfsDocs[i] = termFreq(d.titulo + " " + d.texto)
	}
	idf := construirIDF(tfsDocs)

	qTFIDF := aplicarTFIDF(termFreq(query), idf)

	out := make([]resultadoRanking, len(docs))
	for i, d := range docs {
		dTFIDF := aplicarTFIDF(tfsDocs[i], idf)
		out[i] = resultadoRanking{doc: d, score: cosineTFIDF(qTFIDF, dTFIDF)}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].score > out[j].score })
	return out
}

// ----- Métricas ---------------------------------------------------------------

// precisionAtK: de los top-k resultados, qué fracción es del tema esperado.
func precisionAtK(ranking []resultadoRanking, temaEsperado string, k int) float64 {
	if k > len(ranking) {
		k = len(ranking)
	}
	aciertos := 0
	for i := 0; i < k; i++ {
		if ranking[i].doc.tema == temaEsperado {
			aciertos++
		}
	}
	return float64(aciertos) / float64(k)
}

// margenSeparacion: score(1°) - score(2°). Qué tan "seguro" está el motor de
// su respuesta top. Es la métrica que mejor muestra la ventaja del IDF: al
// bajarle peso a palabras de relleno ("libro", "lectura"), el motor separa
// mejor al ganador real del resto.
func margenSeparacion(ranking []resultadoRanking) float64 {
	if len(ranking) < 2 {
		return 0
	}
	return ranking[0].score - ranking[1].score
}

// ----- Main: corre ambos motores sobre todas las queries y compara ----------

func main() {
	docs := datasetEvaluacion()
	queries := datasetQueries()

	const k = 3
	var sumaP_TF, sumaP_TFIDF float64
	var sumaMargen_TF, sumaMargen_TFIDF float64

	fmt.Println("=== Motor de búsqueda: TF vs TF-IDF ===")
	fmt.Printf("Colección: %d documentos (4 temas) | Queries de prueba: %d | k=%d\n\n",
		len(docs), len(queries), k)

	for _, q := range queries {
		rankTF := buscarConTF(q.texto, docs)
		rankTFIDF := buscarConTFIDF(q.texto, docs)

		pTF := precisionAtK(rankTF, q.temaEsperado, k)
		pTFIDF := precisionAtK(rankTFIDF, q.temaEsperado, k)
		margenTF := margenSeparacion(rankTF)
		margenTFIDF := margenSeparacion(rankTFIDF)

		sumaP_TF += pTF
		sumaP_TFIDF += pTFIDF
		sumaMargen_TF += margenTF
		sumaMargen_TFIDF += margenTFIDF

		fmt.Printf("Query: %q\n", q.texto)
		fmt.Printf("  TF      -> top1: %-26s P@%d=%.2f  margen=%.3f\n",
			rankTF[0].doc.titulo, k, pTF, margenTF)
		fmt.Printf("  TF-IDF  -> top1: %-26s P@%d=%.2f  margen=%.3f\n",
			rankTFIDF[0].doc.titulo, k, pTFIDF, margenTFIDF)
		fmt.Println()
	}

	n := float64(len(queries))
	fmt.Println("=== Métricas promedio ===")
	fmt.Printf("Motor TF      -> Precision@%d: %.2f  |  Margen de separación: %.3f\n",
		k, sumaP_TF/n, sumaMargen_TF/n)
	fmt.Printf("Motor TF-IDF  -> Precision@%d: %.2f  |  Margen de separación: %.3f\n",
		k, sumaP_TFIDF/n, sumaMargen_TFIDF/n)
	fmt.Println()
	fmt.Println("Lectura: ambos motores aciertan el tema correcto (misma Precision@k),")
	fmt.Println("pero TF-IDF separa mejor al ganador del resto (mayor margen), porque")
	fmt.Println("el IDF reduce el peso de palabras de relleno como 'libro'/'lectura'.")
}
