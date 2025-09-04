package main

import (
	"encoding/json"
	"net/http"
)

// Handler para /api/articulos (GET)
func handleGetArticulos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	// 1. Obtener todos los artículos
	var articlesRaw []map[string]interface{}
	err := supabaseClient.DB.From("articulos").Select("*").Execute(&articlesRaw)
	if err != nil {
		http.Error(w, `{"error":"Error al obtener artículos: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// 2. Obtener todas las categorías
	var categoriesRaw []map[string]interface{}
	err = supabaseClient.DB.From("categorias").Select("id,nombre").Execute(&categoriesRaw)
	if err != nil {
		http.Error(w, `{"error":"Error al obtener categorías: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// 3. Crear un mapa para buscar nombres de categorías por ID
	categoryMap := make(map[int]string)
	for _, cat := range categoriesRaw {
		if id, ok := cat["id"].(float64); ok { // Supabase devuelve números como float64
			if name, ok := cat["nombre"].(string); ok {
				categoryMap[int(id)] = name
			}
		}
	}

	// 4. Transformar los artículos a ArticleResponse y añadir el nombre de la categoría
	var articlesResponse []ArticleResponse
	for _, articleRaw := range articlesRaw {
		var article ArticleResponse
		// Marshal y Unmarshal para convertir map[string]interface{} a ArticleResponse
		// Esto es un truco para mapear los campos automáticamente
		articleBytes, _ := json.Marshal(articleRaw)
		json.Unmarshal(articleBytes, &article)

		// Añadir el nombre de la categoría
		if catID, ok := articleRaw["categoria_id"].(float64); ok { // Supabase devuelve IDs como float64
			if categoryName, found := categoryMap[int(catID)]; found {
				article.CategoriaNombre = categoryName
			} else {
				article.CategoriaNombre = "Categoría Desconocida" // O un valor por defecto
			}
		} else {
			article.CategoriaNombre = "Sin Categoría" // Si categoria_id no es un número
		}
		articlesResponse = append(articlesResponse, article)
	}

	jsonResp, err := json.Marshal(articlesResponse)
	if err != nil {
		http.Error(w, `{"error":"Error al convertir resultado a JSON"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResp)
}
