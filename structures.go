package main

import "time"

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
	ID                  int        `json:"id"`
	Nombre              string     `json:"nombre"`
	CantidadActual      float64    `json:"cantidad_actual"`
	UltimaActualizacion *time.Time `json:"ultima_actualizacion"`
}

type Movimiento struct {
	ArticuloID     int     `json:"articulo_id"`
	TipoMovimiento string  `json:"tipo_movimiento"`
	Cantidad       float64 `json:"cantidad"`
	Motivo         string  `json:"motivo"`
	UsuarioNombre  string  `json:"usuario_nombre"`
	Fecha          string  `json:"fecha"`
}

type MovimientoConNombre struct {
	ID                int     `json:"id"`
	ArticuloID        int     `json:"articulo_id"`
	NombreArticulo    string  `json:"nombre_articulo"`
	ProveedorArticulo string  `json:"proveedor_articulo"` // nueva columna
	TipoMovimiento    string  `json:"tipo_movimiento"`
	Cantidad          float64 `json:"cantidad"`
	Motivo            string  `json:"motivo"`
	Fecha             string  `json:"fecha"`
	UsuarioNombre     string  `json:"usuario_nombre"`
}

type CategoryDetail struct {
	Nombre string `json:"nombre,omitempty"`
}
