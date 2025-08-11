package postgres

import (
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/newrelic/go-agent/v3/integrations/nrpq"
)

const size = 512

// New creates a new postgres client
func New(opts ...Option) (Postgres, error) {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	var (
		sqlxDB *sqlx.DB
		err    error
	)

	if cfg.dsn == "" {
		if err = cfg.BuildDsn(); err != nil {
			return nil, err
		}
	}

	sqlxDB, err = sqlx.Connect(cfg.driverName, cfg.dsn)
	if err != nil {
		return nil, err
	}

	if err = sqlxDB.Ping(); err != nil {
		return nil, err
	}

	pq := &postgres{
		database: sqlxDB,
	}

	if cfg.maxOpenConns > 0 {
		pq.database.SetMaxOpenConns(cfg.maxOpenConns)
	}
	if cfg.maxIdleConns > 0 {
		pq.database.SetMaxIdleConns(cfg.maxIdleConns)
	}
	if cfg.connMaxLifetime > 0 {
		pq.database.SetConnMaxLifetime(cfg.connMaxLifetime)
	}
	if cfg.connMaxIdleTime > 0 {
		pq.database.SetConnMaxIdleTime(cfg.connMaxIdleTime)
	}

	return pq, nil
}
