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

func handleCrearTomaFisica(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	ctxToken := r.Context().Value(jwtmiddleware.ContextKey{})
	if ctxToken == nil {
		http.Error(w, `{"message":"Token inválido"}`, http.StatusUnauthorized)
		return
	}

	token := ctxToken.(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*middleware.CustomClaims)

	usuarioAuth0Sub := token.RegisteredClaims.Subject
	usuarioCorreo := claims.Email

	var payload struct {
		CategoriaID *int `json:"categoria_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"message":"Datos inválidos"}`, http.StatusBadRequest)
		return
	}

	var resultado []struct {
		TomaID int `json:"toma_id"`
		Folio  int `json:"folio"`
	}

	err := supabaseClient.DB.
		Rpc("crear_toma_fisica", map[string]interface{}{
			"p_usuario_auth0_sub": usuarioAuth0Sub,
			"p_usuario_correo":    usuarioCorreo,
			"p_categoria_id":      payload.CategoriaID,
		}).
		Execute(&resultado)

	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if len(resultado) == 0 {
		http.Error(w, `{"message":"No se pudo crear la toma"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resultado[0])
}
