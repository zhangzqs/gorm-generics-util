package gormutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestItem is a test table structure.
type TestItem struct {
	UserID    string `gorm:"column:user_id;primaryKey"`
	CreatedAt string `gorm:"column:created_at;primaryKey"`
	ID        string `gorm:"column:id;primaryKey"`
	Name      string `gorm:"column:name"`
}

func (TestItem) TableName() string {
	return "test_items"
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&TestItem{})
	require.NoError(t, err)

	return db
}

func TestCompositePaginationMarker_Encode(t *testing.T) {
	tests := []struct {
		name    string
		marker  CompositePaginationMarker
		want    string
		wantErr bool
	}{
		{
			name:    "empty marker",
			marker:  CompositePaginationMarker{},
			want:    "",
			wantErr: false,
		},
		{
			name: "single value",
			marker: CompositePaginationMarker{
				Values: []string{"user123"},
			},
			want:    `{"values":["user123"]}`,
			wantErr: false,
		},
		{
			name: "multiple values",
			marker: CompositePaginationMarker{
				Values: []string{"user123", "2024-01-01T00:00:00Z", "id456"},
			},
			want:    `{"values":["user123","2024-01-01T00:00:00Z","id456"]}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.marker.Encode()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestDecodeCompositePaginationMarker(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    CompositePaginationMarker
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			want:    CompositePaginationMarker{},
			wantErr: false,
		},
		{
			name:  "single value",
			input: `{"values":["user123"]}`,
			want: CompositePaginationMarker{
				Values: []string{"user123"},
			},
			wantErr: false,
		},
		{
			name:  "multiple values",
			input: `{"values":["user123","2024-01-01T00:00:00Z","id456"]}`,
			want: CompositePaginationMarker{
				Values: []string{"user123", "2024-01-01T00:00:00Z", "id456"},
			},
			wantErr: false,
		},
		{
			name:    "invalid json",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeCompositePaginationMarker(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestBuildCompositeWhereClause(t *testing.T) {
	tests := []struct {
		name          string
		columns       []CompositePaginationColumn
		values        []string
		wantClause    string
		wantArgsCount int
	}{
		{
			name: "single column ASC",
			columns: []CompositePaginationColumn{
				{ColumnName: "id", Desc: false},
			},
			values:        []string{"100"},
			wantClause:    "(id > ?)",
			wantArgsCount: 1,
		},
		{
			name: "single column DESC",
			columns: []CompositePaginationColumn{
				{ColumnName: "created_at", Desc: true},
			},
			values:        []string{"2024-01-01"},
			wantClause:    "(created_at < ?)",
			wantArgsCount: 1,
		},
		{
			name: "two columns ASC, ASC",
			columns: []CompositePaginationColumn{
				{ColumnName: "user_id", Desc: false},
				{ColumnName: "id", Desc: false},
			},
			values:        []string{"user123", "id456"},
			wantClause:    "(user_id > ?) OR (user_id = ? AND id > ?)",
			wantArgsCount: 3,
		},
		{
			name: "two columns ASC, DESC",
			columns: []CompositePaginationColumn{
				{ColumnName: "user_id", Desc: false},
				{ColumnName: "created_at", Desc: true},
			},
			values:        []string{"user123", "2024-01-01"},
			wantClause:    "(user_id > ?) OR (user_id = ? AND created_at < ?)",
			wantArgsCount: 3,
		},
		{
			name: "three columns ASC, DESC, ASC",
			columns: []CompositePaginationColumn{
				{ColumnName: "user_id", Desc: false},
				{ColumnName: "created_at", Desc: true},
				{ColumnName: "id", Desc: false},
			},
			values:        []string{"user123", "2024-01-01", "id456"},
			wantClause:    "(user_id > ?) OR (user_id = ? AND created_at < ?) OR (user_id = ? AND created_at = ? AND id > ?)",
			wantArgsCount: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotClause, gotArgs := buildCompositeWhereClause(tt.columns, tt.values)
			require.Equal(t, tt.wantClause, gotClause)
			require.Len(t, gotArgs, tt.wantArgsCount)
		})
	}
}

func TestFindWithCompositePagination(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)

	// Prepare test data
	testData := []TestItem{
		{UserID: "user1", CreatedAt: "2024-01-03", ID: "id1", Name: "item1"},
		{UserID: "user1", CreatedAt: "2024-01-02", ID: "id2", Name: "item2"},
		{UserID: "user1", CreatedAt: "2024-01-01", ID: "id3", Name: "item3"},
		{UserID: "user2", CreatedAt: "2024-01-03", ID: "id4", Name: "item4"},
		{UserID: "user2", CreatedAt: "2024-01-02", ID: "id5", Name: "item5"},
		{UserID: "user2", CreatedAt: "2024-01-01", ID: "id6", Name: "item6"},
	}

	for _, item := range testData {
		require.NoError(t, db.Create(&item).Error)
	}

	// Define pagination column config: user_id ASC, created_at DESC, id ASC
	columns := []CompositePaginationColumn{
		{ColumnName: "user_id", Desc: false},
		{ColumnName: "created_at", Desc: true},
		{ColumnName: "id", Desc: false},
	}

	markerExtractor := func(item TestItem) CompositePaginationMarker {
		return CompositePaginationMarker{
			Values: []string{item.UserID, item.CreatedAt, item.ID},
		}
	}

	t.Run("first page", func(t *testing.T) {
		results, nextMarker, err := FindWithCompositePagination(
			ctx,
			G[TestItem](db),
			columns,
			markerExtractor,
			CompositePaginationMarker{},
			2,
		)

		require.NoError(t, err)
		require.Len(t, results, 2)

		// Verify sorting: user1 data should come first, sorted by created_at DESC
		require.Equal(t, "user1", results[0].UserID)
		require.Equal(t, "2024-01-03", results[0].CreatedAt)
		require.Equal(t, "id1", results[0].ID)

		require.Equal(t, "user1", results[1].UserID)
		require.Equal(t, "2024-01-02", results[1].CreatedAt)
		require.Equal(t, "id2", results[1].ID)

		// Verify nextMarker: should be the 3rd record (index limit i.e. 2)
		// Actually queried 3 records, return first 2, nextMarker is the 3rd
		require.NotEmpty(t, nextMarker.Values)
		require.Equal(t, []string{"user1", "2024-01-01", "id3"}, nextMarker.Values)
	})

	t.Run("second page", func(t *testing.T) {
		marker := CompositePaginationMarker{
			Values: []string{"user1", "2024-01-01", "id3"},
		}

		results, nextMarker, err := FindWithCompositePagination(
			ctx,
			G[TestItem](db),
			columns,
			markerExtractor,
			marker,
			2,
		)

		require.NoError(t, err)
		require.Len(t, results, 2)

		// Verify second page data: from after marker, should be user2 data
		require.Equal(t, "user2", results[0].UserID)
		require.Equal(t, "2024-01-03", results[0].CreatedAt)
		require.Equal(t, "id4", results[0].ID)

		require.Equal(t, "user2", results[1].UserID)
		require.Equal(t, "2024-01-02", results[1].CreatedAt)
		require.Equal(t, "id5", results[1].ID)

		// Verify nextMarker
		require.NotEmpty(t, nextMarker.Values)
		require.Equal(t, []string{"user2", "2024-01-01", "id6"}, nextMarker.Values)
	})

	t.Run("last page", func(t *testing.T) {
		marker := CompositePaginationMarker{
			Values: []string{"user2", "2024-01-01", "id6"},
		}

		results, nextMarker, err := FindWithCompositePagination(
			ctx,
			G[TestItem](db),
			columns,
			markerExtractor,
			marker,
			2,
		)

		require.NoError(t, err)
		require.Empty(t, results) // No more data

		// Verify no next page
		require.Empty(t, nextMarker.Values)
	})

	t.Run("empty result", func(t *testing.T) {
		marker := CompositePaginationMarker{
			Values: []string{"user3", "2024-01-01", "id999"},
		}

		results, nextMarker, err := FindWithCompositePagination(
			ctx,
			G[TestItem](db),
			columns,
			markerExtractor,
			marker,
			2,
		)

		require.NoError(t, err)
		require.Empty(t, results)
		require.Empty(t, nextMarker.Values)
	})
}

func TestFindWithCompositePagination_ParameterValidation(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)

	columns := []CompositePaginationColumn{
		{ColumnName: "user_id", Desc: false},
	}
	markerExtractor := func(item TestItem) CompositePaginationMarker {
		return CompositePaginationMarker{Values: []string{item.UserID}}
	}

	t.Run("invalid limit", func(t *testing.T) {
		_, _, err := FindWithCompositePagination(
			ctx,
			G[TestItem](db),
			columns,
			markerExtractor,
			CompositePaginationMarker{},
			0,
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "limit must be positive")
	})

	t.Run("empty columns", func(t *testing.T) {
		_, _, err := FindWithCompositePagination(
			ctx,
			G[TestItem](db),
			[]CompositePaginationColumn{},
			markerExtractor,
			CompositePaginationMarker{},
			10,
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "columns cannot be empty")
	})

	t.Run("nil markerExtractor", func(t *testing.T) {
		_, _, err := FindWithCompositePagination(
			ctx,
			G[TestItem](db),
			columns,
			nil,
			CompositePaginationMarker{},
			10,
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "markerExtractor cannot be nil")
	})

	t.Run("marker values length mismatch", func(t *testing.T) {
		marker := CompositePaginationMarker{
			Values: []string{"user1", "extra_value"},
		}
		_, _, err := FindWithCompositePagination(
			ctx,
			G[TestItem](db),
			columns,
			markerExtractor,
			marker,
			10,
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "marker values length")
	})
}

func TestFindWithMarkerPagination(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)

	// Prepare test data
	testData := []TestItem{
		{UserID: "user1", CreatedAt: "2024-01-01", ID: "id1", Name: "item1"},
		{UserID: "user1", CreatedAt: "2024-01-01", ID: "id2", Name: "item2"},
		{UserID: "user1", CreatedAt: "2024-01-01", ID: "id3", Name: "item3"},
	}

	for _, item := range testData {
		require.NoError(t, db.Create(&item).Error)
	}

	markerExtractor := func(item TestItem) string {
		return item.ID
	}

	t.Run("first page", func(t *testing.T) {
		results, nextMarker, err := FindWithMarkerPagination(
			ctx,
			G[TestItem](db),
			"id",
			markerExtractor,
			"",
			2,
		)

		require.NoError(t, err)
		require.Len(t, results, 2)
		require.Equal(t, "id1", results[0].ID)
		require.Equal(t, "id2", results[1].ID)
		// nextMarker should be the 3rd record (index limit i.e. 2)
		require.Equal(t, "id3", nextMarker)
	})

	t.Run("second page", func(t *testing.T) {
		results, nextMarker, err := FindWithMarkerPagination(
			ctx,
			G[TestItem](db),
			"id",
			markerExtractor,
			"id3",
			2,
		)

		require.NoError(t, err)
		require.Empty(t, results) // No more data
		require.Empty(t, nextMarker)
	})
}

func TestFindWithMarkerPagination_ParameterValidation(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)

	markerExtractor := func(item TestItem) string {
		return item.ID
	}

	t.Run("invalid limit", func(t *testing.T) {
		_, _, err := FindWithMarkerPagination(
			ctx,
			G[TestItem](db),
			"id",
			markerExtractor,
			"",
			0,
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "limit must be positive")
	})

	t.Run("empty markerColumnName", func(t *testing.T) {
		_, _, err := FindWithMarkerPagination(
			ctx,
			G[TestItem](db),
			"",
			markerExtractor,
			"",
			10,
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "markerColumnName cannot be empty")
	})

	t.Run("nil markerExtractor", func(t *testing.T) {
		_, _, err := FindWithMarkerPagination(
			ctx,
			G[TestItem](db),
			"id",
			nil,
			"",
			10,
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "markerFieldExtractor cannot be nil")
	})
}

// BenchmarkFindWithCompositePagination benchmark test.
func BenchmarkFindWithCompositePagination(b *testing.B) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		b.Fatal(err)
	}

	err = db.AutoMigrate(&TestItem{})
	if err != nil {
		b.Fatal(err)
	}

	// Prepare large amount of test data
	for i := 0; i < 1000; i++ {
		item := TestItem{
			UserID:    fmt.Sprintf("user%d", i%10),
			CreatedAt: fmt.Sprintf("2024-01-%02d", i%30+1),
			ID:        fmt.Sprintf("id%d", i),
			Name:      fmt.Sprintf("item%d", i),
		}
		_ = db.Create(&item).Error
	}

	columns := []CompositePaginationColumn{
		{ColumnName: "user_id", Desc: false},
		{ColumnName: "created_at", Desc: true},
		{ColumnName: "id", Desc: false},
	}

	markerExtractor := func(item TestItem) CompositePaginationMarker {
		return CompositePaginationMarker{
			Values: []string{item.UserID, item.CreatedAt, item.ID},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = FindWithCompositePagination(
			ctx,
			G[TestItem](db),
			columns,
			markerExtractor,
			CompositePaginationMarker{},
			10,
		)
	}
}
