package mysql

import (
	"context"
	"database/sql"
	"io/fs"

	_ "github.com/go-sql-driver/mysql"
	"go.opentelemetry.io/otel"
	"movieexample.com/pkg/migrate"
	"movieexample.com/rating/pkg/model"
)

const tracerID = "rating-repository-mysql"

// Repository defines a MySQL-based rating repository.
type Repository struct {
	db *sql.DB
}

// New creates a new MySQL-based rating repository.
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

// Get retrieves all ratings for a given record.
func (r *Repository) Get(ctx context.Context, recordID model.RecordID, recordType model.RecordType) ([]model.Rating, error) {
	_, span := otel.Tracer(tracerID).Start(ctx, "Repository/Get")
	defer span.End()
	query := `SELECT user_id,value FROM ratings WHERE record_id=? AND record_type=?`
	rows, err := r.db.QueryContext(ctx, query, recordID, recordType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ratings []model.Rating
	for rows.Next() {
		var value int32
		var userID string
		if err := rows.Scan(&userID, &value); err != nil {
			return nil, err
		}

		ratings = append(ratings, model.Rating{
			RecordID:   recordID,
			RecordType: recordType,
			UserID:     model.UserID(userID),
			Value:      model.RatingValue(value),
		})
	}

	return ratings, nil
}

// Put adds a rating for a given record.
func (r *Repository) Put(ctx context.Context, recordID model.RecordID, recordType model.RecordType, rating *model.Rating) error {
	_, span := otel.Tracer(tracerID).Start(ctx, "Repository/Put")
	defer span.End()

	query := `INSERT INTO ratings (record_id,record_type,user_id,value) 
	VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, recordID, recordType, rating.UserID, rating.Value)
	return err
}

// Close gracefully shuts down the MySQL connection.
func (r *Repository) Close() error {
	return r.db.Close()
}
