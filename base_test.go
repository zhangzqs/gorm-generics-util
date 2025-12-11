package gormutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SimpleTestItem is a simple test table structure.
type SimpleTestItem struct {
	ID   string `gorm:"column:id;primaryKey"`
	Name string `gorm:"column:name"`
}

func (SimpleTestItem) TableName() string {
	return "simple_test_items"
}

func setupSimpleTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestBase_AutoMigrate(t *testing.T) {
	ctx := context.Background()
	db := setupSimpleTestDB(t)

	base := NewBase[SimpleTestItem](db, "id")
	err := base.AutoMigrate(ctx)
	require.NoError(t, err)

	// Verify table exists by trying to query it
	var count int64
	err = db.Model(&SimpleTestItem{}).Count(&count).Error
	require.NoError(t, err)
}

func TestBase_AutoMigrate_WithExtraSQL(t *testing.T) {
	ctx := context.Background()
	db := setupSimpleTestDB(t)

	// Create base with extra migration SQL
	base := NewBase[SimpleTestItem](db, "id", "CREATE INDEX IF NOT EXISTS idx_name ON simple_test_items(name)")
	err := base.AutoMigrate(ctx)
	require.NoError(t, err)
}

func TestBase_Create(t *testing.T) {
	ctx := context.Background()
	db := setupSimpleTestDB(t)

	base := NewBase[SimpleTestItem](db, "id")
	err := base.AutoMigrate(ctx)
	require.NoError(t, err)

	item := &SimpleTestItem{
		ID:   "id1",
		Name: "test item",
	}

	err = base.Create(ctx, item)
	require.NoError(t, err)

	// Verify item was created
	var result SimpleTestItem
	err = db.Where("id = ?", "id1").First(&result).Error
	require.NoError(t, err)
	require.Equal(t, "id1", result.ID)
	require.Equal(t, "test item", result.Name)
}

func TestBase_GetByID(t *testing.T) {
	ctx := context.Background()
	db := setupSimpleTestDB(t)

	base := NewBase[SimpleTestItem](db, "id")
	err := base.AutoMigrate(ctx)
	require.NoError(t, err)

	// Create test item
	item := &SimpleTestItem{
		ID:   "id1",
		Name: "test item",
	}
	err = db.Create(item).Error
	require.NoError(t, err)

	// Get by ID
	result, err := base.GetByID(ctx, "id1")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "id1", result.ID)
	require.Equal(t, "test item", result.Name)
}

func TestBase_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	db := setupSimpleTestDB(t)

	base := NewBase[SimpleTestItem](db, "id")
	err := base.AutoMigrate(ctx)
	require.NoError(t, err)

	// Try to get non-existent item
	result, err := base.GetByID(ctx, "nonexistent")
	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestBase_PhysicalDelete(t *testing.T) {
	ctx := context.Background()
	db := setupSimpleTestDB(t)

	base := NewBase[SimpleTestItem](db, "id")
	err := base.AutoMigrate(ctx)
	require.NoError(t, err)

	// Create test item
	item := &SimpleTestItem{
		ID:   "id1",
		Name: "test item",
	}
	err = db.Create(item).Error
	require.NoError(t, err)

	// Delete item
	deleted, err := base.PhysicalDelete(ctx, "id1")
	require.NoError(t, err)
	require.True(t, deleted)

	// Verify item was deleted
	var count int64
	err = db.Model(&SimpleTestItem{}).Where("id = ?", "id1").Count(&count).Error
	require.NoError(t, err)
	require.Equal(t, int64(0), count)
}

func TestBase_PhysicalDelete_NotFound(t *testing.T) {
	ctx := context.Background()
	db := setupSimpleTestDB(t)

	base := NewBase[SimpleTestItem](db, "id")
	err := base.AutoMigrate(ctx)
	require.NoError(t, err)

	// Try to delete non-existent item
	deleted, err := base.PhysicalDelete(ctx, "nonexistent")
	require.NoError(t, err)
	require.False(t, deleted)
}
