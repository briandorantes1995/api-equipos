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

type ArticleResponse struct {
	ID              int     `json:"id,omitempty"`
	CreatedAt       string  `json:"created_at,omitempty"`
	CategoriaID     int     `json:"categoria_id,omitempty"`
	CodigoBarras    string  `json:"codigo_barras,omitempty"`
	Costo           float64 `json:"costo,omitempty"`
	Descripcion     string  `json:"descripcion,omitempty"`
	Imagen          string  `json:"imagen,omitempty"`
	Inventario      int     `json:"inventario,omitempty"`
	Nombre          string  `json:"nombre,omitempty"`
	PrecioVenta     float64 `json:"precio_venta,omitempty"`
	Proveedor       string  `json:"proveedor,omitempty"`
	SKU             string  `json:"sku,omitempty"`
	CategoriaNombre string  `json:"categoria_nombre,omitempty"`
}

type InventarioArticulo struct {
	ID             int     `json:"id"`
	Nombre         string  `json:"nombre"`
	CantidadActual float64 `json:"cantidad_actual"`
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

		// 1. Obtener todos los artículos
		var articlesRaw []map[string]interface{}
		err := supabaseClient.DB.From("articulos").Select("*").Execute(&articlesRaw)
		if err != nil {
			http.Error(w, `{"error":"Error al obtener artículos: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		// 2. Obtener todas las categorías
		var categoriesRaw []map[string]interface{}
		err = supabaseClient.DB.From("categorias").Select("id,nombre").Execute(&categoriesRaw)
		if err != nil {
			http.Error(w, `{"error":"Error al obtener categorías: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		// 3. Crear un mapa para buscar nombres de categorías por ID
		categoryMap := make(map[int]string)
		for _, cat := range categoriesRaw {
			if id, ok := cat["id"].(float64); ok { // Supabase devuelve números como float64
				if name, ok := cat["nombre"].(string); ok {
					categoryMap[int(id)] = name
				}
			}
		}

		// 4. Transformar los artículos a ArticleResponse y añadir el nombre de la categoría
		var articlesResponse []ArticleResponse
		for _, articleRaw := range articlesRaw {
			var article ArticleResponse
			// Marshal y Unmarshal para convertir map[string]interface{} a ArticleResponse
			// Esto es un truco para mapear los campos automáticamente
			articleBytes, _ := json.Marshal(articleRaw)
			json.Unmarshal(articleBytes, &article)

			// Añadir el nombre de la categoría
			if catID, ok := articleRaw["categoria_id"].(float64); ok { // Supabase devuelve IDs como float64
				if categoryName, found := categoryMap[int(catID)]; found {
					article.CategoriaNombre = categoryName
				} else {
					article.CategoriaNombre = "Categoría Desconocida" // O un valor por defecto
				}
			} else {
				article.CategoriaNombre = "Sin Categoría" // Si categoria_id no es un número
			}
			articlesResponse = append(articlesResponse, article)
		}

		jsonResp, err := json.Marshal(articlesResponse)
		if err != nil {
			http.Error(w, `{"error":"Error al convertir resultado a JSON"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(jsonResp)
	})

	// Handler para /api/articulos/{id} (GET)

	handleGetArticuloPorID := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		// Extraer ID desde la URL
		idStr := strings.TrimPrefix(r.URL.Path, "/api/articulos/")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, `{"error":"ID inválido"}`, http.StatusBadRequest)
			return
		}

		// Consultar el artículo por ID
		var articuloRaw []map[string]interface{}
		err = supabaseClient.DB.From("articulos").Select("*").Eq("id", strconv.Itoa(id)).Execute(&articuloRaw)
		if err != nil || len(articuloRaw) == 0 {
			http.Error(w, `{"error":"Artículo no encontrado"}`, http.StatusNotFound)
			return
		}

		// Obtener el nombre de la categoría
		nombreCategoria := "Sin Categoría"
		if catID, ok := articuloRaw[0]["categoria_id"].(float64); ok {
			var categoria []map[string]interface{}
			err = supabaseClient.DB.From("categorias").Select("nombre").Eq("id", strconv.Itoa(int(catID))).Execute(&categoria)
			if err == nil && len(categoria) > 0 {
				if nombre, ok := categoria[0]["nombre"].(string); ok {
					nombreCategoria = nombre
				}
			}
		}

		var articulo ArticleResponse
		articleBytes, _ := json.Marshal(articuloRaw[0])
		json.Unmarshal(articleBytes, &articulo)

		articulo.CategoriaNombre = nombreCategoria

		// Enviar JSON de respuesta
		resp, err := json.Marshal(articulo)
		if err != nil {
			http.Error(w, `{"error":"Error al generar JSON"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resp)
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

		// Extraemos inventario inicial y usuario_id del payload si viene
		cantidadInicial, ok := nuevo["inventario"].(float64)
		if !ok {
			cantidadInicial = 0 // default
		}
		nombreUsuario, _ := nuevo["name"].(float64)
		delete(nuevo, "inventario")
		delete(nuevo, "name")

		// 1️⃣ Insertar artículo
		var results []map[string]interface{}
		err = supabaseClient.DB.From("articulos").Insert(nuevo).Execute(&results)
		if err != nil || len(results) == 0 {
			http.Error(w, `{"error":"Error al insertar artículo: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		articuloID := results[0]["id"].(float64) // id del artículo insertado

		// 2️⃣ Insertar inventario inicial
		inventario := map[string]interface{}{
			"articulo_id":     articuloID,
			"cantidad_actual": cantidadInicial,
		}
		err = supabaseClient.DB.From("inventarios").Insert(inventario).Execute(nil)
		if err != nil {
			http.Error(w, `{"error":"Error al insertar inventario: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		// 3️⃣ Registrar movimiento inicial
		movimiento := map[string]interface{}{
			"articulo_id":     articuloID,
			"tipo_movimiento": "alta",
			"cantidad":        cantidadInicial,
			"motivo":          "Inventario inicial",
			"usuario_nombre":  nombreUsuario,
		}
		err = supabaseClient.DB.From("movimientos_inventario").Insert(movimiento).Execute(nil)
		if err != nil {
			http.Error(w, `{"error":"Error al registrar movimiento: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		// Responder con el artículo creado
		json.NewEncoder(w).Encode(results[0])
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

	handleBuscarArticulos := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		termino := strings.ToLower(r.URL.Query().Get("busqueda"))
		if termino == "" {
			http.Error(w, `{"error":"Parámetro 'busqueda' obligatorio"}`, http.StatusBadRequest)
			return
		}

		var articlesRaw []map[string]interface{}
		err := supabaseClient.DB.From("articulos").Select("*").Execute(&articlesRaw)
		if err != nil {
			http.Error(w, `{"error":"Error al obtener artículos: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		// Obtener categorías para mapear id->nombre
		var categoriesRaw []map[string]interface{}
		err = supabaseClient.DB.From("categorias").Select("id,nombre").Execute(&categoriesRaw)
		if err != nil {
			http.Error(w, `{"error":"Error al obtener categorías: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		categoryMap := make(map[int]string)
		for _, cat := range categoriesRaw {
			if id, ok := cat["id"].(float64); ok {
				if name, ok := cat["nombre"].(string); ok {
					categoryMap[int(id)] = name
				}
			}
		}

		// Filtrar manualmente por nombre, proveedor o categoria_nombre
		var filtered []ArticleResponse
		for _, art := range articlesRaw {
			nombre := strings.ToLower(art["nombre"].(string))
			proveedor, _ := art["proveedor"].(string)
			proveedor = strings.ToLower(proveedor)
			categoriaNombre := ""
			if catIDf, ok := art["categoria_id"].(float64); ok {
				categoriaNombre = strings.ToLower(categoryMap[int(catIDf)])
			}

			if strings.Contains(nombre, termino) ||
				strings.Contains(proveedor, termino) ||
				strings.Contains(categoriaNombre, termino) {

				var article ArticleResponse
				bytesArt, _ := json.Marshal(art)
				json.Unmarshal(bytesArt, &article)
				if catIDf, ok := art["categoria_id"].(float64); ok {
					article.CategoriaNombre = categoryMap[int(catIDf)]
				} else {
					article.CategoriaNombre = "Sin Categoría"
				}
				filtered = append(filtered, article)
			}
		}

		jsonResp, err := json.Marshal(filtered)
		if err != nil {
			http.Error(w, `{"error":"Error al convertir a JSON: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(jsonResp)
	})

	handleReporteInventario := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			From("inventarios").
			Select("articulos:id,articulos:nombre,cantidad_actual").
			Execute(&inventarios)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(inventarios)
	})

	// Registro de rutas con http.ServeMux
	router.Handle("/api/public", handlePublic)
	router.Handle("/api/articulos", handleGetArticulos)
	router.Handle("/api/articulos/", handleGetArticuloPorID)
	router.Handle("/api/articulos/agregar", middleware.EnsureValidToken()(handleAgregarArticulo))
	router.Handle("/api/articulos/actualizar/", middleware.EnsureValidToken()(handleActualizarArticulo))
	router.Handle("/api/articulos/eliminar/", middleware.EnsureValidToken()(handleEliminarArticulo))
	router.Handle("/api/articulos/buscar", handleBuscarArticulos)
	router.Handle("/api/categorias", handleGetCategorias)
	router.Handle("/api/categorias/agregar", middleware.EnsureValidToken()(handleAgregarCategoria))
	router.Handle("/api/categorias/actualizar/", middleware.EnsureValidToken()(handleActualizarCategoria))
	router.Handle("/api/categorias/eliminar/", middleware.EnsureValidToken()(handleEliminarCategoria))
	router.Handle("/api/inventario", middleware.EnsureValidToken()(handleReporteInventario))

	log.Print("Server listening on http://0.0.0.0:3010")
	if err := http.ListenAndServe("0.0.0.0:3010", corsHandler(router)); err != nil {
		log.Fatalf("There was an error with the http server: %v", err)
	}

}
