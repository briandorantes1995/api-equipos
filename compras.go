package main

import (
	"encoding/json"
	"equiposmedicos/middleware"
	"net/http"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
)

// handleRegistrarCompra maneja la ruta para registrar una nueva compra.
func handleRegistrarCompra(w http.ResponseWriter, r *http.Request) {
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

	var payload struct {
		Articulos []struct {
			ArticuloID     int     `json:"articulo_id"`
			Cantidad       int     `json:"cantidad"`
			PrecioUnitario float64 `json:"precio_unitario"`
		} `json:"articulos"`
		Notas string `json:"notas,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"JSON inválido: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	if len(payload.Articulos) == 0 {
		http.Error(w, `{"error":"Debe enviar al menos un artículo"}`, http.StatusBadRequest)
		return
	}

	// Insertar compra
	compra := map[string]interface{}{
		"notas": payload.Notas,
	}
	var compraResult []map[string]interface{}
	if err := supabaseClient.DB.From("compras").Insert(compra).Execute(&compraResult); err != nil || len(compraResult) == 0 {
		http.Error(w, `{"error":"Error al crear compra: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	compraID := compraResult[0]["id"].(float64)

	// Insertar detalles
	for _, item := range payload.Articulos {
		detalle := map[string]interface{}{
			"compra_id":       compraID,
			"articulo_id":     item.ArticuloID,
			"cantidad":        item.Cantidad,
			"precio_unitario": item.PrecioUnitario,
		}
		if err := supabaseClient.DB.From("compras_detalles").Insert(detalle).Execute(nil); err != nil {
			http.Error(w, `{"error":"Error al insertar detalle: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Compra registrada correctamente",
		"compra_id": compraID,
	})
}
