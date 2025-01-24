package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"            // Driver para PostgreSQL
	"github.com/joho/godotenv"       // Biblioteca para cargar variables de entorno
	"github.com/fsnotify/fsnotify"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

var (
	watcher     *fsnotify.Watcher
	eventStream chan string
	db          *sql.DB // Conexión a la base de datos
)

// Inicializar la conexión a PostgreSQL
func initDB() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Error pinging the database: %v", err)
	}

	log.Println("Connected to the database successfully!")
}

// Guardar eventos en la base de datos
func saveEventToDB(eventType, fileName, filePath, lastModified string) {
	query := `
		INSERT INTO directory_events (event_type, file_name, file_path, last_modified, event_time)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Exec(query, eventType, fileName, filePath, lastModified, time.Now())
	if err != nil {
		log.Printf("Error saving event to database: %v", err)
	}
}

// Monitorear un directorio y todos sus subdirectorios
func watchDirectoryRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Si es un directorio, añadirlo al watcher
		if info.IsDir() {
			err = watcher.Add(path)
			if err != nil {
				log.Printf("Error adding directory to watcher: %v", err)
			} else {
				log.Printf("Watching directory: %s", path)
			}
		}
		return nil
	})
}

// Función para monitorear eventos
func watchEvents() {
	for {
		select {
		case event := <-watcher.Events:
			now := time.Now().Format("2006-01-02 15:04:05")
			action := event.Op.String()
			file := event.Name

			// Obtener información del archivo
			modTime := "Unknown"
			if fileInfo, err := os.Stat(file); err == nil {
				modTime = fileInfo.ModTime().Format("2006-01-02 15:04:05")
			}

			// Guardar en la base de datos
			saveEventToDB(action, filepath.Base(file), file, modTime)

			// Si se crea un directorio, añadirlo al watcher dinámicamente
			if event.Op&fsnotify.Create == fsnotify.Create {
				if fileInfo, err := os.Stat(file); err == nil && fileInfo.IsDir() {
					watchDirectoryRecursive(file) // Añadir el nuevo directorio y sus subdirectorios
				}
			}

			// Enviar el evento al cliente
			eventDetails := fmt.Sprintf("[%s] %s %q (Last Modified: %s)", now, action, file, modTime)
			log.Println("Sending event:", eventDetails)
			eventStream <- eventDetails

		case err := <-watcher.Errors:
			log.Println("Watcher error:", err)
		}
	}
}

func main() {
	// Inicializar la conexión a la base de datos
	initDB()
	defer db.Close()

	// Inicializar el watcher
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error creating watcher: %v", err)
	}
	defer watcher.Close()

	// Canal para transmitir eventos
	eventStream = make(chan string)

	// Crear la aplicación Fiber
	app := fiber.New()
	app.Use(logger.New())

	// Servir archivos estáticos
	app.Static("/", "./static")

	// Ruta para iniciar el monitoreo
	app.Post("/watch", func(c *fiber.Ctx) error {
		body := struct {
			Dir string `json:"dir"`
		}{}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).SendString("Invalid request body")
		}

		dirToWatch := body.Dir
		if _, err := os.Stat(dirToWatch); os.IsNotExist(err) {
			return c.Status(400).SendString("Directory does not exist")
		}

		err = watchDirectoryRecursive(dirToWatch)
		if err != nil {
			return c.Status(500).SendString("Error adding directory and subdirectories to watcher")
		}

		go watchEvents()
		return c.SendString("Watching directory and subdirectories: " + dirToWatch)
	})

	// Ruta para enviar eventos SSE
	app.Get("/events", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/event-stream")
		reader, writer := io.Pipe()
		c.Context().SetBodyStream(reader, -1)
		go func() {
			defer writer.Close()
			for event := range eventStream {
				writer.Write([]byte("data: " + event + "\n\n"))
			}
		}()
		return nil
	})

	// Iniciar el servidor
	log.Println("Server is running on http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}
