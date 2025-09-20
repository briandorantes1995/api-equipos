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
	ctxToken := r.Context().Value(jwtmiddleware.ContextKey{})
	if ctxToken == nil {
		http.Error(w, `{"message":"Token inválido"}`, http.StatusUnauthorized)
		return
	}

	token, ok := ctxToken.(*validator.ValidatedClaims)
	if !ok {
		http.Error(w, `{"message":"Token inválido"}`, http.StatusUnauthorized)
		return
	}

	claims, ok := token.CustomClaims.(*middleware.CustomClaims)
	if !ok || !claims.HasPermission("read") {
		http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
		return
	}

	var inventarios []TomaInventario

	// Consulta usando join soportado por PostgREST
	err := supabaseClient.DB.
		From("tomafisica").
		Select(`
            id,
            folio,
            fecha_inicio,
            fecha_fin,
            usuario_auth0_sub,
            usuario_correo,
            estado,
            categoria_id,
            categorias(id,nombre)
        `).
		Execute(&inventarios)

	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Mapear nombre de categoría al campo plano
	for i := range inventarios {
		if inventarios[i].Categoria != nil {
			inventarios[i].CategoriaNombre = inventarios[i].Categoria.Nombre
		} else {
			inventarios[i].CategoriaNombre = "Todas"
		}
	}

	// Retornar JSON
	if inventarios == nil {
		inventarios = []TomaInventario{}
	}

	json.NewEncoder(w).Encode(inventarios)
}
