package mysql

import (
	"context"
	"database/sql"
	"io/fs"

	_ "github.com/go-sql-driver/mysql"
	"go.opentelemetry.io/otel"
	"movieexample.com/metadata/internal/repository"
	"movieexample.com/metadata/pkg/model"
	"movieexample.com/pkg/migrate"
)

const tracerID = "metadata-repository-mysql"

// Repository defines a MySQL-based movie metadata repository.
type Repository struct {
	db *sql.DB
}

// New creates a new MySQL-based repository.
func New(DSN string) (*Repository, error) {
	db, err := sql.Open("mysql", DSN)
	if err != nil {
		return nil, err
	}

	return &Repository{db}, nil
}

// New creates a new MySQL-based repository and running migrations.
func NewWithMigration(DSN string, fs fs.FS, dir string) (*Repository, error) {
	db, err := sql.Open("mysql", DSN)
	if err != nil {
		return nil, err
	}

	err = migrate.MigrateFS(db, fs, ".")
	if err != nil {
		return nil, err
	}

	return &Repository{db}, nil
}

// Get retrieves movie metadata for by movie id.
func (r *Repository) Get(ctx context.Context, id string) (*model.Metadata, error) {
	_, span := otel.Tracer(tracerID).Start(ctx, "Repository/Get")
	defer span.End()

	query := `SELECT title, description, director FROM movies WHERE id = ?`

	var title, description, director string
	row := r.db.QueryRowContext(ctx, query, id)
	if err := row.Scan(&title, &description, &director); err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	metaData := &model.Metadata{
		ID:          id,
		Title:       title,
		Description: description,
		Director:    director,
	}

	return metaData, nil
}

// Put adds movie metadata for a given movie id.
func (r *Repository) Put(ctx context.Context, id string, metadata *model.Metadata) error {
	_, span := otel.Tracer(tracerID).Start(ctx, "Repository/Put")
	defer span.End()

	qeury := `INSERT INTO movies (id,title,description,director) VALUES (?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, qeury, id, metadata.Title, metadata.Description, metadata.Director)

	return err
}

// Close gracefully shuts down the MySQL connection.
func (r *Repository) Close() error {
	return r.db.Close()
}
