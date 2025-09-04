package main

import (
	"encoding/json"
	"equiposmedicos/middleware"
	"net/http"
	"strconv"
	"strings"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
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

// Handler para /api/articulos/{id} (GET)

func handleGetArticuloPorID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Extraer ID desde la URL
	idStr := strings.TrimPrefix(r.URL.Path, "/api/articulos/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"ID inválido"}`, http.StatusBadRequest)
		return
	}

	// Consultar el artículo por ID
	var articuloRaw []map[string]interface{}
	err = supabaseClient.DB.From("articulos").Select("*").Eq("id", strconv.Itoa(id)).Execute(&articuloRaw)
	if err != nil || len(articuloRaw) == 0 {
		http.Error(w, `{"error":"Artículo no encontrado"}`, http.StatusNotFound)
		return
	}

	// Obtener el nombre de la categoría
	nombreCategoria := "Sin Categoría"
	if catID, ok := articuloRaw[0]["categoria_id"].(float64); ok {
		var categoria []map[string]interface{}
		err = supabaseClient.DB.From("categorias").Select("nombre").Eq("id", strconv.Itoa(int(catID))).Execute(&categoria)
		if err == nil && len(categoria) > 0 {
			if nombre, ok := categoria[0]["nombre"].(string); ok {
				nombreCategoria = nombre
			}
		}
	}

	var articulo ArticleResponse
	articleBytes, _ := json.Marshal(articuloRaw[0])
	json.Unmarshal(articleBytes, &articulo)

	articulo.CategoriaNombre = nombreCategoria

	// Enviar JSON de respuesta
	resp, err := json.Marshal(articulo)
	if err != nil {
		http.Error(w, `{"error":"Error al generar JSON"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

// Handler para /api/articulos/agregar (POST)
func handleAgregarArticulo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var nuevo map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&nuevo)
	if err != nil {
		http.Error(w, `{"error":"JSON inválido: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Extraemos inventario inicial y usuario_id del payload si viene
	cantidadInicial, ok := nuevo["inventario"].(float64)
	if !ok {
		cantidadInicial = 0 // default
	}
	nombreUsuario, _ := nuevo["name"].(float64)
	delete(nuevo, "inventario")
	delete(nuevo, "name")

	// 1️⃣ Insertar artículo
	var results []map[string]interface{}
	err = supabaseClient.DB.From("articulos").Insert(nuevo).Execute(&results)
	if err != nil || len(results) == 0 {
		http.Error(w, `{"error":"Error al insertar artículo: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	articuloID := results[0]["id"].(float64) // id del artículo insertado

	// 2️⃣ Insertar inventario inicial
	inventario := map[string]interface{}{
		"articulo_id":     articuloID,
		"cantidad_actual": cantidadInicial,
	}
	err = supabaseClient.DB.From("inventarios").Insert(inventario).Execute(nil)
	if err != nil {
		http.Error(w, `{"error":"Error al insertar inventario: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// 3️⃣ Registrar movimiento inicial
	movimiento := map[string]interface{}{
		"articulo_id":     articuloID,
		"tipo_movimiento": "alta",
		"cantidad":        cantidadInicial,
		"motivo":          "Inventario inicial",
		"usuario_nombre":  nombreUsuario,
	}
	err = supabaseClient.DB.From("movimientos_inventario").Insert(movimiento).Execute(nil)
	if err != nil {
		http.Error(w, `{"error":"Error al registrar movimiento: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Responder con el artículo creado
	json.NewEncoder(w).Encode(results[0])
}

// Handler para /api/articulos/actualizar (PUT)
func handleActualizarArticulo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("update") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/articulos/actualizar/")
	var id int
	var err error
	id, err = strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"ID inválido"}`, http.StatusBadRequest)
		return
	}

	var datos map[string]interface{}
	err = json.NewDecoder(r.Body).Decode(&datos)
	if err != nil {
		http.Error(w, `{"error":"JSON inválido: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	var results []map[string]interface{} // Usamos directamente slice de mapas
	err = supabaseClient.DB.From("articulos").Update(datos).Eq("id", strconv.Itoa(id)).Execute(&results)
	if err != nil {
		http.Error(w, `{"error":"Error al actualizar: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if len(results) > 0 {
		json.NewEncoder(w).Encode(results[0])
	} else {
		http.Error(w, `{"message":"Actualización exitosa, pero no se recibieron datos de respuesta."}`, http.StatusOK)
	}
}

// Handler para /api/articulos/eliminar (DELETE)
func handleEliminarArticulo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("delete") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/articulos/eliminar/")
	var id int
	var err error
	id, err = strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"ID inválido"}`, http.StatusBadRequest)
		return
	}

	var results []map[string]interface{} // Usamos directamente slice de mapas
	err = supabaseClient.DB.From("articulos").Delete().Eq("id", strconv.Itoa(id)).Execute(&results)
	if err != nil {
		http.Error(w, `{"error":"Error al eliminar: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if len(results) > 0 {
		json.NewEncoder(w).Encode(results[0])
	} else {
		http.Error(w, `{"message":"Eliminación exitosa, pero no se recibieron datos de respuesta."}`, http.StatusOK)
	}
}

func handleBuscarArticulos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	termino := strings.ToLower(r.URL.Query().Get("busqueda"))
	if termino == "" {
		http.Error(w, `{"error":"Parámetro 'busqueda' obligatorio"}`, http.StatusBadRequest)
		return
	}

	var articlesRaw []map[string]interface{}
	err := supabaseClient.DB.From("articulos").Select("*").Execute(&articlesRaw)
	if err != nil {
		http.Error(w, `{"error":"Error al obtener artículos: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Obtener categorías para mapear id->nombre
	var categoriesRaw []map[string]interface{}
	err = supabaseClient.DB.From("categorias").Select("id,nombre").Execute(&categoriesRaw)
	if err != nil {
		http.Error(w, `{"error":"Error al obtener categorías: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	categoryMap := make(map[int]string)
	for _, cat := range categoriesRaw {
		if id, ok := cat["id"].(float64); ok {
			if name, ok := cat["nombre"].(string); ok {
				categoryMap[int(id)] = name
			}
		}
	}

	// Filtrar manualmente por nombre, proveedor o categoria_nombre
	var filtered []ArticleResponse
	for _, art := range articlesRaw {
		nombre := strings.ToLower(art["nombre"].(string))
		proveedor, _ := art["proveedor"].(string)
		proveedor = strings.ToLower(proveedor)
		categoriaNombre := ""
		if catIDf, ok := art["categoria_id"].(float64); ok {
			categoriaNombre = strings.ToLower(categoryMap[int(catIDf)])
		}

		if strings.Contains(nombre, termino) ||
			strings.Contains(proveedor, termino) ||
			strings.Contains(categoriaNombre, termino) {

			var article ArticleResponse
			bytesArt, _ := json.Marshal(art)
			json.Unmarshal(bytesArt, &article)
			if catIDf, ok := art["categoria_id"].(float64); ok {
				article.CategoriaNombre = categoryMap[int(catIDf)]
			} else {
				article.CategoriaNombre = "Sin Categoría"
			}
			filtered = append(filtered, article)
		}
	}

	jsonResp, err := json.Marshal(filtered)
	if err != nil {
		http.Error(w, `{"error":"Error al convertir a JSON: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResp)
}
