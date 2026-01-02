# ULID Generator

基于 [oklog/ulid](https://github.com/oklog/ulid) 的 ULID 生成器封装。

## 特性

- ✅ **时间排序** - 前 48 位是毫秒级时间戳，天然按时间排序
- ✅ **URL 安全** - 使用 Crockford's Base32 编码，无特殊字符
- ✅ **大小写不敏感** - 避免混淆字符（0/O, 1/I/l）
- ✅ **固定长度** - 26 个字符，便于存储和索引
- ✅ **128 位唯一性** - 与 UUID 相同的唯一性保证
- ✅ **加密安全** - 使用 `crypto/rand` 作为熵源
- ✅ **并发安全** - 内置互斥锁保护
- ✅ **无需配置** - 不需要节点 ID 等配置

## ULID vs Snowflake

| 特性 | ULID | Snowflake |
|------|------|-----------|
| 长度 | 26 字符 (128 位) | 19 位数字 (64 位) |
| 排序 | 字典序 = 时间序 | 数值序 = 时间序 |
| 配置 | 无需配置 | 需要节点 ID |
| 格式 | Base32 字符串 | 整数 |
| 人类可读性 | 较好 | 较差 |
| 数据库类型 | CHAR(26) | BIGINT |
| URL 安全 | ✅ | ❌（需编码） |

## 快速开始

### 基础使用

```go
import "github.com/aisgo/ais-go-pkg/utils/id-generator/ulid"

// 生成 ULID
id := ulid.Generate()
fmt.Println(id.String()) // 01HN3K8X9FQZM6Y8VWXQR2JNPT

// 直接生成字符串
str := ulid.GenerateString()
```

### 解析和比较

```go
// 解析 ULID 字符串
id, err := ulid.Parse("01HN3K8X9FQZM6Y8VWXQR2JNPT")
if err != nil {
    panic(err)
}

// 提取时间戳
timestamp := ulid.Time(id)
fmt.Println(timestamp.Format(time.RFC3339))

// 比较两个 ULID
result := ulid.Compare(id1, id2)
// -1: id1 < id2, 0: id1 == id2, 1: id1 > id2
```

### 批量生成

```go
// 批量生成（高性能场景）
ids := ulid.GenerateBatch(100)

// 批量生成字符串格式
strs := ulid.GenerateBatchString(100)
```

### 使用指定时间

```go
// 适用于数据迁移等场景
specificTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
id := ulid.GenerateWithTime(specificTime)
```

### 独立生成器

```go
// 创建独立的生成器实例
gen := ulid.NewGenerator(nil)

id := gen.Generate()
str := gen.GenerateString()
```

### ULID ⇄ UUID 互转

```go
// ULID 转 UUID
id := ulid.Generate()
u := ulid.ToUUID(id)
uuidStr := ulid.ToUUIDString(id)

// UUID 转 ULID
uuidStr := "550e8400-e29b-41d4-a716-446655440000"
id, err := ulid.FromUUIDString(uuidStr)

// 往返转换（保持一致性）
original := ulid.Generate()
converted, _ := ulid.FromUUIDString(ulid.ToUUIDString(original))
// original == converted ✓
```

**使用场景：**
- 与需要 UUID 格式的系统集成
- 从 UUID 系统迁移到 ULID
- 内部使用 ULID，对外接口提供 UUID

## 在 GORM 中使用

### 方式一：使用 BaseModel

```go
import "github.com/aisgo/ais-go-pkg/repository"

type User struct {
    repository.BaseModel  // 自动包含 ULID 类型的 ID
    Name  string
    Email string
}

// 创建时会自动生成 ULID
user := &User{Name: "Alice", Email: "alice@example.com"}
db.Create(user)
// user.ID 已自动填充
```

### 方式二：自定义字段

```go
import (
    "github.com/aisgo/ais-go-pkg/utils/id-generator/ulid"
    "github.com/oklog/ulid/v2"
)

type Product struct {
    ID    ulid.ULID `gorm:"type:char(26);primaryKey"`
    Name  string
    Price float64
}

// BeforeCreate 钩子
func (p *Product) BeforeCreate(tx *gorm.DB) error {
    if ulid.IsZero(p.ID) {
        p.ID = ulid.Generate()
    }
    return nil
}
```

### 方式三：字符串类型

```go
type Order struct {
    ID        string    `gorm:"type:char(26);primaryKey"`
    UserID    string    `gorm:"type:char(26);index"`
    Amount    float64
    CreatedAt time.Time
}

func (o *Order) BeforeCreate(tx *gorm.DB) error {
    if o.ID == "" {
        o.ID = ulid.GenerateString()
    }
    return nil
}
```

## 数据库迁移

### PostgreSQL

```sql
-- 创建表时使用 CHAR(26)
CREATE TABLE users (
    id         CHAR(26) PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- 插入示例
INSERT INTO users (id, name) 
VALUES ('01HN3K8X9FQZM6Y8VWXQR2JNPT', 'Alice');

-- 按时间排序（字典序即时间序）
SELECT * FROM users ORDER BY id;
```

### MySQL

```sql
CREATE TABLE users (
    id         CHAR(26) PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

## API 参考

### 全局函数

| 函数 | 说明 |
|------|------|
| `Generate() ulid.ULID` | 生成 ULID |
| `GenerateString() string` | 生成 ULID 字符串 |
| `GenerateWithTime(t time.Time) ulid.ULID` | 使用指定时间生成 |
| `Parse(s string) (ulid.ULID, error)` | 解析 ULID 字符串 |
| `MustParse(s string) ulid.ULID` | 解析 ULID，失败时 panic |
| `Time(id ulid.ULID) time.Time` | 提取时间戳 |
| `Compare(a, b ulid.ULID) int` | 比较两个 ULID |
| `Zero() ulid.ULID` | 返回零值 ULID |
| `IsZero(id ulid.ULID) bool` | 检查是否为零值 |
| `GenerateBatch(count int) []ulid.ULID` | 批量生成 |
| `GenerateBatchString(count int) []string` | 批量生成字符串 |
| `ToUUID(id ulid.ULID) uuid.UUID` | 将 ULID 转换为 UUID |
| `FromUUID(u uuid.UUID) ulid.ULID` | 将 UUID 转换为 ULID |
| `ToUUIDString(id ulid.ULID) string` | 将 ULID 转换为 UUID 字符串 |
| `FromUUIDString(s string) (ulid.ULID, error)` | 从 UUID 字符串创建 ULID |
| `MustFromUUIDString(s string) ulid.ULID` | 从 UUID 字符串创建 ULID，失败时 panic |

### Generator 方法

| 方法 | 说明 |
|------|------|
| `NewGenerator(entropy io.Reader) *Generator` | 创建生成器 |
| `Generate() ulid.ULID` | 生成 ULID |
| `GenerateString() string` | 生成 ULID 字符串 |
| `GenerateWithTime(t time.Time) ulid.ULID` | 使用指定时间生成 |

## 性能测试

```bash
go test -bench=. -benchmem ./utils/id-generator/ulid/
```

典型性能指标：
- 单次生成：~500 ns/op
- 批量生成 100 个：~50 μs/op
- 并发生成：~800 ns/op

## 最佳实践

### ✅ 推荐

```go
// 1. 使用全局函数（简单场景）
id := ulid.Generate()

// 2. 在 GORM 钩子中自动生成
func (m *Model) BeforeCreate(tx *gorm.DB) error {
    if ulid.IsZero(m.ID) {
        m.ID = ulid.Generate()
    }
    return nil
}

// 3. 批量生成（高性能场景）
ids := ulid.GenerateBatch(1000)
```

### ❌ 避免

```go
// 不要在循环中频繁创建生成器
for i := 0; i < 1000; i++ {
    gen := ulid.NewGenerator(nil) // ❌ 低效
    id := gen.Generate()
}

// 应该使用全局函数或批量生成
ids := ulid.GenerateBatch(1000) // ✅ 高效
```

## 常见问题

### Q: ULID 和 UUID 有什么区别？

**A:** 主要区别：
- ULID 按时间排序，UUID v4 完全随机
- ULID 26 字符，UUID 36 字符（含连字符）
- ULID 大小写不敏感，UUID 区分大小写
- ULID 更适合作为数据库主键（索引友好）

### Q: 如何从 Snowflake 迁移到 ULID？

**A:** 迁移步骤：
1. 新表使用 ULID
2. 旧表保持 Snowflake ID
3. 关联表使用联合查询
4. 逐步迁移数据（可选）

### Q: ULID 的唯一性如何保证？

**A:** 
- 时间戳部分：48 位毫秒级时间戳
- 随机部分：80 位加密安全随机数
- 碰撞概率：极低（2^80 种可能）

### Q: 可以在分布式系统中使用吗？

**A:** 可以！ULID 不需要节点 ID 配置，天然支持分布式环境。每个实例独立生成，通过时间戳 + 随机数保证唯一性。

## 相关资源

- [ULID 规范](https://github.com/ulid/spec)
- [oklog/ulid 库](https://github.com/oklog/ulid)
- [Crockford's Base32](https://www.crockford.com/base32.html)

## License

MIT
