package gormutil

import (
	"context"

	"gorm.io/gorm"
)

// ChainInterface represents a GORM query chain with generic type support.
type ChainInterface[T any] interface {
	Where(query interface{}, args ...interface{}) ChainInterface[T]
	Order(value interface{}) ChainInterface[T]
	Limit(limit int) ChainInterface[T]
	Find(ctx context.Context) ([]T, error)
	First(ctx context.Context) (T, error)
	Delete(ctx context.Context) (int64, error)
	Create(ctx context.Context, value *T) error
	Update(ctx context.Context, column string, value interface{}) error
	Updates(ctx context.Context, values interface{}) error
	Count(ctx context.Context) (int64, error)
	Exec(ctx context.Context, sql string, values ...interface{}) error
}

// chain implements ChainInterface with GORM DB.
type chain[T any] struct {
	db *gorm.DB
}

// G creates a new ChainInterface from a GORM DB instance.
func G[T any](db *gorm.DB) ChainInterface[T] {
	return &chain[T]{db: db}
}

// Where adds a WHERE condition to the query.
func (c *chain[T]) Where(query interface{}, args ...interface{}) ChainInterface[T] {
	return &chain[T]{db: c.db.Where(query, args...)}
}

// Order adds an ORDER BY clause to the query.
func (c *chain[T]) Order(value interface{}) ChainInterface[T] {
	return &chain[T]{db: c.db.Order(value)}
}

// Limit sets the maximum number of records to retrieve.
func (c *chain[T]) Limit(limit int) ChainInterface[T] {
	return &chain[T]{db: c.db.Limit(limit)}
}

// Find retrieves all matching records.
func (c *chain[T]) Find(ctx context.Context) ([]T, error) {
	var results []T
	err := c.db.WithContext(ctx).Find(&results).Error
	return results, err
}

// First retrieves the first matching record.
func (c *chain[T]) First(ctx context.Context) (T, error) {
	var result T
	err := c.db.WithContext(ctx).First(&result).Error
	return result, err
}

// Delete deletes matching records and returns the number of rows affected.
func (c *chain[T]) Delete(ctx context.Context) (int64, error) {
	var t T
	result := c.db.WithContext(ctx).Delete(&t)
	return result.RowsAffected, result.Error
}

// Create creates a new record.
func (c *chain[T]) Create(ctx context.Context, value *T) error {
	return c.db.WithContext(ctx).Create(value).Error
}

// Update updates a single column.
func (c *chain[T]) Update(ctx context.Context, column string, value interface{}) error {
	var t T
	return c.db.WithContext(ctx).Model(&t).Update(column, value).Error
}

// Updates updates multiple columns.
func (c *chain[T]) Updates(ctx context.Context, values interface{}) error {
	var t T
	return c.db.WithContext(ctx).Model(&t).Updates(values).Error
}

// Count returns the number of matching records.
func (c *chain[T]) Count(ctx context.Context) (int64, error) {
	var count int64
	var t T
	err := c.db.WithContext(ctx).Model(&t).Count(&count).Error
	return count, err
}

// Exec executes a raw SQL query.
func (c *chain[T]) Exec(ctx context.Context, sql string, values ...interface{}) error {
	return c.db.WithContext(ctx).Exec(sql, values...).Error
}
