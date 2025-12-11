package gormutil_test

import (
	"context"
	"fmt"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	gormutil "github.com/zhangzqs/gorm-generics-util"
)

// User represents a user entity
type User struct {
	ID   string `gorm:"column:id;primaryKey"`
	Name string `gorm:"column:name"`
	Age  int    `gorm:"column:age"`
}

func (User) TableName() string {
	return "users"
}

// ExampleBase demonstrates basic CRUD operations
func ExampleBase() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Create base instance
	base := gormutil.NewBase[User](db, "id")
	ctx := context.Background()

	// Auto migrate
	err = base.AutoMigrate(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Create a user
	user := &User{ID: "1", Name: "Alice", Age: 25}
	err = base.Create(ctx, user)
	if err != nil {
		log.Fatal(err)
	}

	// Get by ID
	result, err := base.GetByID(ctx, "1")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("User: %s, Age: %d\n", result.Name, result.Age)

	// Output:
	// User: Alice, Age: 25
}

// ExampleFindWithMarkerPagination demonstrates marker-based pagination
func ExampleFindWithMarkerPagination() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Auto migrate
	err = db.AutoMigrate(&User{})
	if err != nil {
		log.Fatal(err)
	}

	// Create test data
	users := []User{
		{ID: "1", Name: "Alice", Age: 25},
		{ID: "2", Name: "Bob", Age: 30},
		{ID: "3", Name: "Charlie", Age: 35},
	}
	for _, u := range users {
		err = db.Create(&u).Error
		if err != nil {
			log.Fatal(err)
		}
	}

	// First page
	results, nextMarker, err := gormutil.FindWithMarkerPagination(
		ctx,
		gormutil.G[User](db),
		"id",
		func(u User) string { return u.ID },
		"",
		2,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Page 1: %d users, Next Marker: %s\n", len(results), nextMarker)

	// Output:
	// Page 1: 2 users, Next Marker: 3
}

// ExampleFindWithCompositePagination demonstrates composite pagination
func ExampleFindWithCompositePagination() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Auto migrate
	type Order struct {
		UserID    string `gorm:"column:user_id;primaryKey"`
		CreatedAt string `gorm:"column:created_at;primaryKey"`
		ID        string `gorm:"column:id;primaryKey"`
	}
	err = db.AutoMigrate(&Order{})
	if err != nil {
		log.Fatal(err)
	}

	// Create test data
	orders := []Order{
		{UserID: "user1", CreatedAt: "2024-01-03", ID: "id1"},
		{UserID: "user1", CreatedAt: "2024-01-02", ID: "id2"},
	}
	for _, o := range orders {
		err = db.Create(&o).Error
		if err != nil {
			log.Fatal(err)
		}
	}

	// Define pagination columns
	columns := []gormutil.CompositePaginationColumn{
		{ColumnName: "user_id", Desc: false},
		{ColumnName: "created_at", Desc: true},
		{ColumnName: "id", Desc: false},
	}

	markerExtractor := func(o Order) gormutil.CompositePaginationMarker {
		return gormutil.CompositePaginationMarker{
			Values: []string{o.UserID, o.CreatedAt, o.ID},
		}
	}

	// Query
	results, nextMarker, err := gormutil.FindWithCompositePagination(
		ctx,
		gormutil.G[Order](db),
		columns,
		markerExtractor,
		gormutil.CompositePaginationMarker{},
		10,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d orders\n", len(results))
	if len(nextMarker.Values) > 0 {
		fmt.Println("Has next page")
	} else {
		fmt.Println("No more pages")
	}

	// Output:
	// Found 2 orders
	// No more pages
}

// ExampleG demonstrates chain operations
func ExampleG() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Auto migrate
	err = db.AutoMigrate(&User{})
	if err != nil {
		log.Fatal(err)
	}

	// Create test data
	users := []User{
		{ID: "1", Name: "Alice", Age: 25},
		{ID: "2", Name: "Bob", Age: 30},
		{ID: "3", Name: "Charlie", Age: 20},
	}
	for _, u := range users {
		err = db.Create(&u).Error
		if err != nil {
			log.Fatal(err)
		}
	}

	// Chain query
	results, err := gormutil.G[User](db).
		Where("age > ?", 21).
		Order("age ASC").
		Limit(2).
		Find(ctx)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d users with age > 21\n", len(results))

	// Output:
	// Found 2 users with age > 21
}
