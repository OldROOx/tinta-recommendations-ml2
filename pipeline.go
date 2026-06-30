package main

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// bookNamespace: namespace fijo para derivar un UUID determinista del volume id.
var bookNamespace = uuid.MustParse("b1d7e2a0-1c3f-5e6a-9b2c-0f1a2b3c4d5e")

const (
	defaultTopK      = 5
	maxCandidates    = 40
	defaultClusters  = 3
	kmeansIterations = 10
	keywordsPerBook  = 10
)

// topicSubject mapea el dominio (es) al 'subject' de Google Books (en).
var topicSubject = map[string]string{
	"fisica": "physics", "matematicas": "mathematics", "quimica": "chemistry",
	"biologia": "biology", "medicina": "medicine", "historia": "history",
	"literatura": "literature", "programacion": "computers",
	"idiomas": "foreign language study", "general": "",
}

const upsertBookSQL = `
INSERT INTO recommended_books
    (book_id, google_volume_id, title, authors, thumbnail, info_link, description)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (book_id) DO UPDATE SET
    title       = EXCLUDED.title,
    authors     = EXCLUDED.authors,
    thumbnail   = EXCLUDED.thumbnail,
    info_link   = EXCLUDED.info_link,
    description = EXCLUDED.description,
    updated_at  = NOW();`

const upsertRecoSQL = `
INSERT INTO recommendations
    (user_id, book_id, score, cluster_id, source, generated_at, updated_at)
VALUES ($1, $2, $3, $4, 'content', NOW(), NOW())
ON CONFLICT (user_id, book_id) DO UPDATE SET
    score        = EXCLUDED.score,
    cluster_id   = EXCLUDED.cluster_id,
    source       = 'content',
    generated_at = NOW(),
    updated_at   = NOW();`

type scoredBook struct {
	cand    candidate
	score   float64
	cluster int
}

// Regenerate sigue el flujo del diagrama:
//
//	libro -> extractor de tema  ┐
//	preguntas -> vector interés ┴-> perfil -> Google Books ->
//	vectorización TF -> re-ranking coseno -> top 5 (K-Means) -> BD
func Regenerate(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID,
	bookText string, questions []string) (int, error) {

	// 1) Extractor de tema (del libro): dominio + keywords
	keywords := extractKeywords(bookText, keywordsPerBook)
	domain := detectDomain(bookText)

	// 2) + 3) Perfil de recomendación = tema (keywords) + interés (preguntas)
	parts := append([]string{}, keywords...)
	parts = append(parts, questions...)
	profile := termFreq(strings.Join(parts, " "))
	if len(profile) == 0 {
		profile = termFreq(domain) // fallback: nunca queda vacío
	}

	// 4) Google Books: subject(dominio) + keywords -> N libros
	client := &http.Client{Timeout: 10 * time.Second}
	cands, err := searchGoogleBooks(ctx, client, buildBookQuery(domain, keywords), maxCandidates)
	if err != nil {
		return 0, err
	}
	cands = dedup(cands)
	if len(cands) == 0 {
		return 0, nil
	}

	// 5) Vectorización TF de las descripciones ("encoder") +
	// 6) Re-ranking coseno (perfil vs candidato)
	docs := make([]string, len(cands))
	for i, c := range cands {
		docs[i] = c.Title + " " + c.Description
	}
	clusters := kmeans(vectorize(docs), defaultClusters, kmeansIterations)

	ranked := make([]scoredBook, len(cands))
	for i, c := range cands {
		ranked[i] = scoredBook{
			cand:    c,
			score:   cosineTF(profile, termFreq(docs[i])),
			cluster: clusters[i],
		}
	}

	// 7) Top 5: diversidad por cluster (K-Means) + guardar
	selected := diversify(ranked, defaultTopK)
	if err := save(ctx, pool, userID, selected); err != nil {
		return 0, err
	}
	return len(selected), nil
}

func dedup(cands []candidate) []candidate {
	seen := make(map[string]bool)
	out := cands[:0]
	for _, c := range cands {
		if c.VolumeID == "" || seen[c.VolumeID] {
			continue
		}
		seen[c.VolumeID] = true
		out = append(out, c)
	}
	return out
}

func buildBookQuery(domain string, keywords []string) string {
	var parts []string
	if subj := topicSubject[domain]; subj != "" {
		parts = append(parts, `subject:"`+subj+`"`)
	}
	kw := keywords
	if len(kw) > 4 {
		kw = kw[:4]
	}
	parts = append(parts, kw...)
	if len(parts) == 0 {
		return "libros"
	}
	return strings.Join(parts, " ")
}

func diversify(books []scoredBook, topK int) []scoredBook {
	sort.Slice(books, func(i, j int) bool { return books[i].score > books[j].score })

	byCluster := make(map[int][]scoredBook)
	var order []int
	for _, b := range books {
		if _, ok := byCluster[b.cluster]; !ok {
			order = append(order, b.cluster)
		}
		byCluster[b.cluster] = append(byCluster[b.cluster], b)
	}

	var selected []scoredBook
	idx := make(map[int]int)
	for len(selected) < topK {
		progressed := false
		for _, c := range order {
			i := idx[c]
			if i < len(byCluster[c]) {
				selected = append(selected, byCluster[c][i])
				idx[c] = i + 1
				progressed = true
				if len(selected) >= topK {
					break
				}
			}
		}
		if !progressed {
			break
		}
	}
	return selected
}

func save(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, books []scoredBook) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Borra las recomendaciones anteriores de este usuario antes de insertar
	// las nuevas, para que cada libro subido REEMPLACE a las anteriores en vez
	// de acumularse. Va dentro de la misma transacción: si algo falla, no se
	// pierde nada.
	if _, err := tx.Exec(ctx,
		`DELETE FROM recommendations WHERE user_id = $1`, userID,
	); err != nil {
		return err
	}

	for _, b := range books {
		bookID := uuid.NewSHA1(bookNamespace, []byte(b.cand.VolumeID))

		authors := b.cand.Authors
		if authors == nil {
			authors = []string{}
		}

		if _, err := tx.Exec(ctx, upsertBookSQL,
			bookID, b.cand.VolumeID, b.cand.Title, authors,
			nullStr(b.cand.Thumbnail), nullStr(b.cand.InfoLink), b.cand.Description,
		); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, upsertRecoSQL,
			userID, bookID, b.score, b.cluster,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func nullStr(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
