package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	applicationauth "github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/application/auth"
	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/domain/user"
	drivermysql "github.com/go-sql-driver/mysql"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, newUser user.User) (user.User, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO users (uuid, name, last_name, email, password_hash) VALUES (?, ?, ?, ?, ?)`,
		newUser.UUID, newUser.Name, newUser.LastName, newUser.Email, newUser.PasswordHash,
	)
	if err != nil {
		if isDuplicateEmail(err) {
			return user.User{}, applicationauth.ErrEmailAlreadyExists
		}
		return user.User{}, fmt.Errorf("insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return user.User{}, fmt.Errorf("get inserted user ID: %w", err)
	}
	return r.findByID(ctx, uint64(id))
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (user.User, error) {
	return scanUser(r.db.QueryRowContext(ctx,
		`SELECT id, uuid, name, last_name, email, password_hash, created_at, updated_at FROM users WHERE email = ?`, email,
	))
}

func (r *UserRepository) FindByUUID(ctx context.Context, publicUUID string) (user.User, error) {
	return scanUser(r.db.QueryRowContext(ctx,
		`SELECT id, uuid, name, last_name, email, password_hash, created_at, updated_at FROM users WHERE uuid = ?`, publicUUID,
	))
}

func (r *UserRepository) Update(ctx context.Context, updatedUser user.User) (user.User, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET name = ?, last_name = ?, email = ?, updated_at = CURRENT_TIMESTAMP(6) WHERE id = ?`,
		updatedUser.Name, updatedUser.LastName, updatedUser.Email, updatedUser.ID,
	)
	if err != nil {
		if isDuplicateEmail(err) {
			return user.User{}, applicationauth.ErrEmailAlreadyExists
		}
		return user.User{}, fmt.Errorf("update user: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return user.User{}, fmt.Errorf("check updated user: %w", err)
	}
	if affected == 0 {
		return r.findByID(ctx, updatedUser.ID)
	}
	return r.findByID(ctx, updatedUser.ID)
}

func (r *UserRepository) Delete(ctx context.Context, publicUUID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE uuid = ?`, publicUUID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check deleted user: %w", err)
	}
	if affected == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *UserRepository) findByID(ctx context.Context, id uint64) (user.User, error) {
	return scanUser(r.db.QueryRowContext(ctx,
		`SELECT id, uuid, name, last_name, email, password_hash, created_at, updated_at FROM users WHERE id = ?`, id,
	))
}

type rowScanner interface {
	Scan(...any) error
}

func scanUser(row rowScanner) (user.User, error) {
	var found user.User
	err := row.Scan(&found.ID, &found.UUID, &found.Name, &found.LastName, &found.Email, &found.PasswordHash, &found.CreatedAt, &found.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return user.User{}, user.ErrNotFound
	}
	if err != nil {
		return user.User{}, fmt.Errorf("scan user: %w", err)
	}
	return found, nil
}

func isDuplicateEmail(err error) bool {
	var mysqlErr *drivermysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
