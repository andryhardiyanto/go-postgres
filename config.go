package postgres

import (
	"fmt"
	"time"
)

// Option is a function that configures the postgres database.
type (
	Option func(*config)

	// Config is the configuration for the postgres database.
	config struct {
		driverName      string
		host            string
		port            int
		user            string
		password        string
		dbName          string
		sslMode         string
		dsn             string
		maxIdleConns    int
		maxOpenConns    int
		connMaxLifetime time.Duration
		connMaxIdleTime time.Duration
	}
)

// BuildDsn builds the dsn.
func (c *config) BuildDsn() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if c.host == "" {
		return fmt.Errorf("host is required")
	}
	if c.port == 0 {
		return fmt.Errorf("port is required")
	}
	if c.user == "" {
		return fmt.Errorf("username is required")
	}
	if c.password == "" {
		return fmt.Errorf("password is required")
	}
	if c.dbName == "" {
		return fmt.Errorf("database name is required")
	}
	if c.sslMode == "" {
		return fmt.Errorf("ssl mode is required")
	}
	if c.driverName == "" {
		return fmt.Errorf("driver name is required")
	}

	c.dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.host, c.port, c.user, c.password, c.dbName, c.sslMode)

	return nil
}

// WithDriverName sets the driver name.
func WithDriverName(driverName string) Option {
	return func(c *config) {
		c.driverName = driverName
	}
}

// WithDsn sets the dsn.
// dsn is the data source name.
func WithDsn(dsn string) Option {
	return func(c *config) {
		c.dsn = dsn
	}
}

// WithMaxOpenConns sets the max open conns.
// maxOpenConns is the maximum number of open connections to the database.
func WithMaxOpenConns(maxOpenConns int) Option {
	return func(c *config) {
		c.maxOpenConns = maxOpenConns
	}
}

func WithMaxIdleConns(maxIdleConns int) Option {
	// maxIdleConns is the maximum number of idle connections in the pool.
	return func(c *config) {
		c.maxIdleConns = maxIdleConns
	}
}

// WithConnMaxIdleTime sets the max conn idle time.
// maxConnIdleTime is the maximum amount of time a connection may be idle.
func WithConnMaxIdleTime(maxConnIdleTime time.Duration) Option {
	return func(c *config) {
		c.connMaxIdleTime = maxConnIdleTime
	}
}

// WithConnMaxLifetime sets the max conn lifetime.
// maxConnLifetime is the maximum amount of time a connection may be reused.
func WithConnMaxLifetime(maxConnLifetime time.Duration) Option {
	return func(c *config) {
		c.connMaxLifetime = maxConnLifetime
	}
}

// WithHost sets the host.
func WithHost(host string) Option {
	return func(c *config) {
		c.host = host
	}
}

// WithPort sets the port.
// port is the port number to connect to.
func WithPort(port int) Option {
	return func(c *config) {
		c.port = port
	}
}

// WithUser sets the user.
// user is the username to connect as.
func WithUser(user string) Option {
	return func(c *config) {
		c.user = user
	}
}

// WithPassword sets the password.
// password is the password to connect with.
func WithPassword(password string) Option {
	return func(c *config) {
		c.password = password
	}
}

// WithDBName sets the db name.
// dbName is the name of the database to connect to.
func WithDBName(dbName string) Option {
	return func(c *config) {
		c.dbName = dbName
	}
}

// WithSSLMode sets the ssl mode.
// sslMode is the SSL mode to use.
func WithSSLMode(sslMode string) Option {
	return func(c *config) {
		c.sslMode = sslMode
	}
}

// WithConnMax sets the max conn.
// maxOpenConns is the maximum number of open connections to the database.
func WithConnMax(maxOpenConns int) Option {
	return func(c *config) {
		c.maxOpenConns = maxOpenConns
	}
}
