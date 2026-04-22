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
	seedDatabase(ctx, db) // Insert some dummy data for pagination

	fmt.Println("=== Pagination ===")

	pagination := postgres.NewPagination[User](db).Debug()

	// 1. Offset Pagination
	fmt.Println("--> Offset Pagination")
	offsetResp, err := pagination.Offset(ctx, &postgres.RequestPaginationOffset{
		Page:       1,
		Size:       5,
		Query:      "SELECT id, name, email FROM users WHERE status = :status ORDER BY id DESC",
		QueryCount: "SELECT count(*) FROM users WHERE status = :status",
		Kv:         []any{"status", "active"},
	})

	if err != nil {
		fmt.Printf("Offset Pagination error: %v\n", err)
	} else {
		fmt.Printf("Page %d of %d (Total Items: %d)\n", offsetResp.Page, offsetResp.TotalPages, offsetResp.TotalItems)
		for _, u := range offsetResp.Items {
			fmt.Printf("  - %v (%v)\n", u.Name, u.Email)
		}
	}

	// 2. Cursor Pagination
	fmt.Println("\n--> Cursor Pagination")
	cursorResp, err := pagination.Cursor(ctx, &postgres.RequestPaginationCursor{
		Query:      "SELECT id, name, email, created_at FROM users WHERE status = :status",
		QueryCount: "SELECT count(*) FROM users WHERE status = :status",
		Size:       5,
		Kv:         []any{"status", "active"},
		Sorts: []postgres.SortField{
			{Column: "created_at", Key: "created_at", IsDesc: true},
			{Column: "id", Key: "id", IsDesc: true}, // Unique fallback column
		},
		Token: "eyJjcmVhdGVkX2F0IjoiMjAyNi0wNC0yMlQwMzoxOToyOS40MTA1NThaIiwiaWQiOjIsImlzX3ByZXYiOmZhbHNlLCJvcGVyYXRvciI6IiBcdTAwM2MgIiwidG90YWxfaXRlbXMiOjZ9", // Emulate first page. To get next page, pass cursorResp.NextToken
	})

	if err != nil {
		fmt.Printf("Cursor Pagination error: %v\n", err)
	} else {
		fmt.Printf("Total Items: %d\n", cursorResp.TotalItems)
		for _, u := range cursorResp.Items {
			fmt.Printf("  - %v (%v)\n", u.Name, u.Email)
		}
		if cursorResp.NextToken != nil {
			fmt.Printf("Next Page Token: %s\n", *cursorResp.NextToken)
		}
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

func seedDatabase(ctx context.Context, db postgres.Postgres) {
	var count int64
	_, _ = db.Select("SELECT count(*) FROM users", &count).One(ctx)
	if count > 0 {
		return // Already seeded
	}

	fmt.Println("Seeding tables for pagination test...")
	for i := 1; i <= 15; i++ {
		_, _ = db.Insert("INSERT INTO users (name, email, status) VALUES (:name, :email, :status)",
			"name", fmt.Sprintf("User %d", i),
			"email", fmt.Sprintf("user%d@example.com", i),
			"status", "active").Exec(ctx)
	}
}
