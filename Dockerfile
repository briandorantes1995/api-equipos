# Etapa de construcción
FROM golang:1.24.4-bullseye AS builder
WORKDIR /app

# Copiar solo go.mod y go.sum primero para aprovechar cache de dependencias
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar el resto del código
COPY . .

# Compilar el binario estático para Linux
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o go-api .

# Etapa de ejecución
FROM debian:bullseye-slim
WORKDIR /app

# Instalar certificados para HTTPS (muy importante para llamadas TLS)
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/go-api .

ENV PORT=3010
EXPOSE 3010

CMD ["./go-api"]


