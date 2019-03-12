package lite

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"time"
)

// DB on top of underlying storage sqlite3
type DB struct {
	// underlying storage is Sqlite3. If needed, it might be replaced with redis
	storage *sql.DB
	path    string

	// If true, accepts read-only Tx
	readOnly bool
	opened   bool
}

// NewDatabase returns a pointer to a newly created or existing LevelDB blockchain db
// If open in Serialized mode, one connection be shared by multiple goroutines
func NewDatabase(path string, readonly bool) (*DB, error) {

	storage, err := sql.Open("sqlite3", "./"+path)
	if err != nil {
		return nil, err
	}
	// Create HEADERS table if not exist
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS HEADERS (
		hash            		TEXT    PRIMARY KEY  	NOT NULL,
		height           		INT     NOT NULL)
		`)

	stmt, err := storage.Prepare(query)

	if err != nil {
		return nil, err
	}
	stmt.Exec()

	return &DB{storage, path, false, true}, nil
}

func (db DB) isOpen() bool {
	return db.opened
}

// Begin builds (read-only or read-write) Tx, do initial validations
func (db DB) Begin(writable bool) (database.Tx, error) {
	// If the database was opened with Options.ReadOnly, return an error.
	if db.readOnly && writable {
		return nil, errors.New("database is read-only")
	}
	if !db.isOpen() {
		return nil, errors.New("database is not open")
	}

	// Create a transaction associated with the database.
	t := Tx{writable: writable, db: &db}

	return t, nil
}

// Update provides an execution of managed, read-write Tx
func (db DB) Update(fn func(database.Tx) error) error {
	start := time.Now()
	t, err := db.Begin(true)
	if err != nil {
		return err
	}

	err = fn(t)
	duration := time.Since(start)
	log.WithField("prefix", "database").Debugf("Transaction duration %d", duration.Nanoseconds())

	return err
}

// View provides an execution of managed, read-only Tx
func (db DB) View(fn func(database.Tx) error) error {
	start := time.Now()
	t, err := db.Begin(false)
	if err != nil {
		return err
	}

	// If an error is returned from the function then pass it through.
	err = fn(t)

	duration := time.Since(start)
	log.WithField("prefix", "database").Debugf("Transaction duration %d", duration.Nanoseconds())

	return err
}

func (db DB) Close() error {
	return nil
}
