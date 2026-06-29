// Servidor HTTP para que la app (Flutter) suba un PDF y reciba sus
// recomendaciones generadas. Usa el mismo pipeline que el modo CLI
// (pdfToMarkdown + Regenerate); no duplica lógica, solo cambia la entrada
// (multipart/form-data en vez de un argumento de terminal).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// runServer levanta el servidor HTTP. Bloquea (ListenAndServe).
func runServer(pool *pgxpool.Pool) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/recommendations/generate", corsWrap(handleGenerate(pool)))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	addr := ":" + port
	fmt.Printf("servidor de recomendaciones escuchando en %s\n", addr)
	fmt.Printf("  POST http://localhost:%s/api/v1/recommendations/generate\n", port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Println("error en el servidor:", err)
		os.Exit(1)
	}
}

// corsWrap permite que la app Flutter (en otro origen) pueda llamar al
// endpoint sin bloqueo de CORS, incluyendo el preflight OPTIONS.
func corsWrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

type generateResponse struct {
	Message               string `json:"message"`
	RecomendacionesNuevas int    `json:"recomendaciones_nuevas"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// handleGenerate atiende POST /api/v1/recommendations/generate.
//
// Espera multipart/form-data con:
//   - "user_id"   (texto, UUID del usuario)
//   - "book"      (archivo, el PDF subido)
//   - "questions" (texto, opcional, preguntas separadas por "|")
//
// Responde 200 con cuántas recomendaciones se generaron, o un error claro.
func handleGenerate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{"usa POST"})
			return
		}

		// Límite de 20 MB para el PDF subido.
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{"form inválido: " + err.Error()})
			return
		}

		userIDStr := r.FormValue("user_id")
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{"user_id inválido"})
			return
		}

		file, header, err := r.FormFile("book")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{"falta el archivo 'book' (el PDF)"})
			return
		}
		defer file.Close()

		questions := parseQuestions(r.FormValue("questions"))

		bookText, err := extraerTextoDeUpload(file, header.Filename)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errorResponse{err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		n, err := Regenerate(ctx, pool, userID, bookText, questions)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{"error generando recomendaciones: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, generateResponse{
			Message:               "recomendaciones generadas",
			RecomendacionesNuevas: n,
		})
	}
}

// extraerTextoDeUpload guarda el archivo subido en un temporal (la librería
// de PDF necesita un archivo en disco, no solo bytes en memoria), reusa
// pdfToMarkdown/leerLibro tal cual existen, y borra el temporal al salir.
func extraerTextoDeUpload(file io.Reader, nombreOriginal string) (string, error) {
	tmp, err := os.CreateTemp("", "upload-*.pdf")
	if err != nil {
		return "", fmt.Errorf("no se pudo crear archivo temporal: %w", err)
	}
	rutaTmp := tmp.Name()
	defer os.Remove(rutaTmp)

	if _, err := io.Copy(tmp, file); err != nil {
		tmp.Close()
		return "", fmt.Errorf("no se pudo guardar el archivo subido: %w", err)
	}
	tmp.Close()

	// Reutiliza leerLibro (main.go): detecta extensión por nombre original,
	// así que renombramos lógicamente vía el mismo flujo que .pdf siempre usa.
	texto, err := pdfToMarkdown(rutaTmp)
	if err != nil {
		return "", err
	}
	return texto, nil
}

// parseQuestions separa preguntas enviadas como "pregunta uno|pregunta dos".
func parseQuestions(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < len(raw); i++ {
		if raw[i] == '|' {
			out = append(out, raw[start:i])
			start = i + 1
		}
	}
	out = append(out, raw[start:])
	return out
}
