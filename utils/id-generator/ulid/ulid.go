package ulid

import (
	"crypto/rand"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

/* ========================================================================
 * ULID Generator - ULID 生成器
 * ========================================================================
 * 职责: 生成分布式唯一 ID
 * 特点:
 *   - 128 位唯一性
 *   - 字典序排序（按时间戳）
 *   - URL 安全（Crockford's Base32）
 *   - 大小写不敏感
 *   - 固定 26 字符长度
 * ID 结构:
 *   - 48 位时间戳（毫秒级，可用至 10889 年）
 *   - 80 位随机数（加密安全）
 *
 * 优势:
 *   - 无需配置节点 ID
 *   - 天然时间排序
 *   - 适合数据库索引
 *   - 人类可读性更好
 * ======================================================================== */

var (
	globalEntropy io.Reader
	once          sync.Once
	mu            sync.Mutex
)

// Generator ULID 生成器
type Generator struct {
	entropy io.Reader
	mu      sync.Mutex
}

// NewGenerator 创建新的 ULID 生成器
// entropy: 熵源（随机数生成器），传 nil 则使用 crypto/rand.Reader
//
// 如果需要自定义熵源（如测试场景），可以使用此方法。
// 否则建议直接使用全局函数 Generate()。
func NewGenerator(entropy io.Reader) *Generator {
	if entropy == nil {
		entropy = rand.Reader
	}
	// 使用 oklog/ulid 的 Monotonic 熵源，保证同一毫秒内按生成顺序递增（更利于排序/索引）。
	// 注意：Monotonic 熵源本身不是并发安全的，因此需要配合互斥锁使用。
	if _, ok := entropy.(ulid.MonotonicEntropy); !ok {
		entropy = ulid.Monotonic(entropy, 0)
	}
	return &Generator{entropy: entropy}
}

// Generate 生成 ULID
func (g *Generator) Generate() ulid.ULID {
	g.mu.Lock()
	defer g.mu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), g.entropy)
}

// GenerateString 生成 ULID（字符串格式）
func (g *Generator) GenerateString() string {
	return g.Generate().String()
}

// GenerateWithTime 使用指定时间生成 ULID
// 适用于需要精确控制时间戳的场景（如数据迁移）
func (g *Generator) GenerateWithTime(t time.Time) ulid.ULID {
	g.mu.Lock()
	defer g.mu.Unlock()
	return ulid.MustNew(ulid.Timestamp(t), g.entropy)
}

// ========================================================================
// 全局函数（使用加密安全随机源）
// ========================================================================

// initEntropy 初始化全局熵源（仅执行一次）
func initEntropy() {
	entropy := rand.Reader
	if _, ok := entropy.(ulid.MonotonicEntropy); !ok {
		entropy = ulid.Monotonic(entropy, 0)
	}
	globalEntropy = entropy
}

// Generate 生成 ULID
// 使用 crypto/rand.Reader 作为熵源，保证加密安全
//
// 示例:
//
//	id := ulid.Generate()
//	fmt.Println(id.String()) // 01HN3K8X9FQZM6Y8VWXQR2JNPT
func Generate() ulid.ULID {
	once.Do(initEntropy)

	mu.Lock()
	defer mu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), globalEntropy)
}

// GenerateString 生成 ULID（字符串格式）
func GenerateString() string {
	return Generate().String()
}

// GenerateWithTime 使用指定时间生成 ULID
func GenerateWithTime(t time.Time) ulid.ULID {
	once.Do(initEntropy)

	mu.Lock()
	defer mu.Unlock()
	return ulid.MustNew(ulid.Timestamp(t), globalEntropy)
}

// MustParse 解析 ULID 字符串，失败时 panic
func MustParse(s string) ulid.ULID {
	id, err := ulid.Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}

// Parse 解析 ULID 字符串
func Parse(s string) (ulid.ULID, error) {
	return ulid.Parse(s)
}

// ========================================================================
// 辅助函数
// ========================================================================

// Time 提取 ULID 中的时间戳
func Time(id ulid.ULID) time.Time {
	return ulid.Time(id.Time())
}

// Compare 比较两个 ULID
// 返回值: -1 (a < b), 0 (a == b), 1 (a > b)
func Compare(a, b ulid.ULID) int {
	return a.Compare(b)
}

// Zero 返回零值 ULID
func Zero() ulid.ULID {
	return ulid.ULID{}
}

// IsZero 检查 ULID 是否为零值
func IsZero(id ulid.ULID) bool {
	return id.Compare(ulid.ULID{}) == 0
}

// ========================================================================
// 批量生成（高性能场景）
// ========================================================================

// GenerateBatch 批量生成 ULID
// count: 生成数量
//
// 注意: 使用 Monotonic 熵源时，同一毫秒内生成的 ULID 会按生成顺序递增
func GenerateBatch(count int) []ulid.ULID {
	if count <= 0 {
		return []ulid.ULID{}
	}
	once.Do(initEntropy)

	mu.Lock()
	defer mu.Unlock()

	ids := make([]ulid.ULID, count)
	now := ulid.Timestamp(time.Now())

	for i := 0; i < count; i++ {
		ids[i] = ulid.MustNew(now, globalEntropy)
	}

	return ids
}

// GenerateBatchString 批量生成 ULID（字符串格式）
func GenerateBatchString(count int) []string {
	ids := GenerateBatch(count)
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = id.String()
	}
	return strs
}

// ========================================================================
// ULID ⇄ UUID 互转
// ========================================================================

// ToUUID 将 ULID 转换为 UUID
// ULID 和 UUID 都是 128 位，可以直接转换字节数组
//
// 注意: 转换后的 UUID 不保留 ULID 的时间排序特性
// 适用场景: 与需要 UUID 格式的系统集成
func ToUUID(id ulid.ULID) uuid.UUID {
	var u uuid.UUID
	copy(u[:], id[:])
	return u
}

// FromUUID 将 UUID 转换为 ULID
// 直接复制 128 位字节数组
//
// 注意: 转换后的 ULID 可能不包含有效的时间戳
// 适用场景: 从 UUID 系统迁移到 ULID
func FromUUID(u uuid.UUID) ulid.ULID {
	var id ulid.ULID
	copy(id[:], u[:])
	return id
}

// ToUUIDString 将 ULID 转换为 UUID 字符串格式
// 返回标准 UUID 格式: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
func ToUUIDString(id ulid.ULID) string {
	return ToUUID(id).String()
}

// FromUUIDString 从 UUID 字符串创建 ULID
// 支持标准 UUID 格式和无连字符格式
func FromUUIDString(s string) (ulid.ULID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return ulid.ULID{}, fmt.Errorf("invalid UUID string: %w", err)
	}
	return FromUUID(u), nil
}

// MustFromUUIDString 从 UUID 字符串创建 ULID，失败时 panic
func MustFromUUIDString(s string) ulid.ULID {
	id, err := FromUUIDString(s)
	if err != nil {
		panic(err)
	}
	return id
}
