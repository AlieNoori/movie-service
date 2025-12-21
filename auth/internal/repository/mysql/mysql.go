package mysql

import (
	"context"
	"database/sql"
	"io/fs"

	"movieexample.com/auth/internal/repository"
	"movieexample.com/auth/pkg/model"
	"movieexample.com/pkg/migrate"
)

// Repository defines a MySQL-based user repository.
type Repository struct {
	db *sql.DB
}

// New creates a new MySQL-based user repository.
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

// Put adds or updates a user.
func (r *Repository) Put(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, username, email, password_hash, first_name, last_name, created_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE
			email = VALUES(email),
			username = VALUES(username),
			password_hash = VALUES(password_hash),
			first_name = VALUES(first_name),
			last_name = VALUES(last_name)`

	_, err := r.db.ExecContext(
		ctx,
		query,
		user.ID, user.Email, user.Username, user.PasswordHash, user.FirstName, user.LastName,
	)

	return err
}

// GetByEmail retrieves a user by their email address.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `SELECT id,username,password_hash,first_name,last_name,created_at FROM users
	WHERE email = ?`

	user := &model.User{
		Email: email,
	}

	if err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return user, nil
}

// GetByID retrieves a user by their unique ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*model.User, error) {
	query := `SELECT username,email,password_hash,first_name,last_name,created_at FROM users
	WHERE id = ?`

	user := &model.User{
		ID: id,
	}

	if err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return user, nil
}
