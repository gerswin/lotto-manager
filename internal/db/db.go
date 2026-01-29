package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init(dataSourceName string) error {
	var err error
	DB, err = sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return err
	}

	if err = DB.Ping(); err != nil {
		return err
	}

	return createTables()
}

func createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		telegram_id INTEGER UNIQUE,
		name TEXT NOT NULL,
		phone TEXT
	);

	CREATE TABLE IF NOT EXISTS raffles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		total_numbers INTEGER NOT NULL,
		ticket_price REAL NOT NULL,
		reserve_hours INTEGER DEFAULT 24,
		status TEXT DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tickets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		raffle_id INTEGER,
		number TEXT NOT NULL,
		user_id INTEGER,
		status TEXT DEFAULT 'available',
		reserved_at DATETIME,
		FOREIGN KEY(raffle_id) REFERENCES raffles(id),
		FOREIGN KEY(user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS payments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ticket_id INTEGER,
		amount REAL NOT NULL,
		method TEXT,
		reference TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		is_verified BOOLEAN DEFAULT 0,
		FOREIGN KEY(ticket_id) REFERENCES tickets(id)
	);
	`

	_, err := DB.Exec(query)
	if err != nil {
		log.Printf("Error creating tables: %v", err)
		return err
	}

	return nil
}
