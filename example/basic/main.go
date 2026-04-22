package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/andryhardiyanto/go-postgres"
	"github.com/joho/godotenv"
)

type User struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	Email     string    `db:"email"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	// Initialize the connection
	db, err := postgres.New(
		postgres.WithDsn(os.Getenv("DATABASE_URL")),
		postgres.WithDriverName("postgres"),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	setupDatabase(ctx, db)

	fmt.Println("=== Basic Exec & Select ===")

	// 1. Exec (Insert)
	fmt.Println("Inserting new user...")
	result, err := db.
		Insert("INSERT INTO users (name, email, status) VALUES (:name, :email, :status) RETURNING id",
			"name", "John Doe",
			"email", "john@example.com",
			"status", "active").
		Insert("INSERT INTO users (name, email, status) VALUES (:name, :email, :status) RETURNING id",
			"name", "John Doe 2",
			"email", "john2@example.com",
			"status", "active").
		Insert("INSERT INTO users (name, email, status) VALUES (:name, :email, :status) RETURNING id",
			"name", "John Doe 3",
			"email", "john3@example.com",
			"status", "inactive").
		Insert("INSERT INTO users (name, email, status) VALUES (:name, :email, :status) RETURNING id",
			"name", "John Doe 4",
			"email", "john4@example.com",
			"status", "active").
		Insert("INSERT INTO users (name, email, status) VALUES (:name, :email, :status) RETURNING id",
			"name", "John Doe 5",
			"email", "john5@example.com",
			"status", "inactive").
		Insert("INSERT INTO users (name, email, status) VALUES (:name, :email, :status) RETURNING id",
			"name", "John Doe 6",
			"email", "john6@example.com",
			"status", "active").
		ExecInTx(ctx)
	if err != nil {
		fmt.Printf("Insert failed: %v\n", err)
	} else {
		fmt.Printf("Inserted User ID: %v\n", result)
	}

	// 2. Select One
	var user User
	found, err := db.Select("SELECT * FROM users WHERE status = :status ORDER BY id DESC LIMIT 1", &user, "status", "active").One(ctx)
	if err != nil {
		fmt.Printf("Select One error: %v\n", err)
	} else if found {
		fmt.Printf("Found latest active user: %v (%v)\n", user.Name, user.Email)
	}

	// 3. Select Many
	var users []User
	found, err = db.Select("SELECT * FROM users WHERE status = :status", &users, "status", "active").Many(ctx)
	if err != nil {
		fmt.Printf("Select Many error: %v\n", err)
	} else if found {
		fmt.Printf("Found %d active users\n", len(users))
	}
}

func setupDatabase(ctx context.Context, db postgres.Postgres) {
	fmt.Println("Setting up tables...")

	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		email VARCHAR(100) NOT NULL,
		status VARCHAR(20) DEFAULT 'active',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Update(usersTable).Exec(ctx)
	if err != nil {
		log.Fatalf("Failed to create users table: %v", err)
	}
}
