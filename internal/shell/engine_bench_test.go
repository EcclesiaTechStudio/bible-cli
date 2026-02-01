package shell

import (
	"os"
	"testing"
)

func BenchmarkSearch(b *testing.B) {
	db := getMockDB()
	engine := New(db)

	// Suppress output during benchmarking
	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = old }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.doGrep("the")
	}
}

func BenchmarkNavigation(b *testing.B) {
	db := getMockDB()
	engine := New(db)

	// Suppress output during benchmarking
	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = old }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.doCD("john")
		engine.doCD("/")
	}
}

func BenchmarkReadVerse(b *testing.B) {
	db := getMockDB()
	engine := New(db)
	engine.Path = []string{"NT", "John"}

	// Suppress output during benchmarking
	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = old }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.doCat("3:16")
	}
}

func BenchmarkBookmarkSave(b *testing.B) {
	db := getMockDB()
	engine := New(db)
	engine.Path = []string{"NT", "John"}

	// Suppress output during benchmarking
	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = old }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.saveBookmark("testmark")
	}
}

func BenchmarkMultiRefRead(b *testing.B) {
	db := getMockDB()
	engine := New(db)

	// Suppress output during benchmarking
	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = old }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.RunCommand("cat Genesis 1:1 + John 3:16")
	}
}

func BenchmarkGrepWholeBible(b *testing.B) {
	db := getMockDB()
	engine := New(db)
	engine.Path = []string{} // Search from root

	// Suppress output during benchmarking
	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = old }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.doGrep("god")
	}
}

func BenchmarkTeleport(b *testing.B) {
	db := getMockDB()
	engine := New(db)

	// Suppress output during benchmarking
	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = old }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.doCD("genesis")
		engine.doCD("john")
	}
}
