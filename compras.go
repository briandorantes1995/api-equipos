package main

import (
	"encoding/json"
	"equiposmedicos/middleware"
	"net/http"
	"strconv"

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

// Retorna un resumen de todas las compras registradas
func handleObtenerComprasResumen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	// validacion
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("read") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	// consultar la vista
	var comprasResumen []map[string]interface{}
	if err := supabaseClient.DB.From("vista_compras_resumen").Select("*").Execute(&comprasResumen); err != nil {
		http.Error(w, `{"error":"Error al obtener compras: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(comprasResumen)
}

// handleDetalleCompra retorna el detalle de una compra específica
func handleDetalleCompra(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	// validar token JWT
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("read") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	// obtener compra_id de query params
	compraIDStr := r.URL.Query().Get("compra_id")
	if compraIDStr == "" {
		http.Error(w, `{"error":"Debe especificar compra_id"}`, http.StatusBadRequest)
		return
	}
	compraID, err := strconv.Atoi(compraIDStr)
	if err != nil {
		http.Error(w, `{"error":"compra_id inválido"}`, http.StatusBadRequest)
		return
	}

	// consultar la vista con filtro por compra_id
	var detalles []map[string]interface{}
	if err := supabaseClient.DB.
		From("vista_compras_detalle").
		Select("*").
		Eq("compra_id", strconv.Itoa(compraID)).
		Execute(&detalles); err != nil {
		http.Error(w, `{"error":"Error al obtener detalle de compra: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(detalles)
}
