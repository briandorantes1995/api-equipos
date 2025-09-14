package main

import (
	"log"
	"net/http"
	"os"

	"github.com/nedpals/supabase-go"

	"equiposmedicos/middleware"

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
	FRONT_END2 := os.Getenv("FRONT_END2")
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
			origin := r.Header.Get("Origin")
			if origin == FRONT_END || origin == FRONT_END2 {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

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

	router.Handle("/api/inventario", middleware.EnsureValidToken()(http.HandlerFunc(handleReporteInventario)))

	router.Handle("/api/movimientos/registrar", middleware.EnsureValidToken()(http.HandlerFunc(handleRegistrarMovimiento)))
	router.Handle("/api/movimientos", middleware.EnsureValidToken()(http.HandlerFunc(handleReporteMovimientos)))
	router.Handle("/api/movimientos/editar", middleware.EnsureValidToken()(http.HandlerFunc(handleEditarMovimiento)))
	router.Handle("/api/movimientos/eliminar", middleware.EnsureValidToken()(http.HandlerFunc(handleEliminarMovimiento)))

	router.Handle("/api/compras/registrar", middleware.EnsureValidToken()(http.HandlerFunc(handleRegistrarCompra)))
	router.Handle("/api/compras", middleware.EnsureValidToken()(http.HandlerFunc(handleObtenerComprasResumen)))
	router.Handle("/api/compras/", middleware.EnsureValidToken()(http.HandlerFunc(handleDetalleCompra)))
	router.Handle("/api/compras/editar", middleware.EnsureValidToken()(http.HandlerFunc(handleEditarCompra)))
	router.Handle("/api/compras/eliminar", middleware.EnsureValidToken()(http.HandlerFunc(handleEliminarCompra)))

	log.Print("Server listening on http://0.0.0.0:3010")
	if err := http.ListenAndServe("0.0.0.0:3010", corsHandler(router)); err != nil {
		log.Fatalf("There was an error with the http server: %v", err)
	}

}
