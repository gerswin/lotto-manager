package models

import (
	"time"
)

// User represents a customer
type User struct {
	ID         int64  `json:"id"`
	TelegramID *int64 `json:"telegram_id"` // Pointer allowing null
	Name       string `json:"name"`
	Phone      string `json:"phone"`
}

// Raffle represents a lottery event
type Raffle struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	TotalNumbers  int       `json:"total_numbers"`
	TicketPrice   float64   `json:"ticket_price"`
	ReserveHours  int       `json:"reserve_hours"`
	Status        string    `json:"status"` // 'active', 'finished'
	CreatedAt     time.Time `json:"created_at"`
}

// Ticket represents a single lottery number
type Ticket struct {
	ID          int64   `json:"id"`
	RaffleID    int64   `json:"raffle_id"`
	Number      string  `json:"number"`
	UserID      *int64  `json:"user_id"` // Pointer allowing null (if available)
	Status      string  `json:"status"`  // 'available', 'reserved', 'paid'
	ReservedAt  *time.Time `json:"reserved_at"`
	
	// Virtual fields (calculated via joins/queries)
	UserName    string  `json:"user_name,omitempty"`
	UserPhone   string  `json:"user_phone,omitempty"`
	TotalPaid   float64 `json:"total_paid"`
	Remaining   float64 `json:"remaining"`
}

// Payment represents a transaction for a ticket
type Payment struct {
	ID          int64     `json:"id"`
	TicketID    int64     `json:"ticket_id"`
	Amount      float64   `json:"amount"`
	Method      string    `json:"method"` // 'cash', 'transfer'
	Reference   string    `json:"reference"`
	CreatedAt   time.Time `json:"created_at"`
	IsVerified  bool      `json:"is_verified"`
}
