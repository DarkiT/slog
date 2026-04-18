package xxhash

import (
	"testing"
)

// 测试向量来自 xxHash 官方参考实现
// https://github.com/Cyan4973/xxHash/blob/dev/doc/xxhash_spec.md
var testCases = []struct {
	name     string
	input    string
	expected uint64
}{
	{"empty", "", 0xef46db3751d8e999},
	{"single_byte", "a", 0xd24ec4f1a98c6e5b},
	{"short_string", "abc", 0x44bc2cf5ad770999},
	{"hello", "Hello, World!", 0x7b06c531ea43e89f},
	{"32_bytes", "12345678901234567890123456789012", 0x2d4ed0c8f1e3e8a1},
}

func TestSum64_Empty(t *testing.T) {
	result := Sum64(nil)
	expected := uint64(0xef46db3751d8e999)
	if result != expected {
		t.Errorf("Sum64(nil) = %#x, want %#x", result, expected)
	}

	result = Sum64([]byte{})
	if result != expected {
		t.Errorf("Sum64([]) = %#x, want %#x", result, expected)
	}
}

func TestSum64_SmallInput(t *testing.T) {
	// 测试小于 32 字节的输入
	inputs := [][]byte{
		[]byte("a"),
		[]byte("ab"),
		[]byte("abc"),
		[]byte("abcd"),
		[]byte("abcdefgh"),         // 8 bytes
		[]byte("abcdefghijklmnop"), // 16 bytes
	}

	for _, input := range inputs {
		result := Sum64(input)
		// 验证结果非零且一致
		if result == 0 {
			t.Errorf("Sum64(%q) returned 0", input)
		}
		// 验证多次调用结果一致
		if Sum64(input) != result {
			t.Errorf("Sum64(%q) not consistent", input)
		}
	}
}

func TestSum64_LargeInput(t *testing.T) {
	// 测试大于 32 字节的输入（触发完整的 xxHash 算法）
	input := make([]byte, 100)
	for i := range input {
		input[i] = byte(i)
	}

	result := Sum64(input)
	if result == 0 {
		t.Error("Sum64(100 bytes) returned 0")
	}

	// 验证一致性
	if Sum64(input) != result {
		t.Error("Sum64(100 bytes) not consistent")
	}
}

func TestSum64_VeryLargeInput(t *testing.T) {
	// 测试非常大的输入
	input := make([]byte, 10000)
	for i := range input {
		input[i] = byte(i % 256)
	}

	result := Sum64(input)
	if result == 0 {
		t.Error("Sum64(10000 bytes) returned 0")
	}

	// 验证一致性
	if Sum64(input) != result {
		t.Error("Sum64(10000 bytes) not consistent")
	}
}

func TestSum64String_Consistency(t *testing.T) {
	testStrings := []string{
		"",
		"a",
		"hello",
		"Hello, World!",
		"这是一个中文字符串测试",
		"1234567890123456789012345678901234567890",
	}

	for _, s := range testStrings {
		byteResult := Sum64([]byte(s))
		stringResult := Sum64String(s)

		if byteResult != stringResult {
			t.Errorf("Sum64(%q) = %#x, Sum64String(%q) = %#x, want equal",
				s, byteResult, s, stringResult)
		}
	}
}

func TestSum64String_Empty(t *testing.T) {
	result := Sum64String("")
	expected := Sum64(nil)
	if result != expected {
		t.Errorf("Sum64String(\"\") = %#x, want %#x", result, expected)
	}
}

func TestSum64_Deterministic(t *testing.T) {
	// 验证相同输入总是产生相同输出
	input := []byte("deterministic test input")
	first := Sum64(input)

	for i := 0; i < 100; i++ {
		if Sum64(input) != first {
			t.Fatalf("Sum64 not deterministic at iteration %d", i)
		}
	}
}

func TestSum64_DifferentInputs(t *testing.T) {
	// 验证不同输入产生不同哈希（碰撞概率极低）
	inputs := []string{
		"input1",
		"input2",
		"input3",
		"Input1",  // 大小写不同
		"input1 ", // 多一个空格
	}

	hashes := make(map[uint64]string)
	for _, input := range inputs {
		h := Sum64String(input)
		if existing, ok := hashes[h]; ok {
			t.Errorf("Hash collision: %q and %q both hash to %#x", input, existing, h)
		}
		hashes[h] = input
	}
}

func TestSum64_BoundaryConditions(t *testing.T) {
	// 测试边界条件：31, 32, 33 字节
	for size := 30; size <= 34; size++ {
		input := make([]byte, size)
		for i := range input {
			input[i] = byte(i)
		}

		result := Sum64(input)
		if result == 0 {
			t.Errorf("Sum64(%d bytes) returned 0", size)
		}

		// 验证一致性
		if Sum64(input) != result {
			t.Errorf("Sum64(%d bytes) not consistent", size)
		}
	}
}

func TestSum64_AllZeros(t *testing.T) {
	// 测试全零输入
	input := make([]byte, 64)
	result := Sum64(input)
	if result == 0 {
		t.Error("Sum64(64 zeros) should not return 0")
	}
}

func TestSum64_AllOnes(t *testing.T) {
	// 测试全 0xFF 输入
	input := make([]byte, 64)
	for i := range input {
		input[i] = 0xFF
	}
	result := Sum64(input)
	if result == 0 {
		t.Error("Sum64(64 0xFF) should not return 0")
	}
}

// 基准测试
func BenchmarkSum64_8(b *testing.B) {
	input := []byte("12345678")
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sum64(input)
	}
}

func BenchmarkSum64_32(b *testing.B) {
	input := []byte("12345678901234567890123456789012")
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sum64(input)
	}
}

func BenchmarkSum64_64(b *testing.B) {
	input := make([]byte, 64)
	for i := range input {
		input[i] = byte(i)
	}
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sum64(input)
	}
}

func BenchmarkSum64_1024(b *testing.B) {
	input := make([]byte, 1024)
	for i := range input {
		input[i] = byte(i)
	}
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sum64(input)
	}
}

func BenchmarkSum64String_Short(b *testing.B) {
	s := "hello world"
	b.SetBytes(int64(len(s)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sum64String(s)
	}
}

func BenchmarkSum64String_Long(b *testing.B) {
	s := "this is a longer string that will test the performance of the xxhash algorithm"
	b.SetBytes(int64(len(s)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sum64String(s)
	}
}
