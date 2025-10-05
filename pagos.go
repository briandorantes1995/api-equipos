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

func handleObtenerPagos(w http.ResponseWriter, r *http.Request) {
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

	var pagos []Pago

	err := supabaseClient.DB.
		From("pagos").
		Select("*").
		Execute(&pagos)

	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(pagos)
}

// Handler para obtener pagos por id de venta
func handleObtenerPagosId(w http.ResponseWriter, r *http.Request) {
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

	var pagos []map[string]interface{}
	err = supabaseClient.DB.
		From("pagos").
		Select("*").
		Eq("venta_id", idValue).
		Execute(&pagos)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(pagos)
}

func handleAgregarPagos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Validación de token y permisos
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)
	if !claims.HasPermission("write") {
		http.Error(w, `{"message":"Permiso denegado"}`, http.StatusForbidden)
		return
	}

	// Parsear body
	var payload struct {
		VentaID string `json:"venta_id"`
		Pagos   []Pago `json:"pagos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	if payload.VentaID == "" || len(payload.Pagos) == 0 {
		http.Error(w, `{"error":"Faltan datos de venta o pagos"}`, http.StatusBadRequest)
		return
	}

	// Insertar cada pago
	for _, pago := range payload.Pagos {
		pagoMap := map[string]interface{}{
			"venta_id":    payload.VentaID,
			"monto":       pago.Monto,
			"metodo_pago": pago.MetodoPago,
		}

		if err := supabaseClient.DB.From("pagos").Insert(pagoMap).Execute(nil); err != nil {
			http.Error(w, `{"error":"Error al insertar pago: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	// Responder éxito
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Pagos agregados correctamente",
	})
}
