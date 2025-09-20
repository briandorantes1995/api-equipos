package main

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
	CategoriaNombre string  `json:"categoria_nombre,omitempty"`
	Marca           string  `json:"marca,omitempty"`
}

type InventarioArticulo struct {
	ID                  int     `json:"id"`
	Nombre              string  `json:"nombre"`
	PrecioVenta         float64 `json:"precio_venta"`
	Costo               float64 `json:"costo"`
	Proveedor           string  `json:"proveedor"`
	CodigoBarras        string  `json:"codigo_barras"`
	CantidadActual      float64 `json:"cantidad_actual"`
	UltimaActualizacion string  `json:"ultima_actualizacion"`
	Marca               string  `json:"marca,omitempty"`
}

type InventarioMovimientoArticulo struct {
	ID                  int     `json:"id"`
	ArticuloID          int     `json:"articulo_id"`
	CantidadActual      float64 `json:"cantidad_actual"`
	UltimaActualizacion string  `json:"ultima_actualizacion"`
}

type Movimiento struct {
	ArticuloID     int     `json:"articulo_id"`
	TipoMovimiento string  `json:"tipo_movimiento"`
	Cantidad       float64 `json:"cantidad"`
	Motivo         string  `json:"motivo"`
	UsuarioNombre  string  `json:"usuario_nombre"`
	Fecha          string  `json:"fecha"`
}

type MovimientoEditar struct {
	ID             int     `json:"id"`              // ID del movimiento a editar
	Cantidad       float64 `json:"cantidad"`        // Nueva cantidad
	TipoMovimiento string  `json:"tipo_movimiento"` // Nuevo tipo de movimiento
	Motivo         string  `json:"motivo"`          // Nuevo motivo
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

type TomaInventario struct {
	ID              int    `json:"id,omitempty"`
	Folio           int    `json:"folio,omitempty"`
	FechaInicio     string `json:"fecha_inicio,omitempty"`
	FechaFin        string `json:"fecha_fin,omitempty"`
	UsuarioAuth0Sub string `json:"usuario_auth0_sub,omitempty"`
	UsuarioCorreo   string `json:"usuario_correo,omitempty"`
	Estado          string `json:"estado,omitempty"`
	CategoriaID     *int   `json:"categoria_id,omitempty"`
	CategoriaNombre string `json:"categoria_nombre,omitempty"`
	Observaciones   string `json:"observaciones,omitempty"`
}
