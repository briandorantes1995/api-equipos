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

	//Decodificar payload
	var payload struct {
		CategoriaID *int `json:"categoria_id,omitempty"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, `{"error":"JSON inválido: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	//Validar permisos del usuario
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("create") {
		http.Error(w, `{"message":"Permiso denegado"}`, http.StatusForbidden)
		return
	}

	usuarioSub := claims.Subject
	usuarioCorreo := claims.Email

	//Insertar la toma física
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

	//Obtener artículos
	var articulos []map[string]interface{}
	err = supabaseClient.DB.From("articulos").Select("*").Execute(&articulos)
	if err != nil {
		http.Error(w, `{"error":"Error al obtener artículos: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	//Obtener inventarios
	var inventarios []map[string]interface{}
	err = supabaseClient.DB.From("inventarios").Select("*").Execute(&inventarios)
	if err != nil {
		http.Error(w, `{"error":"Error al obtener inventarios: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	//Crear un mapa de cantidad actual por articulo_id
	cantidadMap := map[int]float64{}
	for _, inv := range inventarios {
		if idFloat, ok := inv["articulo_id"].(float64); ok {
			id := int(idFloat)
			if cant, ok := inv["cantidad_actual"].(float64); ok {
				cantidadMap[id] = cant
			}
		}
	}

	//Crear los detalles de inventario
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

	//Insertar detalles en la base de datos
	if len(detalles) > 0 {
		err = supabaseClient.DB.From("tomafisicadetalle").Insert(detalles).Execute(nil)
		if err != nil {
			http.Error(w, `{"error":"Error al crear detalles de inventario: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	//Retornar JSON con la toma creada
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
		From("tomafisicadetalle_view").
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

// Handler para guardar los detalles de una toma física (conteo real)
func handleGuardarTomaDetalle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Validación de token y permisos
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("update") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	// Decodificar payload
	var detalles []map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&detalles); err != nil {
		http.Error(w, `{"error":"Error al decodificar JSON: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	for _, d := range detalles {
		detalleID, ok := d["detalle_id"].(float64)
		if !ok {
			continue
		}

		updates := map[string]interface{}{
			"cantidad_real": d["cantidad_real"],
		}

		err := supabaseClient.DB.
			From("tomafisicadetalle").
			Update(updates).
			Eq("id", strconv.Itoa(int(detalleID))).
			Execute(nil)

		if err != nil {
			http.Error(w, `{"error":"Error al actualizar detalle: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Detalles de toma actualizados correctamente",
	})
}

// Handler para cancelar (eliminar) una toma física completa
func handleCancelarToma(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Validación de token y permisos
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("delete") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	// Obtener ID desde URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, `{"error":"ID no proporcionado"}`, http.StatusBadRequest)
		return
	}

	idStr := parts[4]

	// Eliminar toma física
	err := supabaseClient.DB.
		From("tomafisica").
		Delete().
		Eq("id", idStr).
		Execute(nil)

	if err != nil {
		http.Error(w, `{"error":"Error al eliminar toma: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Toma física cancelada correctamente",
	})
}

// handleFinalizarToma actualiza los detalles y cierra el folio de la toma física.
func handleFinalizarToma(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	// Validacion de token y permisos
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("update") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	// Decodificar el payload
	var detalles []map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&detalles); err != nil {
		http.Error(w, `{"error":"Error al decodificar JSON: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	if len(detalles) == 0 {
		json.NewEncoder(w).Encode(map[string]string{"message": "No se proporcionaron detalles para finalizar."})
		return
	}

	for _, d := range detalles {
		detalleID, ok := d["detalle_id"].(float64)
		if !ok {
			continue
		}
		updates := map[string]interface{}{"cantidad_real": d["cantidad_real"]}
		err := supabaseClient.DB.From("tomafisicadetalle").Update(updates).Eq("id", strconv.Itoa(int(detalleID))).Execute(nil)
		if err != nil {
			http.Error(w, `{"error":"Error al actualizar detalle `+strconv.Itoa(int(detalleID))+`: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	firstDetalleID := int(detalles[0]["detalle_id"].(float64))

	var tomaDetalles []struct {
		TomaID int `json:"toma_id"`
	}

	err := supabaseClient.DB.
		From("tomafisicadetalle").
		Select("toma_id").
		Limit(1).
		Eq("id", strconv.Itoa(firstDetalleID)).
		Execute(&tomaDetalles)

	if err != nil {
		http.Error(w, `{"error":"Error al buscar el folio de la toma: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if len(tomaDetalles) == 0 {
		http.Error(w, `{"error":"No se pudo encontrar el folio de la toma para el detalle `+strconv.Itoa(firstDetalleID)+`"}`, http.StatusNotFound)
		return
	}

	tomaID := tomaDetalles[0].TomaID

	// Cambia el estado y actualiza la fecha de finalización
	updates := map[string]interface{}{
		"estado":    "cerrada",
		"fecha_fin": time.Now(),
	}
	err = supabaseClient.DB.From("tomafisica").Update(updates).Eq("id", strconv.Itoa(tomaID)).Execute(nil)
	if err != nil {
		http.Error(w, `{"error":"Error al cerrar el folio de la toma: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Toma física finalizada y cerrada correctamente",
	})
}
