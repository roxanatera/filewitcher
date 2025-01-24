# Etapa 1: Construcción de la aplicación
FROM golang:1.20 AS builder

# Establecer el directorio de trabajo dentro del contenedor
WORKDIR /app

# Copiar los archivos de la aplicación al contenedor
COPY . .

# Descargar dependencias y compilar la aplicación
RUN CGO_ENABLED=0 GOOS=linux go build -a -o filewatcher main.go

# Etapa 2: Imagen ligera para ejecutar la aplicación
FROM alpine:latest

# Instalar dependencias necesarias para la ejecución
RUN apk --no-cache add ca-certificates

# Establecer el directorio de trabajo
WORKDIR /root/

# Copiar el ejecutable desde la etapa de construcción
COPY --from=builder /app/filewatcher .
COPY --from=builder /app/static ./static

# Exponer el puerto 3000
EXPOSE 3000

# Comando para ejecutar la aplicación
CMD ["./filewatcher"]
