package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"lotto-tg-app/internal/db"
	"lotto-tg-app/internal/models"
	"lotto-tg-app/internal/services"
)

// Helper to render templates
func render(w http.ResponseWriter, tmpl string, data interface{}) {
	t, err := template.ParseFiles(
		"web/templates/layout.html",
		"web/templates/"+tmpl,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println("Template error:", err)
		return
	}
	
	if err := t.Execute(w, data); err != nil {
		log.Println("Template execute error:", err)
	}
}

// Home Handler - Shows list of raffles or the selected raffle grid
func Home(w http.ResponseWriter, r *http.Request) {
	raffleIDParam := r.URL.Query().Get("id")

	// IF no ID is provided, show the list of ACTIVE raffles
	if raffleIDParam == "" {
	
rows, err := db.DB.Query("SELECT id, name, total_numbers, ticket_price FROM raffles WHERE status = 'active' ORDER BY created_at DESC")
		if err != nil {
			http.Error(w, "DB Error", 500)
			return
		}
		defer rows.Close()

		var raffles []models.Raffle
		for rows.Next() {
			var raf models.Raffle
		
rows.Scan(&raf.ID, &raf.Name, &raf.TotalNumbers, &raf.TicketPrice)
			raffles = append(raffles, raf)
		}

		        if len(raffles) == 0 {
		            w.Header().Set("Content-Type", "text/html; charset=utf-8")
		            fmt.Fprint(w, `
		                <div style="font-family: sans-serif; text-align: center; padding: 50px;">
		                    <h1>⏳ No hay sorteos activos</h1>
		                    <p>Pronto anunciaremos nuevas rifas.</p>
		                </div>
		            `)
		            return
		        }
		// IF only one raffle, just show it directly (optional, but better UX)
		if len(raffles) == 1 {
			http.Redirect(w, r, "/?id="+strconv.FormatInt(raffles[0].ID, 10), http.StatusSeeOther)
			return
		}

		// Show List Template (we'll create it now)
		data := struct {
			Title      string
			RaffleName string
			Raffles    []models.Raffle
		}{
			Title:      "Sorteos Disponibles",
			RaffleName: "Elige tu Rifa",
			Raffles:    raffles,
		}
		render(w, "raffle_list.html", data)
		return
	}

	// IF ID is provided, show that specific raffle
	id, err := strconv.ParseInt(raffleIDParam, 10, 64)
	if err != nil {
		http.Error(w, "ID de sorteo inválida", 400)
		return
	}

	var raffle models.Raffle
	err = db.DB.QueryRow("SELECT id, name, total_numbers, ticket_price FROM raffles WHERE id = ?", id).Scan(&raffle.ID, &raffle.Name, &raffle.TotalNumbers, &raffle.TicketPrice)
	if err == sql.ErrNoRows {
		http.Error(w, "Sorteo no encontrado", 404)
		return
	}

	tickets, err := getTickets(raffle.ID, "")
	if err != nil {
		log.Printf("Error fetching tickets for raffle %d: %v", raffle.ID, err)
		http.Error(w, "Error cargando tickets", 500)
		return
	}

	log.Printf("Raffle %d (%s) loaded with %d tickets", raffle.ID, raffle.Name, len(tickets))

	data := struct {
		Title      string
		RaffleName string
		RaffleID   int64
		Tickets    []models.Ticket
	}{
		Title:      "Lotería - " + raffle.Name,
		RaffleName: raffle.Name,
		RaffleID:   raffle.ID,
		Tickets:    tickets,
	}

	render(w, "index.html", data)
}

// Search Handler (HTMX) - Now needs to know which raffle to search in
func SearchTickets(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	raffleID, _ := strconv.ParseInt(r.URL.Query().Get("raffle_id"), 10, 64)

	tickets, _ := getTickets(raffleID, query)

	data := struct {
		Tickets  []models.Ticket
		RaffleID int64
	}{
		Tickets:  tickets,
		RaffleID: raffleID,
	}

	t, _ := template.ParseFiles("web/templates/index.html")
	if err := t.ExecuteTemplate(w, "grid", data); err != nil {
		log.Println("Search Template Error:", err)
	}
}

// Get Book Modal - Needs raffle ID implicitly from ticket or explicitly
func GetBookModal(w http.ResponseWriter, r *http.Request) {
	number := chi.URLParam(r, "number")
	raffleID, _ := strconv.ParseInt(r.URL.Query().Get("raffle_id"), 10, 64)
	
	var ticket models.Ticket
	var raffle models.Raffle
	
	err := db.DB.QueryRow(`
		SELECT t.id, t.number, t.status, r.id, r.ticket_price 
		FROM tickets t 
		JOIN raffles r ON t.raffle_id = r.id 
		WHERE t.number = ? AND r.id = ?`, number, raffleID).Scan(&ticket.ID, &ticket.Number, &ticket.Status, &raffle.ID, &raffle.TicketPrice)

	if err != nil {
		http.Error(w, "Ticket not found", 404)
		return
	}

	data := struct {
		Ticket models.Ticket
		Raffle models.Raffle
	}{ticket, raffle}

	t, _ := template.ParseFiles("web/templates/book_modal.html")
	t.Execute(w, data)
}

// Process Booking
func PostBook(w http.ResponseWriter, r *http.Request) {
	number := chi.URLParam(r, "number")
	raffleID, _ := strconv.ParseInt(r.URL.Query().Get("raffle_id"), 10, 64)
	r.ParseForm()
	
	name := r.FormValue("name")
	phone := r.FormValue("phone")
	method := r.FormValue("method")
	ref := r.FormValue("reference")
	amountStr := r.FormValue("amount")
	amount, _ := strconv.ParseFloat(amountStr, 64)

	tx, _ := db.DB.Begin()

	var ticketID int64
	err := tx.QueryRow("SELECT id FROM tickets WHERE number = ? AND raffle_id = ? AND status = 'available'", number, raffleID).Scan(&ticketID)
	if err != nil {
		tx.Rollback()
		http.Error(w, "Ticket no disponible", 400)
		return
	}

	res, _ := tx.Exec("INSERT INTO users (name, phone) VALUES (?, ?)", name, phone)
	userID, _ := res.LastInsertId()

	_, err = tx.Exec("UPDATE tickets SET user_id = ?, status = 'reserved', reserved_at = CURRENT_TIMESTAMP WHERE id = ?", userID, ticketID)
	_, err = tx.Exec("INSERT INTO payments (ticket_id, amount, method, reference) VALUES (?, ?, ?, ?)", ticketID, amount, method, ref)

	if err != nil {
		tx.Rollback()
		http.Error(w, "Error saving", 500)
		return
	}

	tx.Commit()

	// 5. Notify Admin via Telegram
	notificationText := fmt.Sprintf("🎟️ *Nueva Reserva: #%s*\n👤 Cliente: %s\n📞 Telf: %s\n💰 Monto: $%v\n💳 Ref: %s\n\n_Rifa ID: %d_", 
		number, name, phone, amount, ref, raffleID)
	services.NotifyAdmin(notificationText)
	
	// HTMX: Tell the client to refresh the grid
	w.Header().Set("HX-Trigger", "ticketBooked")
	w.WriteHeader(http.StatusOK)
}

// Helper to fetch tickets
func getTickets(raffleID int64, query string) ([]models.Ticket, error) {
	sqlQuery := "SELECT number, status FROM tickets WHERE raffle_id = ?"
	args := []interface{}{raffleID}

	if query != "" {
		sqlQuery += " AND number LIKE ?"
		args = append(args, "%"+query+"%")
	}
	
	sqlQuery += " ORDER BY number ASC"


rows, err := db.DB.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []models.Ticket
	for rows.Next() {
		var t models.Ticket
		rows.Scan(&t.Number, &t.Status)
		tickets = append(tickets, t)
	}
	return tickets, nil
}
