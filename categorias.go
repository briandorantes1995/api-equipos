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

// Handler para /api/categorias (GET)
func handleGetCategorias(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var result []map[string]interface{}
	err := supabaseClient.DB.From("categorias").Select("*").Execute(&result)
	if err != nil {
		http.Error(w, `{"error":"Error de conexión o tabla no existe: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(result)
}

// Handler para /api/categorias/agregar (POST)
func handleAgregarCategoria(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var nueva map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&nueva)
	if err != nil {
		http.Error(w, `{"error":"JSON inválido: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	var results []map[string]interface{} // Usamos directamente slice de mapas
	err = supabaseClient.DB.From("categorias").Insert(nueva).Execute(&results)
	if err != nil {
		http.Error(w, `{"error":"Error al insertar categoría: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if len(results) > 0 {
		json.NewEncoder(w).Encode(results[0])
	} else {
		http.Error(w, `{"message":"Inserción exitosa, pero no se recibieron datos de respuesta."}`, http.StatusOK)
	}
}

// Handler para /api/categorias/actualizar (PUT)
func handleActualizarCategoria(w http.ResponseWriter, r *http.Request) {
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

	idStr := strings.TrimPrefix(r.URL.Path, "/api/categorias/actualizar/")
	var id int
	var err error // Declarar err aquí
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
	err = supabaseClient.DB.From("categorias").Update(datos).Eq("id", strconv.Itoa(id)).Execute(&results)
	if err != nil {
		http.Error(w, `{"error":"Error al actualizar en Supabase: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if len(results) > 0 {
		json.NewEncoder(w).Encode(results[0])
	} else {
		http.Error(w, `{"message":"Actualización exitosa, pero no se recibieron datos de respuesta."}`, http.StatusOK)
	}
}

// Handler para /api/categorias/eliminar (DELETE)
func handleEliminarCategoria(w http.ResponseWriter, r *http.Request) {
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

	idStr := strings.TrimPrefix(r.URL.Path, "/api/categorias/eliminar/")
	var id int
	var err error // Declarar err aquí
	id, err = strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"ID inválido"}`, http.StatusBadRequest)
		return
	}

	var results []map[string]interface{} // Usamos directamente slice de mapas
	err = supabaseClient.DB.From("categorias").Delete().Eq("id", strconv.Itoa(id)).Execute(&results)
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
