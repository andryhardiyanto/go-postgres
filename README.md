# Go PostgreSQL Library

A PostgreSQL library built on top of sqlx with focus on performance, security, and ease of use.

## ✨ Features

- **Connection Pooling**: Optimal connection pool configuration
- **Transaction Support**: Pipeline transactions with automatic rollback
- **Memory Safe**: Prepared statements automatically closed to prevent memory leaks
- **Type Safety**: Custom types for PostgreSQL arrays
- **Parameter Binding**: Named parameters to prevent SQL injection

## 🚀 Quick Start

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/andryhardiyanto/go-postgres"
)

func main() {
    // Initialize database
    db, err := postgres.New(
        postgres.WithHost("localhost"),
        postgres.WithPort(5432),
        postgres.WithUser("username"),
        postgres.WithPassword("password"),
        postgres.WithDBName("mydb"),
        postgres.WithSSLMode("disable"),
        postgres.WithDriverName("postgres"),
        postgres.WithMaxOpenConns(25),
        postgres.WithMaxIdleConns(5),
        postgres.WithConnMaxLifetime(5*time.Minute),
        postgres.WithConnMaxIdleTime(1*time.Minute),
    )
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Select single record
    var user User
    found, err := db.Select("SELECT * FROM users WHERE id = :id", &user, "id", 1).One(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if found {
        log.Printf("User: %+v", user)
    }

    // Select multiple records with debug
    var users []User
    found, err = db.Select("SELECT * FROM users WHERE active = :active", &users, "active", true).Debug().Many(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Insert with returning ID
    id, err := db.Insert("INSERT INTO users (name, email) VALUES (:name, :email) RETURNING id", 
        "name", "John Doe", 
        "email", "john@example.com").Exec(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Transaction
    result, err := db.Insert("INSERT INTO users (name) VALUES (:name) RETURNING id", "name", "User1").
        Insert("INSERT INTO profiles (user_id, bio) VALUES (:user_id, :bio)", 
            "user_id", db.FromResult("INSERT INTO users (name) VALUES (:name) RETURNING id"), 
            "bio", "User bio").
        ExecInTx(ctx)
    if err != nil {
        log.Fatal(err)
    }
}
```

## 📊 Performance Optimizations

### 1. Connection Pool Configuration
```go
db, err := postgres.New(
    // Optimal for web applications with moderate traffic
    postgres.WithMaxOpenConns(25),        // Max connections
    postgres.WithMaxIdleConns(5),         // Idle connections
    postgres.WithConnMaxLifetime(5*time.Minute),  // Connection lifetime
    postgres.WithConnMaxIdleTime(1*time.Minute),  // Idle timeout
)
```

### 2. Context with Timeout
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

found, err := db.Select("SELECT * FROM users WHERE id = :id", &user, "id", 1).One(ctx)
```

### 3. Prepared Statements
The library automatically uses prepared statements and closes them to prevent memory leaks.

### 4. Debug Mode
Enable debug mode for individual queries to see SQL execution:
```go
// Debug a specific select query
found, err := db.Select("SELECT * FROM users WHERE id = :id", &user, "id", 1).Debug().One(ctx)

// Debug an insert query
id, err := db.Insert("INSERT INTO users (name) VALUES (:name) RETURNING id", "name", "John").Debug().Exec(ctx)
```

## 🔒 Security Best Practices

### 1. Parameter Binding
Always use named parameters to prevent SQL injection:
```go
// ✅ CORRECT
db.Select("SELECT * FROM users WHERE email = :email", &user, "email", userEmail)

// ❌ WRONG - vulnerable to SQL injection
db.Select(fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", userEmail), &user)
```

### 2. Connection Security
```go
postgres.New(
    postgres.WithSSLMode("require"),  // Use SSL in production
    postgres.WithHost("localhost"),   // Don't expose to public
)
```

## 🔧 Error Handling

```go
_, err := db.Insert("INSERT INTO users (email) VALUES (:email)", "email", "duplicate@example.com").Exec(ctx)
if err != nil {
    // Handle database errors
    log.Printf("Database error: %v", err)
    return err
}
```

## 🤝 Contributing

We welcome contributions! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

## 🌟 Support

If you encounter any issues or have questions, please open an issue on GitHub.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.