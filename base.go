package gormutil

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// BaseInterface defines common CRUD operations for database entities.
type BaseInterface[T any] interface {
	AutoMigrate(ctx context.Context) error
	Create(ctx context.Context, record *T) error
	GetByID(ctx context.Context, id string) (*T, error)
	PhysicalDelete(ctx context.Context, id string) (bool, error)
}

// Base provides basic CRUD operations for a database table.
type Base[T any] struct {
	db               *gorm.DB
	idColumnName     string
	extraMigrateSQLs []string
}

// NewBase creates a new Base instance.
// idColumnName is the name of the ID column (e.g., "id", "user_id").
// extraMigrateSQLs are optional SQL statements to execute after auto migration.
func NewBase[T any](db *gorm.DB, idColumnName string, extraMigrateSQLs ...string) *Base[T] {
	return &Base[T]{
		db:               db,
		idColumnName:     idColumnName,
		extraMigrateSQLs: extraMigrateSQLs,
	}
}

var _ BaseInterface[any] = (*Base[any])(nil)

// AutoMigrate runs auto migration for the entity type and executes extra SQL statements.
func (b *Base[T]) AutoMigrate(ctx context.Context) error {
	var t T
	if err := b.db.WithContext(ctx).AutoMigrate(&t); err != nil {
		return err
	}
	// Execute extra migration SQL statements
	for _, sql := range b.extraMigrateSQLs {
		if err := b.db.WithContext(ctx).Exec(sql).Error; err != nil {
			return fmt.Errorf("failed to execute extra migrate SQL: %w", err)
		}
	}
	return nil
}

// Create inserts a new record into the database.
func (b *Base[T]) Create(ctx context.Context, record *T) error {
	return b.db.WithContext(ctx).Create(record).Error
}

// GetByID retrieves a record by its ID.
func (b *Base[T]) GetByID(ctx context.Context, id string) (*T, error) {
	var result T
	err := b.db.WithContext(ctx).Where(fmt.Sprintf("%s = ?", b.idColumnName), id).First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// PhysicalDelete permanently deletes a record by its ID.
// Returns true if a record was deleted, false otherwise.
func (b *Base[T]) PhysicalDelete(ctx context.Context, id string) (bool, error) {
	var t T
	result := b.db.WithContext(ctx).Where(fmt.Sprintf("%s = ?", b.idColumnName), id).Delete(&t)
	return result.RowsAffected > 0, result.Error
}
