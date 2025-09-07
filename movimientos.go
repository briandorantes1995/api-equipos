package main

import (
	"encoding/json"
	"equiposmedicos/middleware"
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
	if !claims.HasPermission("write") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	// Decodificar el movimiento
	var movimiento Movimiento
	if err := json.NewDecoder(r.Body).Decode(&movimiento); err != nil {
		http.Error(w, `{"error":"Error al decodificar JSON: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Validar campos
	if movimiento.ArticuloID <= 0 || movimiento.TipoMovimiento == "" || movimiento.Cantidad <= 0 {
		http.Error(w, `{"error":"Datos del movimiento inválidos"}`, http.StatusBadRequest)
		return
	}

	// Registrar movimiento
	err := supabaseClient.DB.
		From("movimientos_inventario").
		Insert(movimiento).
		Execute(nil)
	if err != nil {
		http.Error(w, `{"error":"Error al insertar movimiento: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Obtener inventario actual
	var inventarios []InventarioArticulo
	err = supabaseClient.DB.
		From("inventarios").
		Select("*").
		Eq("articulo_id", strconv.Itoa(movimiento.ArticuloID)).
		Execute(&inventarios)

	if err != nil {
		http.Error(w, `{"error":"Error al obtener inventario: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	cantidadActual := 0.0
	if len(inventarios) > 0 {
		cantidadActual = inventarios[0].CantidadActual
	}

	// Actualizar inventario según tipo
	switch movimiento.TipoMovimiento {
	case "alta", "compra", "transferencia_entrada":
		cantidadActual += movimiento.Cantidad
	case "venta", "baja", "robo", "transferencia_salida":
		cantidadActual -= movimiento.Cantidad
	default:
		http.Error(w, `{"error":"Tipo de movimiento desconocido"}`, http.StatusBadRequest)
		return
	}

	// Upsert en inventario
	upsert := map[string]interface{}{
		"articulo_id":          movimiento.ArticuloID,
		"cantidad_actual":      cantidadActual,
		"ultima_actualizacion": time.Now(),
	}

	// Solo un valor de retorno, no dos
	err = supabaseClient.DB.
		From("inventarios").
		Upsert(upsert).
		Execute(nil)
	if err != nil {
		http.Error(w, `{"error":"Error al actualizar inventario: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Respuesta
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":         "Movimiento registrado y inventario actualizado",
		"articulo_id":     movimiento.ArticuloID,
		"cantidad_actual": cantidadActual,
	})
}

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

// Handler para editar un movimiento existente y ajustar inventario
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

	var payload MovimientoEditar
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"Error al decodificar JSON: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	var movimientos []Movimiento
	err := supabaseClient.DB.
		From("movimientos_inventario").
		Select("*").
		Eq("id", strconv.Itoa(payload.ID)).
		Execute(&movimientos)
	if err != nil || len(movimientos) == 0 {
		http.Error(w, `{"error":"No se encontró el movimiento original: `+err.Error()+`"}`, http.StatusNotFound)
		return
	}
	original := movimientos[0]

	if original.TipoMovimiento == "alta" && payload.TipoMovimiento != "alta" {
		http.Error(w, `{"error":"No se permite cambiar el tipo de un movimiento de alta"}`, http.StatusBadRequest)
		return
	}

	var inventarios []InventarioArticulo
	err = supabaseClient.DB.
		From("inventarios").
		Select("*").
		Eq("articulo_id", strconv.Itoa(original.ArticuloID)).
		Execute(&inventarios)
	if err != nil || len(inventarios) == 0 {
		http.Error(w, `{"error":"No se encontró el movimiento original: `+err.Error()+`"}`, http.StatusNotFound)
		return
	}
	cantidadActual := inventarios[0].CantidadActual

	// Anular efecto del movimiento original
	switch original.TipoMovimiento {
	case "alta", "compra", "transferencia_entrada":
		cantidadActual -= original.Cantidad
	case "venta", "baja", "robo", "transferencia_salida":
		cantidadActual += original.Cantidad
	case "ajuste_inventario":
		// Para ajuste, restamos la diferencia registrada
		cantidadActual -= original.Cantidad
	}

	// Aplicar nuevo movimiento
	if payload.TipoMovimiento == "ajuste_inventario" {
		// Ajuste absoluto: inventario igual a nueva cantidad
		diferencia := payload.Cantidad - cantidadActual
		cantidadActual += diferencia
		payload.Cantidad = diferencia // registrar diferencia como movimiento
	} else {
		switch payload.TipoMovimiento {
		case "alta", "compra", "transferencia_entrada":
			cantidadActual += payload.Cantidad
		case "venta", "baja", "robo", "transferencia_salida":
			cantidadActual -= payload.Cantidad
		}
	}

	// Actualizar inventario
	upsert := map[string]interface{}{
		"articulo_id":          original.ArticuloID,
		"cantidad_actual":      cantidadActual,
		"ultima_actualizacion": time.Now(),
	}
	if err := supabaseClient.DB.From("inventarios").Upsert(upsert).Execute(nil); err != nil {
		http.Error(w, `{"error":"Error al actualizar inventario: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Actualizar movimiento original
	updates := map[string]interface{}{
		"tipo_movimiento": payload.TipoMovimiento,
		"cantidad":        payload.Cantidad,
		"motivo":          payload.Motivo,
		"fecha":           time.Now(),
	}
	if err := supabaseClient.DB.From("movimientos_inventario").
		Update(updates).
		Eq("id", strconv.Itoa(payload.ID)).
		Execute(nil); err != nil {
		http.Error(w, `{"error":"Error al actualizar movimiento: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Respuesta
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":         "Movimiento editado y inventario actualizado",
		"articulo_id":     original.ArticuloID,
		"cantidad_actual": cantidadActual,
	})
}
