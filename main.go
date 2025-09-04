package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
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
	router.HandleFunc("/api/articulos/buscar", handleBuscarArticulos)
	router.Handle("/api/articulos/agregar", middleware.EnsureValidToken()(http.HandlerFunc(handleAgregarArticulo)))
	router.Handle("/api/articulos/actualizar/", middleware.EnsureValidToken()(http.HandlerFunc(handleActualizarArticulo)))
	router.Handle("/api/articulos/eliminar/", middleware.EnsureValidToken()(http.HandlerFunc(handleEliminarArticulo)))

	router.HandleFunc("/api/categorias", handleGetCategorias)
	router.Handle("/api/categorias/agregar", middleware.EnsureValidToken()(http.HandlerFunc(handleAgregarCategoria)))
	router.Handle("/api/categorias/actualizar/", middleware.EnsureValidToken()(http.HandlerFunc(handleActualizarCategoria)))
	router.Handle("/api/categorias/eliminar/", middleware.EnsureValidToken()(http.HandlerFunc(handleEliminarCategoria)))

	router.Handle("/api/inventario", middleware.EnsureValidToken()(handleReporteInventario))

	router.Handle("/api/movimientos/registrar", middleware.EnsureValidToken()(handleRegistrarMovimiento))
	router.Handle("/api/movimientos", middleware.EnsureValidToken()(handleReporteMovimientos))

	log.Print("Server listening on http://0.0.0.0:3010")
	if err := http.ListenAndServe("0.0.0.0:3010", corsHandler(router)); err != nil {
		log.Fatalf("There was an error with the http server: %v", err)
	}

}
