package main

import (
	"encoding/json"
	"equiposmedicos/middleware"
	"log"
	"net/http"
	"strconv"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
)

// Handler para registrar movimientos y actualizar inventario
func handleRegistrarMovimiento(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Validación de token y permisos
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("create") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	// Decodificar el movimiento
	var payload struct {
		ArticuloID     int     `json:"articulo_id"`
		TipoMovimiento string  `json:"tipo_movimiento"`
		Cantidad       float64 `json:"cantidad"`
		Motivo         string  `json:"motivo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"Error al decodificar JSON: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Validar campos esenciales
	if payload.ArticuloID <= 0 || payload.TipoMovimiento == "" || payload.Cantidad <= 0 {
		http.Error(w, `{"error":"Datos del movimiento inválidos"}`, http.StatusBadRequest)
		return
	}

	// Preparar movimiento como map[string]interface{} sin fecha
	movimiento := map[string]interface{}{
		"articulo_id":     payload.ArticuloID,
		"tipo_movimiento": payload.TipoMovimiento,
		"cantidad":        payload.Cantidad,
		"motivo":          payload.Motivo,
		// no incluir "fecha" para que PostgreSQL use DEFAULT now()
	}

	// Insertar movimiento
	if err := supabaseClient.DB.
		From("movimientos_inventario").
		Insert(movimiento).
		Execute(nil); err != nil {
		http.Error(w, `{"error":"Error al insertar movimiento: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Obtener inventario actual
	var inventarios []InventarioArticulo
	err := supabaseClient.DB.
		From("inventarios").
		Select("*").
		Eq("articulo_id", strconv.Itoa(payload.ArticuloID)).
		Execute(&inventarios)
	if err != nil || len(inventarios) == 0 {
		http.Error(w, `{"error":"Inventario no encontrado"}`, http.StatusInternalServerError)
		return
	}

	cantidadActual := inventarios[0].CantidadActual

	// Ajustar inventario según tipo
	switch payload.TipoMovimiento {
	case "compra", "transferencia_entrada":
		cantidadActual += payload.Cantidad
	case "venta", "baja", "robo", "transferencia_salida":
		cantidadActual -= payload.Cantidad
	default:
		http.Error(w, `{"error":"Tipo de movimiento desconocido o no permitido"}`, http.StatusBadRequest)
		return
	}

	// Actualizar inventario con Update (sin fecha)
	update := map[string]interface{}{
		"cantidad_actual": cantidadActual,
	}

	if err := supabaseClient.DB.
		From("inventarios").
		Update(update).
		Eq("articulo_id", strconv.Itoa(payload.ArticuloID)).
		Execute(nil); err != nil {
		http.Error(w, `{"error":"Error al actualizar inventario: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Respuesta
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":         "Movimiento registrado y inventario actualizado",
		"articulo_id":     payload.ArticuloID,
		"cantidad_actual": cantidadActual,
	})
}

// Handler para obtener el reporte de movimientos con nombres de artículos
func handleReporteMovimientos(w http.ResponseWriter, r *http.Request) {
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

	var movimientos []MovimientoConNombre

	// Traemos todos los movimientos desde la view
	err := supabaseClient.DB.
		From("movimientos_con_nombre").
		Select("*").
		Execute(&movimientos)

	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(movimientos)
}

// Handler para editar la cantidad de un movimiento existente
func handleEditarMovimiento(w http.ResponseWriter, r *http.Request) {
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
	var payload MovimientoEditar
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"Error al decodificar JSON: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Obtener movimiento original
	var movimientos []Movimiento
	err := supabaseClient.DB.
		From("movimientos_inventario").
		Select("*").
		Eq("id", strconv.Itoa(payload.ID)).
		Execute(&movimientos)
	if err != nil || len(movimientos) == 0 {
		http.Error(w, `{"error":"No se encontró el movimiento original"}`, http.StatusNotFound)
		return
	}
	original := movimientos[0]

	// Obtener inventario actual
	var inventarios []InventarioMovimientoArticulo
	err = supabaseClient.DB.
		From("inventarios").
		Select("*").
		Eq("articulo_id", strconv.Itoa(original.ArticuloID)).
		Execute(&inventarios)
	if err != nil {
		log.Printf("Error al consultar inventario: %v\n", err)
		http.Error(w, `{"error":"Error al consultar inventario: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if len(inventarios) == 0 {
		log.Printf("Inventario no encontrado para articulo_id %d\n", original.ArticuloID)
		http.Error(w, `{"error":"Inventario no encontrado"}`, http.StatusInternalServerError)
		return
	}

	cantidadActual := inventarios[0].CantidadActual

	delta := payload.Cantidad - original.Cantidad

	switch original.TipoMovimiento {
	case "alta", "compra", "transferencia_entrada":
		cantidadActual += delta
	case "venta", "baja", "robo", "transferencia_salida":
		cantidadActual -= delta
	default:
		http.Error(w, `{"error":"Tipo de movimiento desconocido"}`, http.StatusBadRequest)
		return
	}

	if err := supabaseClient.DB.
		From("inventarios").
		Update(map[string]interface{}{
			"cantidad_actual":      cantidadActual,
			"ultima_actualizacion": time.Now(),
		}).
		Eq("articulo_id", strconv.Itoa(original.ArticuloID)).
		Execute(nil); err != nil {
		http.Error(w, `{"error":"Error al actualizar inventario: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Actualizar cantidad en el movimiento original
	updates := map[string]interface{}{
		"cantidad": payload.Cantidad,
		"fecha":    time.Now(),
	}
	if err := supabaseClient.DB.
		From("movimientos_inventario").
		Update(updates).
		Eq("id", strconv.Itoa(payload.ID)).
		Execute(nil); err != nil {
		http.Error(w, `{"error":"Error al actualizar cantidad del movimiento: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Respuesta
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":         "Inventario y movimiento actualizados",
		"articulo_id":     original.ArticuloID,
		"cantidad_actual": cantidadActual,
	})
}

// Handler para eliminar un movimiento existente y ajustar inventario
func handleEliminarMovimiento(w http.ResponseWriter, r *http.Request) {
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

	// Decodificar payload con ID
	var payload struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"Error al decodificar JSON: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Obtener movimiento original
	var movimientos []Movimiento
	err := supabaseClient.DB.
		From("movimientos_inventario").
		Select("*").
		Eq("id", strconv.Itoa(payload.ID)).
		Execute(&movimientos)
	if err != nil || len(movimientos) == 0 {
		http.Error(w, `{"error":"No se encontró el movimiento original"}`, http.StatusNotFound)
		return
	}
	original := movimientos[0]

	// Obtener inventario actual
	var inventarios []InventarioMovimientoArticulo
	err = supabaseClient.DB.
		From("inventarios").
		Select("*").
		Eq("articulo_id", strconv.Itoa(original.ArticuloID)).
		Execute(&inventarios)
	if err != nil || len(inventarios) == 0 {
		http.Error(w, `{"error":"No se pudo obtener inventario"}`, http.StatusInternalServerError)
		return
	}
	cantidadActual := inventarios[0].CantidadActual

	switch original.TipoMovimiento {
	case "compra", "transferencia_entrada":
		cantidadActual -= original.Cantidad
	case "venta", "baja", "robo", "transferencia_salida":
		cantidadActual += original.Cantidad
	default:
		http.Error(w, `{"error":"Tipo de movimiento desconocido"}`, http.StatusBadRequest)
		return
	}

	update := map[string]interface{}{
		"cantidad_actual":      cantidadActual,
		"ultima_actualizacion": time.Now(),
	}
	if err := supabaseClient.DB.
		From("inventarios").
		Update(update).
		Eq("articulo_id", strconv.Itoa(original.ArticuloID)).
		Execute(nil); err != nil {
		http.Error(w, `{"error":"Error al actualizar inventario: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Eliminar movimiento
	if err := supabaseClient.DB.
		From("movimientos_inventario").
		Delete().
		Eq("id", strconv.Itoa(payload.ID)).
		Execute(nil); err != nil {
		http.Error(w, `{"error":"Error al eliminar movimiento: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Respuesta
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":         "Movimiento eliminado y inventario ajustado",
		"articulo_id":     original.ArticuloID,
		"cantidad_actual": cantidadActual,
	})
}
