// Command migrate is the operator-facing CLI for hongik-backend's database
// schema. The server applies pending migrations on startup, so this is a
// fallback for emergency rollback / forced version pinning, not the
// happy-path tool.
//
// Usage:
//
//	migrate up                 -- apply all pending up migrations
//	migrate down <N>           -- roll back the last N migrations
//	migrate version            -- print current schema version
//	migrate force <version>    -- mark schema at <version> without running SQL
//
// DATABASE_URL must be set in the environment.
package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"hongik-backend/migrations"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is not set")
		os.Exit(1)
	}

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "load migrations: %v\n", err)
		os.Exit(1)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init migrator: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
			fmt.Fprintf(os.Stderr, "migrator close: src=%v db=%v\n", srcErr, dbErr)
		}
	}()

	switch cmd := os.Args[1]; cmd {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fail(err)
		}
		printVersion(m)

	case "down":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "down requires a step count")
			os.Exit(2)
		}
		n, err := strconv.Atoi(os.Args[2])
		if err != nil || n <= 0 {
			fmt.Fprintf(os.Stderr, "invalid step count %q\n", os.Args[2])
			os.Exit(2)
		}
		if err := m.Steps(-n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fail(err)
		}
		printVersion(m)

	case "version":
		printVersion(m)

	case "force":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "force requires a version number")
			os.Exit(2)
		}
		v, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid version %q\n", os.Args[2])
			os.Exit(2)
		}
		if err := m.Force(v); err != nil {
			fail(err)
		}
		printVersion(m)

	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: migrate {up | down <N> | version | force <version>}")
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "migrate failed: %v\n", err)
	os.Exit(1)
}

func printVersion(m *migrate.Migrate) {
	v, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		fmt.Println("no migrations applied yet")
		return
	}
	if err != nil {
		fail(err)
	}
	if dirty {
		fmt.Printf("schema version %d (DIRTY — manual intervention required)\n", v)
		return
	}
	fmt.Printf("schema version %d\n", v)
}
