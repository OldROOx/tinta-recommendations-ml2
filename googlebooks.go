package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

const googleBooksURL = "https://www.googleapis.com/books/v1/volumes"

// candidate es un libro devuelto por Google Books.
type candidate struct {
	VolumeID    string
	Title       string
	Authors     []string
	Description string
	Thumbnail   string
	InfoLink    string
}

// gbResponse mapea solo los campos que usamos de la respuesta de Google Books.
type gbResponse struct {
	Items []struct {
		ID         string `json:"id"`
		VolumeInfo struct {
			Title       string   `json:"title"`
			Authors     []string `json:"authors"`
			Description string   `json:"description"`
			InfoLink    string   `json:"infoLink"`
			ImageLinks  struct {
				Thumbnail string `json:"thumbnail"`
			} `json:"imageLinks"`
		} `json:"volumeInfo"`
	} `json:"items"`
}

var htmlTag = regexp.MustCompile(`<[^>]+>`)

// searchGoogleBooks consulta la API y regresa los candidatos.
func searchGoogleBooks(ctx context.Context, client *http.Client, query string, maxResults int) ([]candidate, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("langRestrict", "es")
	params.Set("printType", "books")
	params.Set("orderBy", "relevance")
	params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	if key := os.Getenv("GOOGLE_BOOKS_API_KEY"); key != "" {
		params.Set("key", key)
	}

	reqURL := googleBooksURL + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google books status %d", resp.StatusCode)
	}

	var parsed gbResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	out := make([]candidate, 0, len(parsed.Items))
	for _, it := range parsed.Items {
		desc := strings.TrimSpace(htmlTag.ReplaceAllString(it.VolumeInfo.Description, ""))
		title := it.VolumeInfo.Title
		if strings.TrimSpace(title) == "" {
			title = "Sin título"
		}
		out = append(out, candidate{
			VolumeID:    it.ID,
			Title:       title,
			Authors:     it.VolumeInfo.Authors,
			Description: desc,
			Thumbnail:   it.VolumeInfo.ImageLinks.Thumbnail,
			InfoLink:    it.VolumeInfo.InfoLink,
		})
	}
	return out, nil
}
