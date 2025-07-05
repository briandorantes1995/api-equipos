package main

// main.go
import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/nedpals/supabase-go"

	"equiposmedicos/middleware"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading the .env file: %v", err)
	}
	supabaseUrl := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_KEY")
	supabaseClient := supabase.CreateClient(supabaseUrl, supabaseKey)

	router := http.NewServeMux()

	// This route is always accessible.
	router.Handle("/api/public", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"Hello from a public endpoint! You don't need to be authenticated to see this."}`))
	}))
	//Mostrar Articulos
	router.Handle("/api/articulos", middleware.EnsureValidToken()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// CORS Headers (antes de cualquier validación)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization")
			w.Header().Set("Content-Type", "application/json")

			token, ok := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
			if !ok || token.CustomClaims == nil {
				http.Error(w, `{"message":"Invalid token claims"}`, http.StatusUnauthorized)
				return
			}
			claims := token.CustomClaims.(*middleware.CustomClaims)
			if !claims.HasPermission("read") {
				http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
				return
			}

			var result []map[string]interface{}
			err := supabaseClient.DB.From("articulos").Select("*").Execute(&result)
			if err != nil {
				http.Error(w, `{"error":"Error de conexión o tabla no existe: `+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}

			jsonResp, err := json.Marshal(result)
			if err != nil {
				http.Error(w, `{"error":"Error al convertir resultado a JSON"}`, http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write(jsonResp)
		}),
	))

	//Insertar Articulos
	router.Handle("/api/articulos/agregar", middleware.EnsureValidToken()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Content-Type", "application/json")

			token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
			claims := token.CustomClaims.(*middleware.CustomClaims)
			if !claims.HasPermission("create") {
				http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
				return
			}

			var nuevo map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&nuevo)
			if err != nil {
				http.Error(w, `{"error":"JSON inválido"}`, http.StatusBadRequest)
				return
			}

			var result map[string]interface{}
			err = supabaseClient.DB.From("articulos").Insert(nuevo).Execute(&result)
			if err != nil {
				http.Error(w, `{"error":"Error al insertar: `+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}

			json.NewEncoder(w).Encode(result)
		}),
	))

	// Actualizar Articulos
	router.Handle("/api/articulos/actualizar", middleware.EnsureValidToken()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Content-Type", "application/json")

			token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
			claims := token.CustomClaims.(*middleware.CustomClaims)
			if !claims.HasPermission("update") {
				http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
				return
			}

			idStr := strings.TrimPrefix(r.URL.Path, "/api/articulos/")
			id, err := strconv.Atoi(idStr)
			if err != nil {
				http.Error(w, `{"error":"ID inválido"}`, http.StatusBadRequest)
				return
			}

			var datos map[string]interface{}
			err = json.NewDecoder(r.Body).Decode(&datos)
			if err != nil {
				http.Error(w, `{"error":"JSON inválido"}`, http.StatusBadRequest)
				return
			}

			var result map[string]interface{}
			err = supabaseClient.DB.From("articulos").Update(datos).Eq("id", strconv.Itoa(id)).Execute(&result)
			if err != nil {
				http.Error(w, `{"error":"Error al actualizar: `+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}

			json.NewEncoder(w).Encode(result)
		}),
	))

	// Eliminar Articulos
	router.Handle("/api/articulos/eliminar", middleware.EnsureValidToken()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Content-Type", "application/json")

			token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
			claims := token.CustomClaims.(*middleware.CustomClaims)
			if !claims.HasPermission("delete") {
				http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
				return
			}

			idStr := strings.TrimPrefix(r.URL.Path, "/api/articulos/")
			id, err := strconv.Atoi(idStr)
			if err != nil {
				http.Error(w, `{"error":"ID inválido"}`, http.StatusBadRequest)
				return
			}

			var result map[string]interface{}
			err = supabaseClient.DB.From("articulos").Delete().Eq("id", strconv.Itoa(id)).Execute(&result)
			if err != nil {
				http.Error(w, `{"error":"Error al eliminar: `+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}

			json.NewEncoder(w).Encode(result)
		}),
	))

	// Mostrar Categorías
	router.Handle("/api/categorias", middleware.EnsureValidToken()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
				return
			}

			// CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization")
			w.Header().Set("Content-Type", "application/json")

			token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
			claims := token.CustomClaims.(*middleware.CustomClaims)
			if !claims.HasPermission("read") {
				http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
				return
			}

			var result []map[string]interface{}
			err := supabaseClient.DB.From("categorias").Select("*").Execute(&result)
			if err != nil {
				http.Error(w, `{"error":"Error al obtener categorías: `+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(result)
		}),
	))

	// Insertar Categorías
	router.Handle("/api/categorias/agregar", middleware.EnsureValidToken()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Content-Type", "application/json")

			token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
			claims := token.CustomClaims.(*middleware.CustomClaims)
			if !claims.HasPermission("create") {
				http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
				return
			}

			var nueva map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&nueva); err != nil {
				http.Error(w, `{"error":"JSON inválido"}`, http.StatusBadRequest)
				return
			}

			var result map[string]interface{}
			err := supabaseClient.DB.From("categorias").Insert(nueva).Execute(&result)
			if err != nil {
				http.Error(w, `{"error":"Error al insertar categoría: `+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(result)
		}),
	))

	// Actualizar Categorías
	router.Handle("/api/categorias/actualizar", middleware.EnsureValidToken()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Content-Type", "application/json")

			token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
			claims := token.CustomClaims.(*middleware.CustomClaims)
			if !claims.HasPermission("update") {
				http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
				return
			}

			// Obtener el ID desde la URL
			idStr := strings.TrimPrefix(r.URL.Path, "/api/categorias/actualizar/")
			id, err := strconv.Atoi(idStr)
			if err != nil {
				http.Error(w, `{"error":"ID inválido"}`, http.StatusBadRequest)
				return
			}

			var datos map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&datos); err != nil {
				http.Error(w, `{"error":"JSON inválido"}`, http.StatusBadRequest)
				return
			}

			var result map[string]interface{}
			err = supabaseClient.DB.From("categorias").Update(datos).Eq("id", strconv.Itoa(id)).Execute(&result)
			if err != nil {
				http.Error(w, `{"error":"Error al actualizar categoría: `+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(result)
		}),
	))

	// Eliminar Categorías
	router.Handle("/api/categorias/eliminar/", middleware.EnsureValidToken()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization")
			w.Header().Set("Content-Type", "application/json")

			token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
			claims := token.CustomClaims.(*middleware.CustomClaims)
			if !claims.HasPermission("delete") {
				http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
				return
			}

			// Obtener ID desde URL
			idStr := strings.TrimPrefix(r.URL.Path, "/api/categorias/eliminar/")
			id, err := strconv.Atoi(idStr)
			if err != nil {
				http.Error(w, `{"error":"ID inválido"}`, http.StatusBadRequest)
				return
			}

			var result map[string]interface{}
			err = supabaseClient.DB.From("categorias").Delete().Eq("id", strconv.Itoa(id)).Execute(&result)
			if err != nil {
				http.Error(w, `{"error":"Error al eliminar categoría: `+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(result)
		}),
	))

	log.Print("Server listening on http://localhost:3010")
	if err := http.ListenAndServe("0.0.0.0:3010", router); err != nil {
		log.Fatalf("There was an error with the http server: %v", err)
	}

}
