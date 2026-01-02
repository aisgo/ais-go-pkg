package ulid

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

/* ========================================================================
 * ULID Generator Tests
 * ======================================================================== */

func TestGenerate(t *testing.T) {
	id := Generate()

	if IsZero(id) {
		t.Error("生成的 ULID 不应为零值")
	}

	str := id.String()
	if len(str) != 26 {
		t.Errorf("ULID 字符串长度应为 26，实际: %d", len(str))
	}
}

func TestGenerateString(t *testing.T) {
	str := GenerateString()

	if len(str) != 26 {
		t.Errorf("ULID 字符串长度应为 26，实际: %d", len(str))
	}

	// 验证可以解析
	_, err := Parse(str)
	if err != nil {
		t.Errorf("生成的 ULID 字符串无法解析: %v", err)
	}
}

func TestGenerateWithTime(t *testing.T) {
	testTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	id := GenerateWithTime(testTime)

	extractedTime := Time(id)

	// 允许毫秒级误差
	diff := extractedTime.Sub(testTime).Abs()
	if diff > time.Millisecond {
		t.Errorf("时间戳不匹配，期望: %v, 实际: %v", testTime, extractedTime)
	}
}

func TestParse(t *testing.T) {
	original := Generate()
	str := original.String()

	parsed, err := Parse(str)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if Compare(original, parsed) != 0 {
		t.Error("解析后的 ULID 与原始 ULID 不匹配")
	}
}

func TestMustParse(t *testing.T) {
	str := GenerateString()

	defer func() {
		if r := recover(); r != nil {
			t.Error("有效的 ULID 字符串不应 panic")
		}
	}()

	id := MustParse(str)
	if IsZero(id) {
		t.Error("解析结果不应为零值")
	}
}

func TestMustParsePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("无效的 ULID 字符串应该 panic")
		}
	}()

	MustParse("invalid-ulid")
}

func TestCompare(t *testing.T) {
	id1 := Generate()
	time.Sleep(2 * time.Millisecond)
	id2 := Generate()

	// id1 应该小于 id2（因为时间戳更早）
	if Compare(id1, id2) >= 0 {
		t.Error("后生成的 ULID 应该大于先生成的")
	}

	// 自己和自己比较应该相等
	if Compare(id1, id1) != 0 {
		t.Error("相同的 ULID 比较应该返回 0")
	}
}

func TestIsZero(t *testing.T) {
	zero := Zero()
	if !IsZero(zero) {
		t.Error("Zero() 返回的应该是零值")
	}

	id := Generate()
	if IsZero(id) {
		t.Error("生成的 ULID 不应该是零值")
	}
}

func TestGenerateBatch(t *testing.T) {
	count := 100
	ids := GenerateBatch(count)

	if len(ids) != count {
		t.Errorf("期望生成 %d 个 ULID，实际: %d", count, len(ids))
	}

	// 验证唯一性
	seen := make(map[string]bool)
	for _, id := range ids {
		str := id.String()
		if seen[str] {
			t.Errorf("发现重复的 ULID: %s", str)
		}
		seen[str] = true
	}
}

func TestGenerateBatchZeroOrNegative(t *testing.T) {
	if ids := GenerateBatch(0); len(ids) != 0 {
		t.Errorf("count=0 期望返回空切片，实际: %d", len(ids))
	}
	if ids := GenerateBatch(-1); len(ids) != 0 {
		t.Errorf("count<0 期望返回空切片，实际: %d", len(ids))
	}
}

func TestGenerateBatchString(t *testing.T) {
	count := 50
	strs := GenerateBatchString(count)

	if len(strs) != count {
		t.Errorf("期望生成 %d 个 ULID 字符串，实际: %d", count, len(strs))
	}

	for _, str := range strs {
		if len(str) != 26 {
			t.Errorf("ULID 字符串长度应为 26，实际: %d", len(str))
		}
	}
}

func TestGenerator(t *testing.T) {
	gen := NewGenerator(nil)

	id1 := gen.Generate()
	id2 := gen.Generate()

	if Compare(id1, id2) == 0 {
		t.Error("连续生成的 ULID 应该不同")
	}
}

func TestGeneratorString(t *testing.T) {
	gen := NewGenerator(nil)
	str := gen.GenerateString()

	if len(str) != 26 {
		t.Errorf("ULID 字符串长度应为 26，实际: %d", len(str))
	}
}

func TestGeneratorWithTime(t *testing.T) {
	gen := NewGenerator(nil)
	testTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	id := gen.GenerateWithTime(testTime)
	extractedTime := Time(id)

	diff := extractedTime.Sub(testTime).Abs()
	if diff > time.Millisecond {
		t.Errorf("时间戳不匹配，期望: %v, 实际: %v", testTime, extractedTime)
	}
}

func TestConcurrency(t *testing.T) {
	const goroutines = 10
	const idsPerGoroutine = 100

	results := make(chan ulid.ULID, goroutines*idsPerGoroutine)

	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < idsPerGoroutine; j++ {
				results <- Generate()
			}
		}()
	}

	seen := make(map[string]bool)
	for i := 0; i < goroutines*idsPerGoroutine; i++ {
		id := <-results
		str := id.String()
		if seen[str] {
			t.Errorf("并发场景下发现重复的 ULID: %s", str)
		}
		seen[str] = true
	}
}

func TestTimeOrdering(t *testing.T) {
	ids := make([]ulid.ULID, 10)
	for i := 0; i < 10; i++ {
		ids[i] = Generate()
		time.Sleep(time.Millisecond)
	}

	// 验证时间递增
	for i := 1; i < len(ids); i++ {
		if Compare(ids[i-1], ids[i]) >= 0 {
			t.Error("ULID 应该按时间递增排序")
		}
	}
}

func TestULIDFormat(t *testing.T) {
	str := GenerateString()

	// ULID 应该只包含 Crockford's Base32 字符
	validChars := "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	for _, c := range str {
		if !strings.ContainsRune(validChars, c) {
			t.Errorf("ULID 包含无效字符: %c", c)
		}
	}
}

/* ========================================================================
 * Benchmarks
 * ======================================================================== */

func BenchmarkGenerate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Generate()
	}
}

func BenchmarkGenerateString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateString()
	}
}

func BenchmarkGenerateBatch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateBatch(100)
	}
}

func BenchmarkParse(b *testing.B) {
	str := GenerateString()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Parse(str)
	}
}

func BenchmarkConcurrent(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Generate()
		}
	})
}

/* ========================================================================
 * ULID ⇄ UUID Conversion Tests
 * ======================================================================== */

func TestToUUID(t *testing.T) {
	id := Generate()
	u := ToUUID(id)

	// UUID 应该是有效的
	if u == (uuid.UUID{}) {
		t.Error("转换后的 UUID 不应为零值")
	}

	// 字节数组应该相同
	if string(id[:]) != string(u[:]) {
		t.Error("ULID 和 UUID 的字节数组应该相同")
	}
}

func TestFromUUID(t *testing.T) {
	u := uuid.New()
	id := FromUUID(u)

	// ULID 应该是有效的
	if IsZero(id) {
		t.Error("转换后的 ULID 不应为零值")
	}

	// 字节数组应该相同
	if string(u[:]) != string(id[:]) {
		t.Error("UUID 和 ULID 的字节数组应该相同")
	}
}

func TestULIDUUIDRoundTrip(t *testing.T) {
	// ULID -> UUID -> ULID
	original := Generate()
	u := ToUUID(original)
	converted := FromUUID(u)

	if Compare(original, converted) != 0 {
		t.Error("ULID -> UUID -> ULID 往返转换应该保持一致")
	}
}

func TestUUIDULIDRoundTrip(t *testing.T) {
	// UUID -> ULID -> UUID
	original := uuid.New()
	id := FromUUID(original)
	converted := ToUUID(id)

	if original != converted {
		t.Error("UUID -> ULID -> UUID 往返转换应该保持一致")
	}
}

func TestToUUIDString(t *testing.T) {
	id := Generate()
	uuidStr := ToUUIDString(id)

	// 验证 UUID 字符串格式 (36 字符，包含 4 个连字符)
	if len(uuidStr) != 36 {
		t.Errorf("UUID 字符串长度应为 36，实际: %d", len(uuidStr))
	}

	// 验证可以解析为 UUID
	_, err := uuid.Parse(uuidStr)
	if err != nil {
		t.Errorf("生成的 UUID 字符串无法解析: %v", err)
	}
}

func TestFromUUIDString(t *testing.T) {
	// 标准 UUID 格式
	uuidStr := "550e8400-e29b-41d4-a716-446655440000"
	id, err := FromUUIDString(uuidStr)
	if err != nil {
		t.Fatalf("解析 UUID 字符串失败: %v", err)
	}

	if IsZero(id) {
		t.Error("转换后的 ULID 不应为零值")
	}

	// 验证往返转换
	convertedUUID := ToUUIDString(id)
	if convertedUUID != uuidStr {
		t.Errorf("往返转换不一致，期望: %s, 实际: %s", uuidStr, convertedUUID)
	}
}

func TestFromUUIDStringInvalid(t *testing.T) {
	invalidUUIDs := []string{
		"invalid-uuid",
		"123",
		"",
		"550e8400-e29b-41d4-a716",
	}

	for _, invalid := range invalidUUIDs {
		_, err := FromUUIDString(invalid)
		if err == nil {
			t.Errorf("无效的 UUID 字符串应该返回错误: %s", invalid)
		}
	}
}

func TestMustFromUUIDString(t *testing.T) {
	uuidStr := "550e8400-e29b-41d4-a716-446655440000"

	defer func() {
		if r := recover(); r != nil {
			t.Error("有效的 UUID 字符串不应 panic")
		}
	}()

	id := MustFromUUIDString(uuidStr)
	if IsZero(id) {
		t.Error("转换后的 ULID 不应为零值")
	}
}

func TestMustFromUUIDStringPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("无效的 UUID 字符串应该 panic")
		}
	}()

	MustFromUUIDString("invalid-uuid")
}

func TestUUIDStringFormats(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		valid bool
	}{
		{"标准格式", "550e8400-e29b-41d4-a716-446655440000", true},
		{"大写", "550E8400-E29B-41D4-A716-446655440000", true},
		{"无连字符", "550e8400e29b41d4a716446655440000", true},
		{"URN 格式", "urn:uuid:550e8400-e29b-41d4-a716-446655440000", true},
		{"花括号", "{550e8400-e29b-41d4-a716-446655440000}", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := FromUUIDString(tc.input)
			if tc.valid {
				if err != nil {
					t.Errorf("应该能解析 %s 格式: %v", tc.name, err)
				}
				if IsZero(id) {
					t.Error("转换后的 ULID 不应为零值")
				}
			} else {
				if err == nil {
					t.Errorf("%s 格式应该返回错误", tc.name)
				}
			}
		})
	}
}

func TestConversionPreservesBytes(t *testing.T) {
	// 测试字节级别的精确转换
	id := Generate()
	originalBytes := make([]byte, 16)
	copy(originalBytes, id[:])

	// ULID -> UUID
	u := ToUUID(id)
	uuidBytes := make([]byte, 16)
	copy(uuidBytes, u[:])

	// 验证字节完全相同
	for i := 0; i < 16; i++ {
		if originalBytes[i] != uuidBytes[i] {
			t.Errorf("字节 %d 不匹配: ULID=%d, UUID=%d", i, originalBytes[i], uuidBytes[i])
		}
	}
}

/* ========================================================================
 * Benchmarks for UUID Conversion
 * ======================================================================== */

func BenchmarkToUUID(b *testing.B) {
	id := Generate()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ToUUID(id)
	}
}

func BenchmarkFromUUID(b *testing.B) {
	u := uuid.New()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		FromUUID(u)
	}
}

func BenchmarkToUUIDString(b *testing.B) {
	id := Generate()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ToUUIDString(id)
	}
}

func BenchmarkFromUUIDString(b *testing.B) {
	uuidStr := "550e8400-e29b-41d4-a716-446655440000"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		FromUUIDString(uuidStr)
	}
}
