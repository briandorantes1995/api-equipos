package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"equiposmedicos/middleware"

	"github.com/joho/godotenv"
	"github.com/nedpals/supabase-go"
)

// FRONT_END global
var FRONT_END string

// supabaseClient global
var supabaseClient *supabase.Client

func main() {
	// Cargar variables de entorno si no estamos en producción
	env := os.Getenv("ENVIRONMENT")
	if env != "production" {
		if err := godotenv.Load(); err != nil {
			log.Printf("No .env file found, continuing without it")
		}
	}

	FRONT_END := os.Getenv("FRONT_END")
	if FRONT_END == "" {
		FRONT_END = "http://localhost:5173"
	}

	FRONT_END2 := os.Getenv("FRONT_END2")
	if FRONT_END2 == "" {
		FRONT_END2 = "https://equipos-front.vercel.app"
	}

	// Inicializar Supabase
	supabaseUrl := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	supabaseClient = supabase.CreateClient(supabaseUrl, supabaseKey)

	router := http.NewServeMux()

	// Middleware CORS dinámico
	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowedOrigins := []string{FRONT_END, FRONT_END2}

			allowed := false
			for _, o := range allowedOrigins {
				if o != "" && o == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Responder a solicitudes OPTIONS (preflight)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Rutas públicas
	router.HandleFunc("/api/public", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"Hello from a public endpoint!"}`))
	})

	// Rutas de artículos
	router.HandleFunc("/api/articulos", handleGetArticulos)
	router.HandleFunc("/api/articulos/", handleGetArticuloPorID)
	router.HandleFunc("/api/articulos/buscar", handleBuscarArticulos)
	router.Handle("/api/articulos/agregar", middleware.EnsureValidToken()(http.HandlerFunc(handleAgregarArticulo)))
	router.Handle("/api/articulos/actualizar/", middleware.EnsureValidToken()(http.HandlerFunc(handleActualizarArticulo)))
	router.Handle("/api/articulos/eliminar/", middleware.EnsureValidToken()(http.HandlerFunc(handleEliminarArticulo)))

	// Rutas de categorías
	router.HandleFunc("/api/categorias", handleGetCategorias)
	router.Handle("/api/categorias/agregar", middleware.EnsureValidToken()(http.HandlerFunc(handleAgregarCategoria)))
	router.Handle("/api/categorias/actualizar/", middleware.EnsureValidToken()(http.HandlerFunc(handleActualizarCategoria)))
	router.Handle("/api/categorias/eliminar/", middleware.EnsureValidToken()(http.HandlerFunc(handleEliminarCategoria)))

	// Inventario
	router.Handle("/api/inventario", middleware.EnsureValidToken()(http.HandlerFunc(handleReporteInventario)))
	router.Handle("/api/inventario/obtener_tomas", middleware.EnsureValidToken()(http.HandlerFunc(handleObtenerInventarios)))
	router.Handle("/api/inventario/crear_tomas", middleware.EnsureValidToken()(http.HandlerFunc(handleCrearTomaFisica)))
	router.Handle("/api/inventario/guardar_tomas", middleware.EnsureValidToken()(http.HandlerFunc(handleGuardarTomaDetalle)))
	router.Handle("/api/inventario/cancelar_tomas/", middleware.EnsureValidToken()(http.HandlerFunc(handleCancelarToma)))
	router.Handle("/api/inventario/detalles_tomas/", middleware.EnsureValidToken()(http.HandlerFunc(handleObtenerDetalleToma)))

	// Movimientos
	router.Handle("/api/movimientos/registrar", middleware.EnsureValidToken()(http.HandlerFunc(handleRegistrarMovimiento)))
	router.Handle("/api/movimientos", middleware.EnsureValidToken()(http.HandlerFunc(handleReporteMovimientos)))
	router.Handle("/api/movimientos/editar", middleware.EnsureValidToken()(http.HandlerFunc(handleEditarMovimiento)))
	router.Handle("/api/movimientos/eliminar", middleware.EnsureValidToken()(http.HandlerFunc(handleEliminarMovimiento)))

	// Compras
	router.Handle("/api/compras/registrar", middleware.EnsureValidToken()(http.HandlerFunc(handleRegistrarCompra)))
	router.Handle("/api/compras", middleware.EnsureValidToken()(http.HandlerFunc(handleObtenerComprasResumen)))
	router.Handle("/api/compras/", middleware.EnsureValidToken()(http.HandlerFunc(handleDetalleCompra)))
	router.Handle("/api/compras/editar", middleware.EnsureValidToken()(http.HandlerFunc(handleEditarCompra)))
	router.Handle("/api/compras/eliminar", middleware.EnsureValidToken()(http.HandlerFunc(handleEliminarCompra)))

	fmt.Println("Servidor escuchando en http://0.0.0.0:3010")
	if err := http.ListenAndServe("0.0.0.0:3010", corsHandler(router)); err != nil {
		log.Fatalf("Error en el servidor HTTP: %v", err)
	}
}
