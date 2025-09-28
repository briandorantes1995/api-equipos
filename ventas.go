package main

import (
	"encoding/json"
	"equiposmedicos/middleware"
	"net/http"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
)

func handleRegistrarVenta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("create") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	var payload VentaPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"JSON inválido: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	if len(payload.Articulos) == 0 {
		http.Error(w, `{"error":"Debe enviar al menos un artículo"}`, http.StatusBadRequest)
		return
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
		if err := supabaseClient.DB.From("venta_detalle").Insert(detalle).Execute(nil); err != nil {
			http.Error(w, `{"error":"Error al insertar detalle: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	// Insertar pagos, si los hay
	for _, pago := range payload.Pagos {
		fecha := pago.FechaPago
		if fecha == "" {
			fecha = time.Now().Format("2006-01-02 15:04:05")
		}
		pagoMap := map[string]interface{}{
			"venta_id":    ventaID,
			"monto":       pago.Monto,
			"metodo_pago": pago.MetodoPago,
			"fecha_pago":  fecha,
		}
		if err := supabaseClient.DB.From("ventas_pagos").Insert(pagoMap).Execute(nil); err != nil {
			http.Error(w, `{"error":"Error al insertar pago: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Venta registrada correctamente",
		"venta_id": ventaID,
	})
}
