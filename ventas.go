package main

import (
	"encoding/json"
	"equiposmedicos/middleware"
	"net/http"
	"strconv"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
)

func handleRegistrarVenta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	// Validar permisos
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("create") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	// Parsear payload
	var payload VentaPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"JSON inválido: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	if len(payload.Articulos) == 0 {
		http.Error(w, `{"error":"Debe enviar al menos un artículo"}`, http.StatusBadRequest)
		return
	}

	// Calcular total
	total := 0.0
	for _, item := range payload.Articulos {
		total += item.PrecioUnitario * float64(item.Cantidad)
	}

	// Insertar venta principal
	venta := map[string]interface{}{
		"cliente_nombre":       payload.ClienteNombre,
		"cliente_razon_social": payload.ClienteRazonSocial,
		"cliente_direccion":    payload.ClienteDireccion,
		"cliente_telefono":     payload.ClienteTelefono,
		"cliente_correo":       payload.ClienteCorreo,
		"requiere_factura":     payload.RequiereFactura,
		"notas":                payload.Notas,
		"total":                total,
	}

	var ventaResult []map[string]interface{}
	if err := supabaseClient.DB.From("ventas").Insert(venta).Execute(&ventaResult); err != nil || len(ventaResult) == 0 {
		http.Error(w, `{"error":"Error al crear venta: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	ventaID := ventaResult[0]["id"].(float64)

	// Insertar detalles de venta
	for _, item := range payload.Articulos {
		detalle := map[string]interface{}{
			"venta_id":        ventaID,
			"articulo_id":     item.ArticuloID,
			"cantidad":        item.Cantidad,
			"precio_unitario": item.PrecioUnitario,
		}
		if err := supabaseClient.DB.From("ventas_detalle").Insert(detalle).Execute(nil); err != nil {
			http.Error(w, `{"error":"Error al insertar detalle: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	// Insertar pagos (fecha se genera automáticamente)
	for _, pago := range payload.Pagos {
		pagoMap := map[string]interface{}{
			"venta_id":    ventaID,
			"monto":       pago.Monto,
			"metodo_pago": pago.MetodoPago,
		}
		if err := supabaseClient.DB.From("pagos").Insert(pagoMap).Execute(nil); err != nil {
			http.Error(w, `{"error":"Error al insertar pago: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	// Respuesta exitosa
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Venta registrada correctamente",
		"venta_id": ventaID,
		"total":    total,
	})
}

// Retorna un resumen de todas las ventas registradas
func handleObtenerVentasResumen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	// Validación de permisos
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("read") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	// Consultar la vista de resumen de ventas
	var ventasResumen []map[string]interface{}
	if err := supabaseClient.DB.From("ventas_resumen").Select("*").Execute(&ventasResumen); err != nil {
		http.Error(w, `{"error":"Error al obtener ventas: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(ventasResumen)
}

// Retorna el detalle de una venta específica
func handleDetalleVenta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	// Validar token JWT
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("read") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	// Obtener venta_id de query params
	ventaIDStr := r.URL.Query().Get("venta_id")
	if ventaIDStr == "" {
		http.Error(w, `{"error":"Debe especificar venta_id"}`, http.StatusBadRequest)
		return
	}
	ventaID, err := strconv.Atoi(ventaIDStr)
	if err != nil {
		http.Error(w, `{"error":"venta_id inválido"}`, http.StatusBadRequest)
		return
	}

	// Consultar la vista de detalle de venta con filtro por venta_id
	var detalles []map[string]interface{}
	if err := supabaseClient.DB.
		From("ventas_detalle_completo").
		Select("*").
		Eq("venta_id", strconv.Itoa(ventaID)).
		Execute(&detalles); err != nil {
		http.Error(w, `{"error":"Error al obtener detalle de venta: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(detalles)
}

// Eliminar Venta
func handleEliminarVenta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	// Validar permisos
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("delete") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	// Payload con venta_id
	var payload struct {
		VentaID int `json:"venta_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"JSON inválido: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	if payload.VentaID == 0 {
		http.Error(w, `{"error":"Debe indicar venta_id"}`, http.StatusBadRequest)
		return
	}

	// Eliminar la venta
	if err := supabaseClient.DB.From("ventas").
		Delete().
		Eq("id", strconv.Itoa(payload.VentaID)).
		Execute(nil); err != nil {
		http.Error(w, `{"error":"Error al eliminar venta: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Respuesta
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Venta eliminada correctamente",
		"venta_id": payload.VentaID,
	})
}

func handleEditarVenta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	// Validar permisos
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("update") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	// Payload
	var payload struct {
		VentaID            int            `json:"venta_id"`
		ClienteNombre      string         `json:"cliente_nombre"`
		ClienteRazonSocial string         `json:"cliente_razon_social"`
		ClienteDireccion   string         `json:"cliente_direccion"`
		ClienteTelefono    string         `json:"cliente_telefono"`
		ClienteCorreo      string         `json:"cliente_correo"`
		Notas              string         `json:"notas,omitempty"`
		Articulos          []VentaDetalle `json:"articulos"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"JSON inválido: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	if payload.VentaID == 0 {
		http.Error(w, `{"error":"Debe indicar venta_id"}`, http.StatusBadRequest)
		return
	}
	if len(payload.Articulos) == 0 {
		http.Error(w, `{"error":"Debe enviar al menos un artículo"}`, http.StatusBadRequest)
		return
	}

	ventaIDStr := strconv.Itoa(payload.VentaID)

	updateVenta := map[string]interface{}{
		"cliente_nombre":       payload.ClienteNombre,
		"cliente_razon_social": payload.ClienteRazonSocial,
		"cliente_direccion":    payload.ClienteDireccion,
		"cliente_telefono":     payload.ClienteTelefono,
		"cliente_correo":       payload.ClienteCorreo,
		"notas":                payload.Notas,
	}

	if err := supabaseClient.DB.From("ventas").
		Update(updateVenta).
		Eq("id", ventaIDStr).
		Execute(nil); err != nil {
		http.Error(w, `{"error":"Error al actualizar venta: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Borrar detalles anteriores
	if err := supabaseClient.DB.From("ventas_detalle").
		Delete().
		Eq("venta_id", ventaIDStr).
		Execute(nil); err != nil {
		http.Error(w, `{"error":"Error al eliminar detalles: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Insertar nuevos detalles
	for _, item := range payload.Articulos {
		detalle := map[string]interface{}{
			"venta_id":        payload.VentaID,
			"articulo_id":     item.ArticuloID,
			"cantidad":        item.Cantidad,
			"precio_unitario": item.PrecioUnitario,
		}
		if err := supabaseClient.DB.From("ventas_detalle").Insert(detalle).Execute(nil); err != nil {
			http.Error(w, `{"error":"Error al insertar detalle: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	// Responder éxito
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Venta editada correctamente",
		"venta_id": payload.VentaID,
	})
}
