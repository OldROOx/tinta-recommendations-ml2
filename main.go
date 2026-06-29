// Servicio de recomendaciones de Tinta (Go).
//
// Dos modos:
//  1. CLI (igual que siempre): go run . <user_id> <archivo.pdf> [preguntas...]
//  2. SERVIDOR HTTP (para que la app suba el PDF):
//     go run . -server
//     Expone POST /api/v1/recommendations/generate (ver server.go)
//
// Flujo interno (sin importar el modo): PDF -> markdown -> extractor de tema
// -> perfil -> Google Books -> vectorización TF -> re-ranking coseno ->
// top 5 (K-Means) -> tabla 'recommendations' compartida.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	loadDotEnv(".env") // carga DATABASE_URL/PORT desde .env si existe

	// Modo servidor: por variable de entorno (más confiable en plataformas
	// como Railway que por un flag de línea de comandos) O por el flag -server
	// (para seguir usándolo fácil en tu compu).
	servidorFlag := flag.Bool("server", false, "correr como servidor HTTP en vez de CLI")
	flag.Parse()
	modoServidor := *servidorFlag || os.Getenv("RUN_MODE") == "server"

	dsn := getDSN()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Println("error conectando a postgres:", err)
		os.Exit(1)
	}
	defer pool.Close()

	if modoServidor {
		runServer(pool) // ver server.go
		return
	}

	runCLI(ctx, pool)
}

func getDSN() string {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://tinta:tinta_dev_pass@localhost:5432/tinta?sslmode=disable&search_path=recommendations,public"
	}
	return dsn
}

// runCLI conserva exactamente el comportamiento de siempre (para pruebas
// rápidas desde terminal, igual que antes).
func runCLI(ctx context.Context, pool *pgxpool.Pool) {
	args := flag.Args() // args posicionales, sin contar -server

	if len(args) < 2 {
		fmt.Println("uso: go run . <user_id> <archivo_libro.pdf> [pregunta1] [pregunta2] ...")
		fmt.Println("   o: go run . -server   (modo servidor HTTP)")
		os.Exit(1)
	}

	userID, err := uuid.Parse(args[0])
	if err != nil {
		fmt.Println("user_id inválido:", err)
		os.Exit(1)
	}

	rutaArchivo := args[1]
	questions := args[2:]

	bookText, err := leerLibro(rutaArchivo)
	if err != nil {
		fmt.Println("no se pudo leer el libro:", err)
		os.Exit(1)
	}

	n, err := Regenerate(ctx, pool, userID, bookText, questions)
	if err != nil {
		fmt.Println("error generando recomendaciones:", err)
		os.Exit(1)
	}

	fmt.Printf("listo: %d recomendaciones guardadas para %s\n", n, userID)
}

// leerLibro acepta .pdf (lo convierte a markdown) o .txt/.md (texto directo).
func leerLibro(ruta string) (string, error) {
	ext := strings.ToLower(filepath.Ext(ruta))

	switch ext {
	case ".pdf":
		md, err := pdfToMarkdown(ruta)
		if err != nil {
			return "", err
		}
		guardarMarkdownDebug(ruta, md)
		return md, nil

	case ".txt", ".md":
		bytes, err := os.ReadFile(ruta)
		if err != nil {
			return "", err
		}
		return string(bytes), nil

	default:
		return "", fmt.Errorf("formato no soportado (%s): usa .pdf, .txt o .md", ext)
	}
}

func guardarMarkdownDebug(rutaPDF, markdown string) {
	rutaMD := strings.TrimSuffix(rutaPDF, filepath.Ext(rutaPDF)) + "_extraido.md"
	_ = os.WriteFile(rutaMD, []byte(markdown), 0644)
}
