package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nedpals/supabase-go"

	"equiposmedicos/middleware"

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
			From("inventario_view").
			Select("*").
			Execute(&inventarios)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(inventarios)
	})

	// Handler para registrar movimientos y actualizar inventario
	handleRegistrarMovimiento := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"message":"Método no permitido"}`, http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		// Validación de token y permisos
		token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
		claims := token.CustomClaims.(*middleware.CustomClaims)
		if !claims.HasPermission("write") {
			http.Error(w, `{"message":"Insufficient scope."}`, http.StatusForbidden)
			return
		}

		// Decodificar el movimiento
		var movimiento Movimiento
		if err := json.NewDecoder(r.Body).Decode(&movimiento); err != nil {
			http.Error(w, `{"error":"Error al decodificar JSON: `+err.Error()+`"}`, http.StatusBadRequest)
			return
		}

		// Validar campos
		if movimiento.ArticuloID <= 0 || movimiento.TipoMovimiento == "" || movimiento.Cantidad <= 0 {
			http.Error(w, `{"error":"Datos del movimiento inválidos"}`, http.StatusBadRequest)
			return
		}

		// Registrar movimiento
		err := supabaseClient.DB.
			From("movimientos_inventario").
			Insert(movimiento).
			Execute(nil)
		if err != nil {
			http.Error(w, `{"error":"Error al insertar movimiento: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		// Obtener inventario actual
		var inventarios []InventarioArticulo
		err = supabaseClient.DB.
			From("inventarios").
			Select("*").
			Eq("articulo_id", strconv.Itoa(movimiento.ArticuloID)).
			Execute(&inventarios)

		if err != nil {
			http.Error(w, `{"error":"Error al obtener inventario: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		cantidadActual := 0.0
		if len(inventarios) > 0 {
			cantidadActual = inventarios[0].CantidadActual
		}

		// Actualizar inventario según tipo
		switch movimiento.TipoMovimiento {
		case "alta", "compra", "transferencia_entrada":
			cantidadActual += movimiento.Cantidad
		case "venta", "baja", "robo", "transferencia_salida":
			cantidadActual -= movimiento.Cantidad
		default:
			http.Error(w, `{"error":"Tipo de movimiento desconocido"}`, http.StatusBadRequest)
			return
		}

		// Upsert en inventario
		upsert := map[string]interface{}{
			"articulo_id":          movimiento.ArticuloID,
			"cantidad_actual":      cantidadActual,
			"ultima_actualizacion": time.Now(),
		}

		// Solo un valor de retorno, no dos
		err = supabaseClient.DB.
			From("inventarios").
			Upsert(upsert).
			Execute(nil)
		if err != nil {
			http.Error(w, `{"error":"Error al actualizar inventario: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		// Respuesta
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":         "Movimiento registrado y inventario actualizado",
			"articulo_id":     movimiento.ArticuloID,
			"cantidad_actual": cantidadActual,
		})
	})

	handleReporteMovimientos := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		var movimientos []MovimientoConNombre

		// Traemos todos los movimientos desde la view
		err := supabaseClient.DB.
			From("movimientos_con_nombre").
			Select("*").
			Execute(&movimientos)

		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(movimientos)
	})

	// Registro de rutas con http.ServeMux
	router.Handle("/api/public", handlePublic)

	router.HandleFunc("/api/articulos", handleGetArticulos)
	router.HandleFunc("/api/articulos/", handleGetArticuloPorID)
	router.Handle("/api/articulos/agregar", middleware.EnsureValidToken()(http.HandlerFunc(handleAgregarArticulo)))
	router.Handle("/api/articulos/actualizar/", middleware.EnsureValidToken()(http.HandlerFunc(handleActualizarArticulo)))
	router.Handle("/api/articulos/eliminar/", middleware.EnsureValidToken()(http.HandlerFunc(handleEliminarArticulo)))
	router.HandleFunc("/api/articulos/buscar", handleBuscarArticulos)

	router.Handle("/api/categorias", handleGetCategorias)
	router.Handle("/api/categorias/agregar", middleware.EnsureValidToken()(handleAgregarCategoria))
	router.Handle("/api/categorias/actualizar/", middleware.EnsureValidToken()(handleActualizarCategoria))
	router.Handle("/api/categorias/eliminar/", middleware.EnsureValidToken()(handleEliminarCategoria))

	router.Handle("/api/inventario", middleware.EnsureValidToken()(handleReporteInventario))

	router.Handle("/api/movimientos/registrar", middleware.EnsureValidToken()(handleRegistrarMovimiento))
	router.Handle("/api/movimientos", middleware.EnsureValidToken()(handleReporteMovimientos))

	log.Print("Server listening on http://0.0.0.0:3010")
	if err := http.ListenAndServe("0.0.0.0:3010", corsHandler(router)); err != nil {
		log.Fatalf("There was an error with the http server: %v", err)
	}

}
