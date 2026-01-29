package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"lotto-tg-app/internal/db"
	"lotto-tg-app/internal/models"
)

// AdminLogin muestra página que captura initData de Telegram y redirige al admin
func AdminLogin(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Admin Login</title>
	<script src="https://telegram.org/js/telegram-web-app.js"></script>
	<style>
		body {
			font-family: -apple-system, sans-serif;
			display: flex;
			justify-content: center;
			align-items: center;
			height: 100vh;
			margin: 0;
			background: #f0f0f0;
		}
		.loader {
			text-align: center;
			padding: 40px;
			background: white;
			border-radius: 16px;
			box-shadow: 0 4px 20px rgba(0,0,0,0.1);
		}
		.spinner {
			width: 40px;
			height: 40px;
			border: 4px solid #e0e0e0;
			border-top-color: #3b82f6;
			border-radius: 50%;
			animation: spin 1s linear infinite;
			margin: 0 auto 16px;
		}
		@keyframes spin { to { transform: rotate(360deg); } }
		.error { color: #ef4444; margin-top: 16px; }
	</style>
</head>
<body>
	<div class="loader">
		<div class="spinner"></div>
		<p>Autenticando...</p>
		<p id="error" class="error" style="display:none;"></p>
	</div>
	<script>
		const tg = window.Telegram.WebApp;
		tg.ready();
		tg.expand();

		if (tg.initData) {
			document.cookie = "tg_init_data=" + encodeURIComponent(tg.initData) + "; path=/; SameSite=Lax; max-age=86400";
			setTimeout(() => {
				window.location.href = "/admin";
			}, 500);
		} else {
			document.getElementById("error").style.display = "block";
			document.getElementById("error").innerText = "No se detectó Telegram. Abre desde la Mini App.";
		}
	</script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

// Data structure for Admin Dashboard
type AdminData struct {
	Title          string
	RaffleName     string
	ActiveRaffles  []models.Raffle // For the dropdown
	SelectedRaffleID int64
	TotalCollected float64
	PendingAmount  float64
	SoldCount      int
	TotalTickets   int
	Tickets        []models.Ticket
}

func AdminSearchUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) < 2 {
		json.NewEncoder(w).Encode([]models.User{})
		return
	}

	rows, err := db.DB.Query("SELECT id, name, phone FROM users WHERE name LIKE ? OR phone LIKE ? LIMIT 5", "%"+q+"%", "%"+q+"%")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		rows.Scan(&u.ID, &u.Name, &u.Phone)
		users = append(users, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func AdminDashboard(w http.ResponseWriter, r *http.Request) {
	selectedID, _ := strconv.ParseInt(r.URL.Query().Get("raffle_id"), 10, 64)

	// 1. Get List of all active raffles for the selector
	rafRows, _ := db.DB.Query("SELECT id, name FROM raffles WHERE status = 'active'")
	var activeRaffles []models.Raffle
	for rafRows.Next() {
		var raf models.Raffle
		rafRows.Scan(&raf.ID, &raf.Name)
		activeRaffles = append(activeRaffles, raf)
	}
	rafRows.Close()

	// If no raffle selected but there are active ones, pick the first one
	if selectedID == 0 && len(activeRaffles) > 0 {
		selectedID = activeRaffles[0].ID
	}

	// 2. Get Stats & Tickets for the SELECTED raffle
	var tickets []models.Ticket
	var totalCollected, pending float64
	var soldCount int
	var raffleName string = "Sin Sorteo Seleccionado"
	var totalTickets int

	if selectedID > 0 {
		rows, err := db.DB.Query(`
			SELECT 
				t.id, t.number, t.status, 
				COALESCE(u.name, 'Anon'), COALESCE(u.phone, ''),
				COALESCE((SELECT SUM(amount) FROM payments WHERE ticket_id = t.id), 0) as paid,
				r.ticket_price,
				r.name
			FROM tickets t
			LEFT JOIN users u ON t.user_id = u.id
			JOIN raffles r ON t.raffle_id = r.id
			WHERE r.id = ?
			ORDER BY t.number ASC
		`, selectedID)
		
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var t models.Ticket
				var price float64
				rows.Scan(&t.ID, &t.Number, &t.Status, &t.UserName, &t.UserPhone, &t.TotalPaid, &price, &raffleName)
				
				t.Remaining = price - t.TotalPaid
				if t.Status != "available" {
					totalCollected += t.TotalPaid
					pending += t.Remaining
					soldCount++
				}
				tickets = append(tickets, t)
			}
			totalTickets = len(tickets)
		}
	}

	data := AdminData{
		Title:            "Admin Panel",
		RaffleName:       raffleName,
		ActiveRaffles:    activeRaffles,
		SelectedRaffleID: selectedID,
		TotalCollected:   totalCollected,
		PendingAmount:    pending,
		SoldCount:        soldCount,
		TotalTickets:     totalTickets, 
		Tickets:          tickets,
	}

	// Custom template parsing to include functions
	funcMap := template.FuncMap{
		"add": func(a, b float64) float64 { return a + b },
	}

	// Use the base filename as the template name
	t, err := template.New("layout.html").Funcs(funcMap).ParseFiles(
		"web/templates/layout.html",
		"web/templates/admin.html",
	)
	if err != nil {
		log.Printf("Error parsing admin templates: %v", err)
		http.Error(w, "Template Parse Error", 500)
		return
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("Error executing admin template: %v", err)
		http.Error(w, "Template Exec Error", 500)
	}
}

// AdminGetTicketDetails returns JSON with ticket, user and payment info
func AdminGetTicketDetails(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "id")

	var data struct {
		Ticket   models.Ticket    `json:"ticket"`
		User     models.User      `json:"user"`
		Payments []models.Payment `json:"payments"`
		Price    float64          `json:"price"`
	}

	// 1. Get Ticket & Price
	err := db.DB.QueryRow(`
		SELECT t.id, t.number, t.status, r.ticket_price, u.id, u.name, u.phone
		FROM tickets t
		JOIN raffles r ON t.raffle_id = r.id
		LEFT JOIN users u ON t.user_id = u.id
		WHERE t.id = ?`, ticketID).Scan(
		&data.Ticket.ID, &data.Ticket.Number, &data.Ticket.Status, &data.Price,
		&data.User.ID, &data.User.Name, &data.User.Phone,
	)
	if err != nil {
		http.Error(w, "Ticket not found", 404)
		return
	}

	// 2. Get Payments
	rows, _ := db.DB.Query("SELECT amount, method, reference, is_verified, created_at FROM payments WHERE ticket_id = ? ORDER BY created_at DESC", ticketID)
	defer rows.Close()
	for rows.Next() {
		var p models.Payment
		rows.Scan(&p.Amount, &p.Method, &p.Reference, &p.IsVerified, &p.CreatedAt)
		data.Payments = append(data.Payments, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func AdminCreateRaffle(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := r.FormValue("name")
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	raffleType := r.FormValue("type") // "terminal" or "triple"

	totalNumbers := 100
	format := "%02d" // 00-99

	if raffleType == "triple" {
		totalNumbers = 1000
		format = "%03d" // 000-999
	}

	// Transaction
	tx, _ := db.DB.Begin()

	// REMOVED: Automatic archive of old raffles.
	// Now we can have multiple active raffles.

	// Create Raffle
	res, err := tx.Exec("INSERT INTO raffles (name, total_numbers, ticket_price) VALUES (?, ?, ?)", name, totalNumbers, price)
	if err != nil {
		tx.Rollback()
		http.Error(w, "Error creando sorteo: "+err.Error(), 500)
		return
	}
	raffleID, _ := res.LastInsertId()

	// Generate Tickets
	stmt, _ := tx.Prepare("INSERT INTO tickets (raffle_id, number) VALUES (?, ?)")
	for i := 0; i < totalNumbers; i++ {
		num := fmt.Sprintf(format, i)
		stmt.Exec(raffleID, num)
	}
	stmt.Close()

	if err := tx.Commit(); err != nil {
		http.Error(w, "Error finalizando transacción", 500)
		return
	}

	// Redirect back to dashboard
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func AdminAddPayment(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "id")
	r.ParseForm()
	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	method := r.FormValue("method")
	ref := r.FormValue("reference")
	name := r.FormValue("name")
	phone := r.FormValue("phone")

	tx, _ := db.DB.Begin()

	// 1. If ticket is available, we need to assign a user first
	var status string
	var userID int64
	err := tx.QueryRow("SELECT status, user_id FROM tickets WHERE id = ?", ticketID).Scan(&status, &userID)

	if status == "available" {
		// Create or find user (simple create for now)
		res, _ := tx.Exec("INSERT INTO users (name, phone) VALUES (?, ?)", name, phone)
		userID, _ = res.LastInsertId()
		tx.Exec("UPDATE tickets SET user_id = ?, status = 'reserved', reserved_at = CURRENT_TIMESTAMP WHERE id = ?", userID, ticketID)
	}

	// 2. Insert Payment
	_, err = tx.Exec("INSERT INTO payments (ticket_id, amount, method, reference, is_verified) VALUES (?, ?, ?, ?, 1)", ticketID, amount, method, ref)
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), 500)
		return
	}

	// 3. Check if fully paid
	var totalPaid, price float64
	tx.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM payments WHERE ticket_id = ?", ticketID).Scan(&totalPaid)
	tx.QueryRow("SELECT r.ticket_price FROM tickets t JOIN raffles r ON t.raffle_id = r.id WHERE t.id = ?", ticketID).Scan(&price)

	if totalPaid >= price {
		tx.Exec("UPDATE tickets SET status = 'paid' WHERE id = ?", ticketID)
	}

	tx.Commit()
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func AdminReleaseTicket(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "id")
	
	// Reset ticket
	tx, _ := db.DB.Begin()
	tx.Exec("DELETE FROM payments WHERE ticket_id = ?", ticketID)
	tx.Exec("UPDATE tickets SET user_id = NULL, status = 'available', reserved_at = NULL WHERE id = ?", ticketID)
		tx.Commit()
	
		// Return simple success text. If hx-target is "closest tr", the row disappears.
		// If hx-swap is "none", nothing happens except the after-request trigger.
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}
	