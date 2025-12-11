# gorm-generics-util

[![CI](https://github.com/zhangzqs/gorm-generics-util/actions/workflows/ci.yml/badge.svg)](https://github.com/zhangzqs/gorm-generics-util/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/zhangzqs/gorm-generics-util.svg)](https://pkg.go.dev/github.com/zhangzqs/gorm-generics-util)
[![Go Report Card](https://goreportcard.com/badge/github.com/zhangzqs/gorm-generics-util)](https://goreportcard.com/report/github.com/zhangzqs/gorm-generics-util)

基于泛型的GORM相关常用查询封装

## 功能特性

- **类型安全的GORM封装**: 使用Go泛型提供类型安全的数据库操作
- **基础CRUD操作**: 提供常用的增删改查操作封装
- **标记分页**: 支持单列标记分页（Marker Pagination）
- **复合标记分页**: 支持多列组合的标记分页，适用于联合主键场景
- **链式查询**: 提供流畅的链式查询接口

## 安装

```bash
go get github.com/zhangzqs/gorm-generics-util
```

## 快速开始

### 基础CRUD操作

```go
package main

import (
    "context"
    "log"
    
    gormutil "github.com/zhangzqs/gorm-generics-util"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

type User struct {
    ID   string `gorm:"column:id;primaryKey"`
    Name string `gorm:"column:name"`
}

func main() {
    db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
    if err != nil {
        log.Fatal(err)
    }
    
    // 创建基础操作实例
    base := gormutil.NewBase[User](db, "id")
    ctx := context.Background()
    
    // 自动迁移
    if err := base.AutoMigrate(ctx); err != nil {
        log.Fatal(err)
    }
    
    // 创建记录
    user := &User{ID: "1", Name: "Alice"}
    if err := base.Create(ctx, user); err != nil {
        log.Fatal(err)
    }
    
    // 根据ID查询
    result, err := base.GetByID(ctx, "1")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("User: %+v", result)
    
    // 删除记录
    deleted, err := base.PhysicalDelete(ctx, "1")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Deleted: %v", deleted)
}
```

### 单列标记分页

```go
package main

import (
    "context"
    "log"
    
    gormutil "github.com/zhangzqs/gorm-generics-util"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

type Item struct {
    ID   string `gorm:"column:id;primaryKey"`
    Name string `gorm:"column:name"`
}

func main() {
    db, _ := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
    ctx := context.Background()
    
    // 标记分页
    results, nextMarker, err := gormutil.FindWithMarkerPagination(
        ctx,
        gormutil.G[Item](db),
        "id",                              // 标记列名
        func(item Item) string {           // 标记提取器
            return item.ID
        },
        "",                                // 起始标记（空表示第一页）
        10,                                // 每页数量
    )
    
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Results: %+v", results)
    log.Printf("Next Marker: %s", nextMarker)
}
```

### 复合标记分页

适用于多列联合主键或需要多列排序的场景：

```go
package main

import (
    "context"
    "log"
    
    gormutil "github.com/zhangzqs/gorm-generics-util"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

type OrderItem struct {
    UserID    string `gorm:"column:user_id;primaryKey"`
    CreatedAt string `gorm:"column:created_at;primaryKey"`
    ID        string `gorm:"column:id;primaryKey"`
    Name      string `gorm:"column:name"`
}

func main() {
    db, _ := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
    ctx := context.Background()
    
    // 定义分页列配置：user_id ASC, created_at DESC, id ASC
    columns := []gormutil.CompositePaginationColumn{
        {ColumnName: "user_id", Desc: false},
        {ColumnName: "created_at", Desc: true},
        {ColumnName: "id", Desc: false},
    }
    
    // 标记提取器
    markerExtractor := func(item OrderItem) gormutil.CompositePaginationMarker {
        return gormutil.CompositePaginationMarker{
            Values: []string{item.UserID, item.CreatedAt, item.ID},
        }
    }
    
    // 执行复合分页查询
    results, nextMarker, err := gormutil.FindWithCompositePagination(
        ctx,
        gormutil.G[OrderItem](db),
        columns,
        markerExtractor,
        gormutil.CompositePaginationMarker{}, // 空标记表示第一页
        10,
    )
    
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Results: %+v", results)
    
    // 编码nextMarker为字符串（可用于API返回）
    if len(nextMarker.Values) > 0 {
        encoded, _ := nextMarker.Encode()
        log.Printf("Next Marker: %s", encoded)
    }
}
```

### 链式查询

```go
package main

import (
    "context"
    "log"
    
    gormutil "github.com/zhangzqs/gorm-generics-util"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

type User struct {
    ID   string `gorm:"column:id;primaryKey"`
    Name string `gorm:"column:name"`
    Age  int    `gorm:"column:age"`
}

func main() {
    db, _ := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
    ctx := context.Background()
    
    // 链式查询
    results, err := gormutil.G[User](db).
        Where("age > ?", 18).
        Order("name ASC").
        Limit(10).
        Find(ctx)
    
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Results: %+v", results)
    
    // 查询单条记录
    user, err := gormutil.G[User](db).
        Where("id = ?", "1").
        First(ctx)
    
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("User: %+v", user)
}
```

## API文档

### 基础CRUD

#### `NewBase[T any](db *gorm.DB, idColumnName string, extraMigrateSQLs ...string) *Base[T]`

创建新的基础操作实例。

**参数：**
- `db`: GORM数据库实例
- `idColumnName`: ID列名称
- `extraMigrateSQLs`: 可选的额外迁移SQL语句

**方法：**
- `AutoMigrate(ctx context.Context) error`: 自动迁移表结构
- `Create(ctx context.Context, record *T) error`: 创建记录
- `GetByID(ctx context.Context, id string) (*T, error)`: 根据ID查询
- `PhysicalDelete(ctx context.Context, id string) (bool, error)`: 物理删除记录

### 标记分页

#### `FindWithMarkerPagination[Record any](...) (results []Record, nextMarker string, err error)`

单列标记分页查询。

**参数：**
- `ctx`: 上下文
- `chain`: GORM查询链
- `markerColumnName`: 标记列名（必须已建立索引且值唯一递增）
- `markerFieldExtractor`: 从记录中提取标记值的函数
- `marker`: 上一页返回的标记值，空字符串表示第一页
- `limit`: 每页返回的最大记录数

**返回：**
- `results`: 当前页的记录列表
- `nextMarker`: 下一页的标记值，空字符串表示没有更多数据
- `err`: 错误信息

### 复合标记分页

#### `FindWithCompositePagination[Record any](...) (results []Record, nextMarker CompositePaginationMarker, err error)`

多列联合标记分页查询。

**参数：**
- `ctx`: 上下文
- `chain`: GORM查询链
- `columns`: 用于分页的列配置（顺序重要，必须与索引列顺序一致）
- `markerExtractor`: 从记录中提取标记值的函数
- `marker`: 上一页返回的标记值，空值表示第一页
- `limit`: 每页返回的最大记录数

**返回：**
- `results`: 当前页的记录列表
- `nextMarker`: 下一页的标记值，空值表示没有更多数据
- `err`: 错误信息

### 链式查询

#### `G[T any](db *gorm.DB) ChainInterface[T]`

创建类型安全的GORM查询链。

**方法：**
- `Where(query interface{}, args ...interface{}) ChainInterface[T]`
- `Order(value interface{}) ChainInterface[T]`
- `Limit(limit int) ChainInterface[T]`
- `Find(ctx context.Context) ([]T, error)`
- `First(ctx context.Context) (T, error)`
- `Delete(ctx context.Context) (int64, error)`
- `Create(ctx context.Context, value *T) error`
- `Update(ctx context.Context, column string, value interface{}) error`
- `Updates(ctx context.Context, values interface{}) error`
- `Count(ctx context.Context) (int64, error)`
- `Exec(ctx context.Context, sql string, values ...interface{}) error`

## 测试

运行所有测试：

```bash
go test -v ./...
```

运行带覆盖率的测试：

```bash
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
```

查看覆盖率报告：

```bash
go tool cover -html=coverage.out
```

## 基准测试

运行基准测试：

```bash
go test -bench=. -benchmem ./...
```

## 贡献

欢迎提交问题和拉取请求！

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。
