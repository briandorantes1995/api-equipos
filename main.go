package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

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

// CategoryDetail representa el objeto de categoría anidado devuelto por Supabase
type CategoryDetail struct {
	Nombre string `json:"nombre,omitempty"`
}

// ArticleResponse representa la estructura de un artículo con el nombre de su categoría
type ArticleResponse struct {
	ID           int             `json:"id,omitempty"`
	CreatedAt    string          `json:"created_at,omitempty"`
	CategoriaID  int             `json:"categoria_id,omitempty"`
	CodigoBarras string          `json:"codigo_barras,omitempty"`
	Costo        float64         `json:"costo,omitempty"`
	Descripcion  string          `json:"descripcion,omitempty"`
	Imagen       string          `json:"imagen,omitempty"`
	Inventario   int             `json:"inventario,omitempty"`
	Nombre       string          `json:"nombre,omitempty"`
	PrecioVenta  float64         `json:"precio_venta,omitempty"`
	Proveedor    string          `json:"proveedor,omitempty"`
	SKU          string          `json:"sku,omitempty"`
	Categoria    *CategoryDetail `json:"categoria,omitempty"` // Objeto de categoría anidado
}

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
			w.Header().Set("Access-Control-Allow-Origin", FRONT_END)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Handler para /api/public
	handlePublic := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"Hello from a public endpoint! You don't need to be authenticated to see this."}`))
	})

	// Handler para /api/articulos (GET)
	handleGetArticulos := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		var result []ArticleResponse // Usa la nueva struct ArticleResponse
		// CAMBIO CLAVE: Elimina 'created_at' de la selección explícita
		err := supabaseClient.DB.From("articulos").Select("id,categoria_id,codigo_barras,costo,descripcion,imagen,inventario,nombre,precio_venta,proveedor,sku,categorias(nombre)").Execute(&result)
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
	})

	// Handler para /api/articulos/agregar (POST)
	handleAgregarArticulo := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		var nuevo map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&nuevo)
		if err != nil {
			http.Error(w, `{"error":"JSON inválido: `+err.Error()+`"}`, http.StatusBadRequest)
			return
		}

		var results []map[string]interface{} // Usamos directamente slice de mapas
		err = supabaseClient.DB.From("articulos").Insert(nuevo).Execute(&results)
		if err != nil {
			http.Error(w, `{"error":"Error al insertar: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		if len(results) > 0 {
			json.NewEncoder(w).Encode(results[0])
		} else {
			http.Error(w, `{"message":"Inserción exitosa, pero no se recibieron datos de respuesta."}`, http.StatusOK)
		}
	})

	// Handler para /api/articulos/actualizar (PUT)
	handleActualizarArticulo := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
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

		idStr := strings.TrimPrefix(r.URL.Path, "/api/articulos/actualizar/")
		var id int
		var err error
		id, err = strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, `{"error":"ID inválido"}`, http.StatusBadRequest)
			return
		}

		var datos map[string]interface{}
		err = json.NewDecoder(r.Body).Decode(&datos)
		if err != nil {
			http.Error(w, `{"error":"JSON inválido: `+err.Error()+`"}`, http.StatusBadRequest)
			return
		}

		var results []map[string]interface{} // Usamos directamente slice de mapas
		err = supabaseClient.DB.From("articulos").Update(datos).Eq("id", strconv.Itoa(id)).Execute(&results)
		if err != nil {
			http.Error(w, `{"error":"Error al actualizar: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		if len(results) > 0 {
			json.NewEncoder(w).Encode(results[0])
		} else {
			http.Error(w, `{"message":"Actualización exitosa, pero no se recibieron datos de respuesta."}`, http.StatusOK)
		}
	})

	// Handler para /api/articulos/eliminar (DELETE)
	handleEliminarArticulo := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
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
		var id int
		var err error
		id, err = strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, `{"error":"ID inválido"}`, http.StatusBadRequest)
			return
		}

		var results []map[string]interface{} // Usamos directamente slice de mapas
		err = supabaseClient.DB.From("articulos").Delete().Eq("id", strconv.Itoa(id)).Execute(&results)
		if err != nil {
			http.Error(w, `{"error":"Error al eliminar: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		if len(results) > 0 {
			json.NewEncoder(w).Encode(results[0])
		} else {
			http.Error(w, `{"message":"Eliminación exitosa, pero no se recibieron datos de respuesta."}`, http.StatusOK)
		}
	})

	// Handler para /api/categorias (GET)
	handleGetCategorias := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		var result []map[string]interface{}
		err := supabaseClient.DB.From("categorias").Select("*").Execute(&result)
		if err != nil {
			http.Error(w, `{"error":"Error de conexión o tabla no existe: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(result)
	})

	// Handler para /api/categorias/agregar (POST)
	handleAgregarCategoria := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		var nueva map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&nueva)
		if err != nil {
			http.Error(w, `{"error":"JSON inválido: `+err.Error()+`"}`, http.StatusBadRequest)
			return
		}

		var results []map[string]interface{} // Usamos directamente slice de mapas
		err = supabaseClient.DB.From("categorias").Insert(nueva).Execute(&results)
		if err != nil {
			http.Error(w, `{"error":"Error al insertar categoría: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		if len(results) > 0 {
			json.NewEncoder(w).Encode(results[0])
		} else {
			http.Error(w, `{"message":"Inserción exitosa, pero no se recibieron datos de respuesta."}`, http.StatusOK)
		}
	})

	// Handler para /api/categorias/actualizar (PUT)
	handleActualizarCategoria := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
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

		idStr := strings.TrimPrefix(r.URL.Path, "/api/categorias/actualizar/")
		var id int
		var err error // Declarar err aquí
		id, err = strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, `{"error":"ID inválido"}`, http.StatusBadRequest)
			return
		}

		var datos map[string]interface{}
		err = json.NewDecoder(r.Body).Decode(&datos)
		if err != nil {
			http.Error(w, `{"error":"JSON inválido: `+err.Error()+`"}`, http.StatusBadRequest)
			return
		}

		var results []map[string]interface{} // Usamos directamente slice de mapas
		err = supabaseClient.DB.From("categorias").Update(datos).Eq("id", strconv.Itoa(id)).Execute(&results)
		if err != nil {
			http.Error(w, `{"error":"Error al actualizar en Supabase: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		if len(results) > 0 {
			json.NewEncoder(w).Encode(results[0])
		} else {
			http.Error(w, `{"message":"Actualización exitosa, pero no se recibieron datos de respuesta."}`, http.StatusOK)
		}
	})

	// Handler para /api/categorias/eliminar (DELETE)
	handleEliminarCategoria := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
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

		idStr := strings.TrimPrefix(r.URL.Path, "/api/categorias/eliminar/")
		var id int
		var err error // Declarar err aquí
		id, err = strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, `{"error":"ID inválido"}`, http.StatusBadRequest)
			return
		}

		var results []map[string]interface{} // Usamos directamente slice de mapas
		err = supabaseClient.DB.From("categorias").Delete().Eq("id", strconv.Itoa(id)).Execute(&results)
		if err != nil {
			http.Error(w, `{"error":"Error al eliminar: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		if len(results) > 0 {
			json.NewEncoder(w).Encode(results[0])
		} else {
			http.Error(w, `{"message":"Eliminación exitosa, pero no se recibieron datos de respuesta."}`, http.StatusOK)
		}
	})

	// Registro de rutas con http.ServeMux
	router.Handle("/api/public", handlePublic)
	router.Handle("/api/articulos", handleGetArticulos)
	router.Handle("/api/articulos/agregar", middleware.EnsureValidToken()(handleAgregarArticulo))
	router.Handle("/api/articulos/actualizar/", middleware.EnsureValidToken()(handleActualizarArticulo))
	router.Handle("/api/articulos/eliminar/", middleware.EnsureValidToken()(handleEliminarArticulo))
	router.Handle("/api/categorias", handleGetCategorias)
	router.Handle("/api/categorias/agregar", middleware.EnsureValidToken()(handleAgregarCategoria))
	router.Handle("/api/categorias/actualizar/", middleware.EnsureValidToken()(handleActualizarCategoria))
	router.Handle("/api/categorias/eliminar/", middleware.EnsureValidToken()(handleEliminarCategoria))

	log.Print("Server listening on http://0.0.0.0:3010")
	if err := http.ListenAndServe("0.0.0.0:3010", corsHandler(router)); err != nil {
		log.Fatalf("There was an error with the http server: %v", err)
	}
}
