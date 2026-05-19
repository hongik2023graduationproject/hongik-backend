package service

import (
	"database/sql"
	"errors"
	"fmt"

	"hongik-backend/migrations"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// RunMigrations applies every pending up-migration embedded in the
// hongik-backend/migrations package against the supplied PostgreSQL
// connection. It is idempotent: re-running on an up-to-date database is
// a no-op.
//
// We intentionally accept *sql.DB rather than a DSN so the caller (main)
// keeps a single owner of the connection pool and can close it during
// graceful shutdown.
func RunMigrations(db *sql.DB) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
	}

	// WithInstance reuses the existing pool instead of opening a second one.
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("postgres migrate driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	// 의도적으로 m.Close()를 호출하지 않는다.
	// golang-migrate v4의 postgres 드라이버 Close()는 WithInstance로 받은 *sql.DB까지 닫는데,
	// 그 풀은 호출자(main)가 소유하며 서버 수명 동안 살아 있어야 한다. 닫으면 이후 모든 쿼리가 실패한다.
	// 누수 범위는 advisory lock용 *sql.Conn 1개로, 프로세스 1회당 고정이라 누적되지 않는다.
	return nil
}
