package main

import (
	"encoding/json"
	"equiposmedicos/middleware"
	"net/http"
	"strconv"
	"strings"
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

// Crear toma de inventario físico
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

	// 2️⃣ Validar permisos del usuario
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

	// 4️⃣ Obtener artículos
	var articulos []map[string]interface{}
	err = supabaseClient.DB.From("articulos").Select("*").Execute(&articulos)
	if err != nil {
		http.Error(w, `{"error":"Error al obtener artículos: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// 5️⃣ Obtener inventarios
	var inventarios []map[string]interface{}
	err = supabaseClient.DB.From("inventarios").Select("*").Execute(&inventarios)
	if err != nil {
		http.Error(w, `{"error":"Error al obtener inventarios: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// 6️⃣ Crear un mapa de cantidad actual por articulo_id
	cantidadMap := map[int]float64{}
	for _, inv := range inventarios {
		if idFloat, ok := inv["articulo_id"].(float64); ok {
			id := int(idFloat)
			if cant, ok := inv["cantidad_actual"].(float64); ok {
				cantidadMap[id] = cant
			}
		}
	}

	// 7️⃣ Crear los detalles de inventario
	detalles := []map[string]interface{}{}
	for _, a := range articulos {
		// Verificar categoría
		categoriaIDArt := 0
		if a["categoria_id"] != nil {
			if floatVal, ok := a["categoria_id"].(float64); ok {
				categoriaIDArt = int(floatVal)
			}
		}
		if payload.CategoriaID != nil && categoriaIDArt != *payload.CategoriaID {
			continue
		}

		// ID del artículo
		articuloID := 0
		if a["id"] != nil {
			if floatVal, ok := a["id"].(float64); ok {
				articuloID = int(floatVal)
			}
		}

		// Cantidad teórica desde inventarios
		cantidadTeorica := 0.0
		if cant, ok := cantidadMap[articuloID]; ok {
			cantidadTeorica = cant
		}

		detalles = append(detalles, map[string]interface{}{
			"toma_id":          tomaID,
			"articulo_id":      articuloID,
			"cantidad_teorica": cantidadTeorica,
			"cantidad_real":    0,
		})
	}

	// 8️⃣ Insertar detalles en la base de datos
	if len(detalles) > 0 {
		err = supabaseClient.DB.From("tomafisicadetalle").Insert(detalles).Execute(nil)
		if err != nil {
			http.Error(w, `{"error":"Error al crear detalles de inventario: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	// 9️⃣ Retornar JSON con la toma creada
	json.NewEncoder(w).Encode(map[string]interface{}{
		"toma_id": tomaID,
		"folio":   folio,
	})
}

func handleObtenerDetalleToma(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("read") {
		http.Error(w, `{"message":"Permiso denegado"}`, http.StatusForbidden)
		return
	}

	// Extraer id desde la URL (/api/inventarios/detalles_tomas/{id})
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, `{"error":"ID no proporcionado"}`, http.StatusBadRequest)
		return
	}
	idStr := parts[4]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"ID inválido"}`, http.StatusBadRequest)
		return
	}

	idValue := strconv.Itoa(id)

	var detalles []map[string]interface{}
	err = supabaseClient.DB.
		From("tomafisica_view").
		Select("*").
		Eq("toma_id", idValue).
		Execute(&detalles)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if detalles == nil {
		detalles = []map[string]interface{}{}
	}

	json.NewEncoder(w).Encode(detalles)
}
