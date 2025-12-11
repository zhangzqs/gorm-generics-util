package gormutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// BaseListInterface defines an interface for paginated list operations.
type BaseListInterface[T any, O any] interface {
	List(ctx context.Context, marker string, limit int, opts *O) (item []T, nextMarker string, err error)
}

// BaseList provides marker-based pagination for listing records.
type BaseList[T any, O any] struct {
	db                   *gorm.DB
	markerFieldExtractor func(T) string
	queryChainBuilder    func(chain ChainInterface[T], opts *O) ChainInterface[T]
	markerColumnName     string
}

// NewBaseList creates a new BaseList instance.
// markerColumnName is the column used for pagination (must be indexed and unique).
// markerFieldExtractor extracts the marker value from a record.
// queryChainBuilder is an optional function to build custom query conditions from options.
func NewBaseList[T any, O any](
	db *gorm.DB,
	markerColumnName string,
	markerFieldExtractor func(T) string,
	queryChainBuilder func(chain ChainInterface[T], opts *O) ChainInterface[T],
) *BaseList[T, O] {
	return &BaseList[T, O]{
		db:                   db,
		markerColumnName:     markerColumnName,
		markerFieldExtractor: markerFieldExtractor,
		queryChainBuilder:    queryChainBuilder,
	}
}

var _ BaseListInterface[any, any] = (*BaseList[any, any])(nil)

// List implements BaseListInterface.
func (b *BaseList[T, Options]) List(ctx context.Context, marker string, limit int, opts *Options) (item []T, nextMarker string, err error) {
	return FindWithMarkerPagination(
		ctx,
		b.queryChainBuilder(G[T](b.db), opts),
		b.markerColumnName,
		b.markerFieldExtractor,
		marker,
		limit,
	)
}

// FindWithMarkerPagination implements marker/limit-based pagination.
//
// Parameters:
//   - ctx: Context
//   - chain: GORM query chain
//   - markerColumnName: Column name used as pagination marker (must be indexed and unique/ascending)
//   - markerFieldExtractor: Function to extract marker value from a record
//   - marker: Marker value from the previous page, empty string for the first page
//   - limit: Maximum number of records per page (must be > 0)
//
// Returns:
//   - results: Current page records
//   - nextMarker: Marker for the next page, empty string if no more data
//   - err: Error information
func FindWithMarkerPagination[Record any](
	ctx context.Context,
	chain ChainInterface[Record],
	markerColumnName string,
	markerFieldExtractor func(Record) string,
	marker string,
	limit int,
) (results []Record, nextMarker string, err error) {
	// Parameter validation
	if limit <= 0 {
		return nil, "", fmt.Errorf("limit must be positive, got %d", limit)
	}
	if markerColumnName == "" {
		return nil, "", errors.New("markerColumnName cannot be empty")
	}
	if markerFieldExtractor == nil {
		return nil, "", errors.New("markerFieldExtractor cannot be nil")
	}

	// Build query: if there's a marker, query from after the marker
	if marker != "" {
		chain = chain.Where(fmt.Sprintf("%s > ?", markerColumnName), marker)
	}

	// Order by marker column ascending, fetch one extra record to check if there's a next page
	chain = chain.Order(fmt.Sprintf("%s ASC", markerColumnName)).Limit(limit + 1)

	// Execute query
	results, err = chain.Find(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to query records: %w", err)
	}

	// If more records than limit, there's a next page
	if len(results) > limit {
		// nextMarker should be the value at index limit (the limit+1 record)
		nextMarker = markerFieldExtractor(results[limit])
		results = results[:limit]
	}

	return results, nextMarker, nil
}

// CompositePaginationColumn defines configuration for a single column in composite pagination.
type CompositePaginationColumn struct {
	// ColumnName is the column name
	ColumnName string
	// Desc indicates descending order, default false means ascending
	Desc bool
}

// CompositePaginationMarker represents marker values for composite pagination.
type CompositePaginationMarker struct {
	// Values correspond to each column's value (in column order), stored as strings
	Values []string `json:"values"`
}

// Encode serializes the marker to a JSON string.
func (m CompositePaginationMarker) Encode() (string, error) {
	if len(m.Values) == 0 {
		return "", nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("failed to encode marker: %w", err)
	}
	return string(data), nil
}

// DecodeCompositePaginationMarker deserializes a JSON string to a marker.
func DecodeCompositePaginationMarker(s string) (CompositePaginationMarker, error) {
	if s == "" {
		return CompositePaginationMarker{}, nil
	}
	var m CompositePaginationMarker
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return CompositePaginationMarker{}, fmt.Errorf("failed to decode marker: %w", err)
	}
	return m, nil
}

// FindWithCompositePagination implements pagination based on multiple columns (composite key).
//
// This function supports multiple columns as pagination markers, suitable for composite keys
// or scenarios requiring multi-column sorting. It implements correct lexicographic comparison
// logic to ensure correctness and consistency of pagination results.
//
// Parameters:
//   - ctx: Context
//   - chain: GORM query chain
//   - columns: Column configurations for pagination (order matters, must match index order)
//   - markerExtractor: Function to extract marker value from a record
//   - marker: Marker value from the previous page, empty for the first page
//   - limit: Maximum number of records per page (must be > 0)
//
// Returns:
//   - results: Current page records
//   - nextMarker: Marker for the next page, empty if no more data
//   - err: Error information
//
// Example:
//
//	// For table items(user_id, created_at, id), paginate in order:
//	// user_id ASC, created_at DESC, id ASC
//	columns := []CompositePaginationColumn{
//	    {ColumnName: "user_id", Desc: false},
//	    {ColumnName: "created_at", Desc: true},
//	    {ColumnName: "id", Desc: false},
//	}
//	markerExtractor := func(item Item) CompositePaginationMarker {
//	    return CompositePaginationMarker{
//	        Values: []string{item.UserID, item.CreatedAt.Format(time.RFC3339), item.ID},
//	    }
//	}
//	results, nextMarker, err := FindWithCompositePagination(
//	    ctx, chain, columns, markerExtractor, marker, 10,
//	)
func FindWithCompositePagination[Record any](
	ctx context.Context,
	chain ChainInterface[Record],
	columns []CompositePaginationColumn,
	markerExtractor func(Record) CompositePaginationMarker,
	marker CompositePaginationMarker,
	limit int,
) (results []Record, nextMarker CompositePaginationMarker, err error) {
	// Parameter validation
	if limit <= 0 {
		return nil, CompositePaginationMarker{}, fmt.Errorf("limit must be positive, got %d", limit)
	}
	if len(columns) == 0 {
		return nil, CompositePaginationMarker{}, errors.New("columns cannot be empty")
	}
	if markerExtractor == nil {
		return nil, CompositePaginationMarker{}, errors.New("markerExtractor cannot be nil")
	}
	if len(marker.Values) > 0 && len(marker.Values) != len(columns) {
		return nil, CompositePaginationMarker{}, fmt.Errorf(
			"marker values length (%d) must match columns length (%d)",
			len(marker.Values), len(columns),
		)
	}

	// Build WHERE condition: if there's a marker, query from after the marker
	if len(marker.Values) > 0 {
		whereClause, args := buildCompositeWhereClause(columns, marker.Values)
		chain = chain.Where(whereClause, args...)
	}

	// Build ORDER BY clause
	var orderClauses []string
	for _, col := range columns {
		direction := "ASC"
		if col.Desc {
			direction = "DESC"
		}
		orderClauses = append(orderClauses, fmt.Sprintf("%s %s", col.ColumnName, direction))
	}
	chain = chain.Order(strings.Join(orderClauses, ", ")).Limit(limit + 1)

	// Execute query
	results, err = chain.Find(ctx)
	if err != nil {
		return nil, CompositePaginationMarker{}, fmt.Errorf("failed to query records: %w", err)
	}

	// If more records than limit, there's a next page
	if len(results) > limit {
		nextMarker = markerExtractor(results[limit])
		results = results[:limit]
	}

	return results, nextMarker, nil
}

// buildCompositeWhereClause builds WHERE condition for composite key.
//
// Implements lexicographic comparison logic:
// For columns (col1, col2, col3) with sort directions (ASC, DESC, ASC),
// WHERE condition should be:
//
//	(col1 > val1) OR
//	(col1 = val1 AND col2 < val2) OR
//	(col1 = val1 AND col2 = val2 AND col3 > val3)
//
// Note: DESC columns use < comparison, ASC columns use > comparison
func buildCompositeWhereClause(columns []CompositePaginationColumn, values []string) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	for i := range columns {
		// Build prefix equality conditions: col1 = val1 AND col2 = val2 AND ... AND col(i-1) = val(i-1)
		var prefixConditions []string
		for j := 0; j < i; j++ {
			prefixConditions = append(prefixConditions, fmt.Sprintf("%s = ?", columns[j].ColumnName))
			args = append(args, values[j])
		}

		// Current column comparison condition: col(i) > val(i) or col(i) < val(i)
		compareOp := ">"
		if columns[i].Desc {
			compareOp = "<"
		}
		currentCondition := fmt.Sprintf("%s %s ?", columns[i].ColumnName, compareOp)
		args = append(args, values[i])

		// Combine conditions
		if len(prefixConditions) > 0 {
			condition := fmt.Sprintf("(%s AND %s)", strings.Join(prefixConditions, " AND "), currentCondition)
			conditions = append(conditions, condition)
		} else {
			conditions = append(conditions, fmt.Sprintf("(%s)", currentCondition))
		}
	}

	// Connect all conditions with OR
	return strings.Join(conditions, " OR "), args
}
