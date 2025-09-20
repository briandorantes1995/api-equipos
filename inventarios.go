package main

import (
	"encoding/json"
	"equiposmedicos/middleware"
	"net/http"

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
		From("TomaFisica").
		Select(`
			id,
			folio,
			fecha_inicio,
			fecha_fin,
			usuario_auth0_sub,
			usuario_correo,
			estado,
			categoria_id,
			categorias!inner(nombre)
		`).
		Execute(&inventarios)

	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Mapear el nombre de la categoría si existe
	for i, inv := range inventarios {
		if inv.CategoriaID != nil {
			inventarios[i].CategoriaNombre = inv.CategoriaNombre
		} else {
			inventarios[i].CategoriaNombre = "Todas"
		}
	}

	json.NewEncoder(w).Encode(inventarios)
}
