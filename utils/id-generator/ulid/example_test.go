package ulid_test

import (
	"fmt"
	"time"

	"github.com/aisgo/ais-go-pkg/utils/id-generator/ulid"
)

/* ========================================================================
 * ULID Generator Examples - 使用示例
 * ======================================================================== */

// Example_basic 基础使用
func Example_basic() {
	// 生成 ULID
	id := ulid.Generate()
	fmt.Println(len(id.String()))
	fmt.Println(ulid.IsZero(id))

	// Output:
	// 26
	// false
}

// Example_string 直接生成字符串格式
func Example_string() {
	str := ulid.GenerateString()
	fmt.Println(len(str))

	// Output:
	// 26
}

// Example_parse 解析 ULID 字符串
func Example_parse() {
	const s = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	parsed, err := ulid.Parse(s)
	if err != nil {
		panic(err)
	}
	fmt.Println(parsed.String() == s)

	// Output:
	// true
}

// Example_time 提取时间戳
func Example_time() {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	id := ulid.GenerateWithTime(t0)
	timestamp := ulid.Time(id).UTC()

	fmt.Printf("时间: %s\n", timestamp.Format(time.RFC3339))

	// Output:
	// 时间: 2024-01-01T00:00:00Z
}

// Example_withTime 使用指定时间生成
func Example_withTime() {
	// 使用特定时间生成 ULID（如数据迁移场景）
	specificTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	id := ulid.GenerateWithTime(specificTime)

	fmt.Printf("时间: %s\n", ulid.Time(id).UTC().Format(time.RFC3339))

	// Output:
	// 时间: 2024-01-01T00:00:00Z
}

// Example_compare 比较 ULID
func Example_compare() {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	id1 := ulid.GenerateWithTime(t0)
	id2 := ulid.GenerateWithTime(t0.Add(time.Millisecond))

	result := ulid.Compare(id1, id2)
	fmt.Println(result)

	// Output:
	// -1
}

// Example_batch 批量生成
func Example_batch() {
	// 批量生成 ULID（高性能场景）
	ids := ulid.GenerateBatch(5)

	ok := true
	for i := 1; i < len(ids); i++ {
		if ulid.Compare(ids[i-1], ids[i]) >= 0 {
			ok = false
			break
		}
	}

	fmt.Println(len(ids))
	fmt.Println(ok)

	// Output:
	// 5
	// true
}

// Example_generator 使用独立生成器
func Example_generator() {
	// 创建独立的生成器实例
	gen := ulid.NewGenerator(nil)

	str := gen.GenerateString()
	fmt.Println(len(str))

	// Output:
	// 26
}

// Example_database 数据库模型中使用
func Example_database() {
	// 在 GORM 模型中使用
	type User struct {
		ID        string    `gorm:"type:char(26);primaryKey"`
		Name      string    `gorm:"size:100"`
		CreatedAt time.Time `gorm:"autoCreateTime"`
	}

	// 创建新用户时自动生成 ULID
	user := User{
		ID:   ulid.GenerateString(),
		Name: "Alice",
	}

	fmt.Println(len(user.ID))
	fmt.Println(user.Name)

	// Output:
	// 26
	// Alice
}

// Example_sorting 排序特性
func Example_sorting() {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	id1 := ulid.GenerateWithTime(t0)
	id2 := ulid.GenerateWithTime(t0.Add(time.Millisecond))
	id3 := ulid.GenerateWithTime(t0.Add(2 * time.Millisecond))

	ok := ulid.Compare(id1, id2) < 0 && ulid.Compare(id2, id3) < 0
	fmt.Println(ok)

	// Output:
	// true
}

// Example_zero 零值检查
func Example_zero() {
	zero := ulid.Zero()
	id := ulid.Generate()

	fmt.Println(ulid.IsZero(zero))
	fmt.Println(ulid.IsZero(id))

	// Output:
	// true
	// false
}

// Example_toUUID ULID 转 UUID
func Example_toUUID() {
	// 生成 ULID
	id := ulid.Generate()

	// 转换为 UUID 字符串
	uuidStr := ulid.ToUUIDString(id)
	fmt.Println(len(uuidStr))

	// Output:
	// 36
}

// Example_fromUUID UUID 转 ULID
func Example_fromUUID() {
	// 从 UUID 字符串创建
	uuidStr := "550e8400-e29b-41d4-a716-446655440000"
	id, err := ulid.FromUUIDString(uuidStr)
	if err != nil {
		panic(err)
	}

	fmt.Println(ulid.ToUUIDString(id) == uuidStr)

	// Output:
	// true
}

// Example_uuidRoundTrip UUID 往返转换
func Example_uuidRoundTrip() {
	// ULID -> UUID -> ULID
	original := ulid.GenerateWithTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	uuidStr := ulid.ToUUIDString(original)
	converted, _ := ulid.FromUUIDString(uuidStr)

	fmt.Println(ulid.Compare(original, converted) == 0)

	// Output:
	// true
}

// Example_migration 系统迁移场景
func Example_migration() {
	// 场景：从 UUID 系统迁移到 ULID 系统

	// 旧系统使用 UUID
	oldUUIDs := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	fmt.Println("迁移 UUID 到 ULID:")
	for i, uuidStr := range oldUUIDs {
		id, _ := ulid.FromUUIDString(uuidStr)
		fmt.Printf("%d. UUID: %s -> ULID: %s\n", i+1, uuidStr, id.String())
	}

	// Output:
	// 迁移 UUID 到 ULID:
	// 1. UUID: 550e8400-e29b-41d4-a716-446655440000 -> ULID: 2N1T201RMV87AAE5J4CSAM8000
	// 2. UUID: 6ba7b810-9dad-11d1-80b4-00c04fd430c8 -> ULID: 3BMYW117DD278R1D00R17X8C68
}

// Example_integration 与 UUID 系统集成
func Example_integration() {
	// 场景：内部使用 ULID，对外接口提供 UUID

	// 内部生成 ULID
	internalID := ulid.Generate()

	// API 响应使用 UUID 格式
	apiResponse := map[string]string{
		"id":   ulid.ToUUIDString(internalID),
		"type": "user",
	}

	fmt.Println(len(internalID.String()))
	fmt.Println(len(apiResponse["id"]))

	// Output:
	// 26
	// 36
}
