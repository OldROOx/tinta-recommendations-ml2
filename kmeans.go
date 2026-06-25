package main

import "math"

// vectorize convierte textos a vectores densos sobre un vocabulario común
// (bolsa de palabras por frecuencia de términos). Reutiliza termFreq.
func vectorize(docs []string) [][]float64 {
	vocabIdx := make(map[string]int)
	tfs := make([]map[string]float64, len(docs))
	for i, d := range docs {
		tf := termFreq(d)
		tfs[i] = tf
		for w := range tf {
			if _, ok := vocabIdx[w]; !ok {
				vocabIdx[w] = len(vocabIdx)
			}
		}
	}
	dim := len(vocabIdx)
	vecs := make([][]float64, len(docs))
	for i, tf := range tfs {
		v := make([]float64, dim)
		for w, c := range tf {
			v[vocabIdx[w]] = c
		}
		vecs[i] = v
	}
	return vecs
}

// cosineDist = 1 - similitud coseno (0 = idénticos, 1 = sin relación).
func cosineDist(a, b []float64) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 1
	}
	return 1 - dot/(math.Sqrt(na)*math.Sqrt(nb))
}

// kmeans agrupa los vectores en k clusters (aprendizaje no supervisado).
// Regresa el índice de cluster asignado a cada punto. Init determinista
// (primeros k puntos) para que la demo sea reproducible.
func kmeans(vecs [][]float64, k, iterations int) []int {
	n := len(vecs)
	assign := make([]int, n)
	if n == 0 {
		return assign
	}
	if k > n {
		k = n
	}
	if k <= 1 {
		return assign // todos al cluster 0
	}
	dim := len(vecs[0])

	centroids := make([][]float64, k)
	for i := 0; i < k; i++ {
		centroids[i] = append([]float64(nil), vecs[i]...)
	}

	for iter := 0; iter < iterations; iter++ {
		changed := false

		// 1) Asignar cada punto al centroide más cercano
		for i, v := range vecs {
			best, bestD := 0, math.MaxFloat64
			for c := range centroids {
				if d := cosineDist(v, centroids[c]); d < bestD {
					bestD, best = d, c
				}
			}
			if assign[i] != best {
				changed = true
			}
			assign[i] = best
		}

		// 2) Recalcular centroides como la media de su cluster
		counts := make([]int, k)
		sums := make([][]float64, k)
		for c := range sums {
			sums[c] = make([]float64, dim)
		}
		for i, v := range vecs {
			c := assign[i]
			counts[c]++
			for j := range v {
				sums[c][j] += v[j]
			}
		}
		for c := range sums {
			if counts[c] > 0 {
				for j := range sums[c] {
					sums[c][j] /= float64(counts[c])
				}
				centroids[c] = sums[c]
			}
		}

		if !changed {
			break
		}
	}
	return assign
}
