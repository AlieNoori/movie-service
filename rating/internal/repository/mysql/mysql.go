package mysql

import (
	"context"
	"database/sql"
	"errors"

	_ "github.com/go-sql-driver/mysql"
	"movieexample.com/rating/pkg/model"
)

// Repository defines a MySQL-based rating repository.
type Repository struct {
	db *sql.DB
}

// New creates a new MySQL-based rating repository.
func New() (*Repository, error) {
	db, err := sql.Open("mysql", "root:password@/movieexample")
	if err != nil {
		return nil, err
	}
	return &Repository{db}, nil
}

// Get retrieves all ratings for a given record.
func (r *Repository) Get(ctx context.Context, recordID model.RecordID, recordType model.RecordType) ([]model.Rating, error) {
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
	if rating == nil {
		return errors.New("rating is nil")
	}

	query := `INSERT INTO ratings (record_id,record_type,user_id,value) VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, recordID, recordType, rating.UserID, rating.Value)
	return err
}
