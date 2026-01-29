package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"lotto-tg-app/internal/db"
	"lotto-tg-app/internal/handlers"
	"lotto-tg-app/internal/services"
)

func main() {
	// 0. Load Config (Envars)
	_ = godotenv.Load() // Load .env file if exists

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Println("Warning: TELEGRAM_TOKEN not set. Bot features disabled.")
	}
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 1. Init Database (Turso)
	dbURL := os.Getenv("TURSO_DATABASE_URL")
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	if dbURL == "" || authToken == "" {
		log.Fatal("TURSO_DATABASE_URL and TURSO_AUTH_TOKEN must be set")
	}

	if err := db.Init(dbURL, authToken); err != nil {
		log.Fatal("Failed to init DB:", err)
	}
	defer db.DB.Close()
	log.Println("Database initialized with Turso")

	// 2. Init Telegram Bot
	if token != "" {
		if err := services.InitBot(token); err != nil {
			log.Printf("Warning: Failed to init Telegram bot: %v", err)
		}
	}

	// 3. Setup Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// 4. Static Files
	workDir, _ := os.Getwd()
	filesDir := filepath.Join(workDir, "web/assets")
	FileServer(r, "/assets", http.Dir(filesDir))

	// 5. Public Routes
	r.Get("/", handlers.Home)
	r.Get("/tickets/search", handlers.SearchTickets)
	r.Get("/tickets/{number}/book", handlers.GetBookModal)
	r.Post("/tickets/{number}/book", handlers.PostBook)

	// 6. Admin Routes (Protected)
	r.Group(func(r chi.Router) {
		r.Use(middleware.BasicAuth("Lotto Admin", map[string]string{
			"admin": os.Getenv("ADMIN_PASSWORD"), // User: admin
		}))
						r.Get("/admin", handlers.AdminDashboard)
						r.Get("/admin/users/search", handlers.AdminSearchUsers) // New
						r.Get("/admin/tickets/{id}/details", handlers.AdminGetTicketDetails)
				 // New
				r.Post("/admin/raffles", handlers.AdminCreateRaffle)
		 // New Route
		r.Post("/admin/tickets/{id}/payment", handlers.AdminAddPayment)
		r.Post("/admin/tickets/{id}/release", handlers.AdminReleaseTicket)
	})

	// 7. Start
	fmt.Printf("Servidor corriendo en http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

// FileServer conveniently sets up a http.FileServer handler at a specific path.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings := len(path) - 1; path[strings] != '/' {
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
