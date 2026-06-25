// Servicio de recomendaciones de Tinta (standalone, Go).
//
// Sigue el flujo: libro + preguntas -> perfil -> Google Books ->
// vectorización TF -> re-ranking coseno -> top 5 (K-Means) -> tabla.
// NO toca el repo de Diego; escribe en la tabla 'recommendations' compartida.
//
// Uso:
//   go run . <user_id> <archivo_libro.txt> [pregunta1] [pregunta2] ...
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	loadDotEnv(".env") // carga DATABASE_URL desde .env si existe

	if len(os.Args) < 3 {
		fmt.Println("uso: go run . <user_id> <archivo_libro.txt> [pregunta1] [pregunta2] ...")
		os.Exit(1)
	}

	userID, err := uuid.Parse(os.Args[1])
	if err != nil {
		fmt.Println("user_id inválido:", err)
		os.Exit(1)
	}

	bookBytes, err := os.ReadFile(os.Args[2])
	if err != nil {
		fmt.Println("no se pudo leer el archivo del libro:", err)
		os.Exit(1)
	}
	bookText := string(bookBytes)
	questions := os.Args[3:]

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://tinta:tinta_dev_pass@localhost:5432/tinta?sslmode=disable&search_path=recommendations"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Println("error conectando a postgres:", err)
		os.Exit(1)
	}
	defer pool.Close()

	n, err := Regenerate(ctx, pool, userID, bookText, questions)
	if err != nil {
		fmt.Println("error generando recomendaciones:", err)
		os.Exit(1)
	}

	fmt.Printf("listo: %d recomendaciones guardadas para %s\n", n, userID)
}
