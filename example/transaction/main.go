package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/andryhardiyanto/go-postgres"
	"github.com/joho/godotenv"
)

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

	fmt.Println("=== Transaction Pipeline ===")

	// The pipeline executes multiple queries in a single transaction.
	// We can use db.FromResult() to grab the returning value of a previous query in the pipeline.
	result, err := db.
		Insert("INSERT INTO users (name, email) VALUES (:name, :email) RETURNING id",
			"name", "Alice",
			"email", "alice@example.com").
		Insert("INSERT INTO profiles (user_id, bio) VALUES (:user_id, :bio)",
			"user_id", db.FromResult("INSERT INTO users (name, email) VALUES (:name, :email) RETURNING id"),
			"bio", "Backend Engineer").
		ExecInTx(ctx)

	if err != nil {
		fmt.Printf("Transaction failed: %v\n", err)
		return
	}

	userId := result.TxResult("INSERT INTO users (name, email) VALUES (:name, :email) RETURNING id")
	fmt.Printf("Successfully created User (ID: %v) and Profile in transaction.\n", userId)
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

	profilesTable := `
	CREATE TABLE IF NOT EXISTS profiles (
		id SERIAL PRIMARY KEY,
		user_id INTEGER REFERENCES users(id),
		bio TEXT
	);`

	_, err := db.Update(usersTable).Exec(ctx)
	if err != nil {
		log.Fatalf("Failed to create users table: %v", err)
	}

	_, err = db.Update(profilesTable).Exec(ctx)
	if err != nil {
		log.Fatalf("Failed to create profiles table: %v", err)
	}
}
