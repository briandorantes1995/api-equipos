package main

import (
	"encoding/json"
	"equiposmedicos/middleware"
	"net/http"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
)

func handleReporteInventario(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Validación de token y permisos
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("read") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	var inventarios []InventarioArticulo

	err := supabaseClient.DB.
		From("inventario_view").
		Select("*").
		Execute(&inventarios)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(inventarios)
}

func handleObtenerInventarios(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Validación de token y permisos
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("read") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	var inventarios []TomaInventario

	err := supabaseClient.DB.
		From("tomafisica_view").
		Select("*").
		Execute(&inventarios)

	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Retornar JSON
	if inventarios == nil {
		inventarios = []TomaInventario{}
	}

	json.NewEncoder(w).Encode(inventarios)
}

// Handler para /api/inventario/crear_toma (POST)
func handleCrearTomaFisica(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// 1️⃣ Decodificar payload
	var payload struct {
		CategoriaID *int `json:"categoria_id,omitempty"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, `{"error":"JSON inválido: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// 2️⃣ Obtener claims del usuario autenticado
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)

	if !claims.HasPermission("create") {
		http.Error(w, `{"message":"Permiso denegado"}`, http.StatusForbidden)
		return
	}

	usuarioSub := claims.Subject
	usuarioCorreo := claims.Email

	// 3️⃣ Insertar la toma física
	toma := map[string]interface{}{
		"fecha_inicio":      time.Now(),
		"estado":            "abierta",
		"categoria_id":      payload.CategoriaID,
		"usuario_auth0_sub": usuarioSub,
		"usuario_correo":    usuarioCorreo,
	}

	var results []map[string]interface{}
	err = supabaseClient.DB.From("tomafisica").Insert(toma).Execute(&results)
	if err != nil || len(results) == 0 {
		http.Error(w, `{"error":"Error al crear la toma: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	tomaID := results[0]["id"].(float64)
	folio := results[0]["folio"].(float64)

	// 4️⃣ Insertar los detalles de inventario
	// Obtener artículos según categoría o todos
	var articulos []map[string]interface{}
	err = supabaseClient.DB.
		From("inventarios").
		Select("*").
		Execute(&articulos)
	if err != nil {
		http.Error(w, `{"error":"Error al obtener inventarios: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	detalles := []map[string]interface{}{}
	for _, a := range articulos {
		categoriaIDArtFloat, ok := a["categoria_id"].(float64)
		if !ok {
			continue // si no es numérico, saltar
		}
		categoriaIDArt := int(categoriaIDArtFloat)

		// Comparar con la categoría seleccionada
		if payload.CategoriaID != nil && categoriaIDArt != *payload.CategoriaID {
			continue
		}

		detalles = append(detalles, map[string]interface{}{
			"toma_id":          tomaID,
			"articulo_id":      a["articulo_id"],
			"cantidad_teorica": a["cantidad_actual"],
			"cantidad_real":    0,
		})
	}

	if len(detalles) > 0 {
		err = supabaseClient.DB.From("tomafisicadetalle").Insert(detalles).Execute(nil)
		if err != nil {
			http.Error(w, `{"error":"Error al crear detalles de inventario: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	// 5️⃣ Retornar JSON con toma creada
	json.NewEncoder(w).Encode(map[string]interface{}{
		"toma_id": tomaID,
		"folio":   folio,
	})
}
