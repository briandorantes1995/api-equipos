package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	// Necesario para Access-Control-Max-Age
	"github.com/nedpals/supabase-go"

	"equiposmedicos/middleware" // Tu paquete de middleware

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/joho/godotenv"
)

// FRONT_END debe ser definido globalmente
var FRONT_END string

// supabaseClient debe ser definido globalmente
var supabaseClient *supabase.Client

func main() {
	// Solo intenta cargar el archivo .env si no estás en producción
	env := os.Getenv("ENVIRONMENT")
	if env != "production" {
		if err := godotenv.Load(); err != nil {
			log.Printf("No .env file found, continuing without it")
		}
	}

	FRONT_END = os.Getenv("FRONT_END")
	if FRONT_END == "" {
		log.Fatal("FRONT_END no está definida en el archivo .env")
	}

	supabaseUrl := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_KEY")
	supabaseClient = supabase.CreateClient(supabaseUrl, supabaseKey)

	router := http.NewServeMux()

	// Define un middleware CORS reutilizable para http.ServeMux
	// Este middleware se encargará de las solicitudes OPTIONS y de establecer las cabeceras CORS
	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// **LOG DE DEPURACIÓN:** Ver qué método y ruta recibe el corsHandler
			log.Printf("CORS Handler received request for %s with method: %s", r.URL.Path, r.Method)

			// Establece las cabeceras CORS para todas las respuestas
			w.Header().Set("Access-Control-Allow-Origin", FRONT_END)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS") // Métodos permitidos
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")     // Cabeceras permitidas
			w.Header().Set("Access-Control-Allow-Credentials", "true")                        // Si usas credenciales (cookies, auth headers)
			w.Header().Set("Access-Control-Max-Age", "86400")                                 // Cachea la preflight por 24 horas

			// Si la solicitud es OPTIONS (preflight), responde con 200 OK y termina
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Pasa la solicitud al siguiente manejador en la cadena
			next.ServeHTTP(w, r)
		})
	}

	// Rutas públicas
	router.Handle("/api/public", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Handler /api/public received method: %s", r.Method) // LOG
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"Hello from a public endpoint! You don't need to be authenticated to see this."}`))
	}))

	// Mostrar Artículos (sin autenticación)
	router.Handle("/api/articulos", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Handler /api/articulos received method: %s", r.Method) // LOG
		if r.Method != http.MethodGet {                                    // Mantiene la verificación de método
			http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

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
	}))

	// Rutas protegidas
	router.Handle("/api/articulos/agregar", middleware.EnsureValidToken()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Handler /api/articulos/agregar received method: %s", r.Method) // LOG
			if r.Method != http.MethodPost {                                           // Mantiene la verificación de método
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
			log.Printf("Handler /api/articulos/actualizar received method: %s", r.Method) // LOG
			if r.Method != http.MethodPut {                                               // Mantiene la verificación de método
				http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
				return
			}

			w.Header().Set("Content-Type", "application/json")

			token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
			claims := token.CustomClaims.(*middleware.CustomClaims)
			if !claims.HasPermission("update") {
				http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
				return
			}

			idStr := strings.TrimPrefix(r.URL.Path, "/api/articulos/actualizar/") // Ajusta la extracción del ID si la URL no es /api/articulos/actualizar/{id}
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
	router.Handle("/api/articulos/eliminar/", middleware.EnsureValidToken()( // La barra al final es importante para http.ServeMux si esperas un ID
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Handler /api/articulos/eliminar received method: %s", r.Method) // LOG
			if r.Method != http.MethodDelete {                                          // Mantiene la verificación de método
				http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
				return
			}

			w.Header().Set("Content-Type", "application/json")

			token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
			claims := token.CustomClaims.(*middleware.CustomClaims)
			if !claims.HasPermission("delete") {
				http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
				return
			}

			idStr := strings.TrimPrefix(r.URL.Path, "/api/articulos/eliminar/")
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

	// Mostrar Categorías (sin autenticación)
	router.Handle("/api/categorias", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Handler /api/categorias received method: %s", r.Method) // LOG
		if r.Method != http.MethodGet {                                     // Mantiene la verificación de método
			http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		var result []map[string]interface{}
		err := supabaseClient.DB.From("categorias").Select("*").Execute(&result)
		if err != nil {
			http.Error(w, `{"error":"Error al obtener categorías: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(result)
	}))

	// Insertar Categorías
	router.Handle("/api/categorias/agregar", middleware.EnsureValidToken()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Handler /api/categorias/agregar received method: %s", r.Method) // LOG
			if r.Method != http.MethodPost {                                            // Mantiene la verificación de método
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
	router.Handle("/api/categorias/actualizar/", middleware.EnsureValidToken()( // La barra al final es importante
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Handler /api/categorias/actualizar received method: %s", r.Method) // LOG
			if r.Method != http.MethodPut {                                                // Mantiene la verificación de método
				http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
				return
			}

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
	router.Handle("/api/categorias/eliminar/", middleware.EnsureValidToken()( // La barra al final es importante
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Handler /api/categorias/eliminar received method: %s", r.Method) // LOG
			if r.Method != http.MethodDelete {                                           // Mantiene la verificación de método
				http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
				return
			}

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
				http.Error(w, `{"error":"Error al eliminar: `+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(result)
		}),
	))

	log.Print("Server listening on http://0.0.0.0:3010")
	// Aplica el corsHandler globalmente al router
	if err := http.ListenAndServe("0.0.0.0:3010", corsHandler(router)); err != nil {
		log.Fatalf("There was an error with the http server: %v", err)
	}
}
