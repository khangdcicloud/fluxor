package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Stats struct {
	TotalLines     int
	GoSourceLines  int
	GoSourceFiles  int
	GoTestLines    int
	GoTestFiles    int
	TSJSLines      int
	TSJSFiles      int
	TotalFiles     int
	TotalCodeLines int // Excluding tests
	TotalWithTests int // Including tests
}

func main() {
	stats := &Stats{}
	rootDir := "."

	excludedDirs := map[string]bool{
		"node_modules": true,
		"vendor":       true,
		".git":         true,
		"dist":         true,
		"coverage":     true,
		".cursor":      true,
		"bin":          true,
		"tmp":          true,
	}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip excluded directories
		if info.IsDir() {
			dirName := filepath.Base(path)
			if excludedDirs[dirName] {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		baseName := filepath.Base(path)

		// Count Go files
		if ext == ".go" {
			lines, err := countLines(path)
			if err != nil {
				return nil
			}

			stats.TotalLines += lines
			stats.TotalWithTests += lines

			if strings.HasSuffix(baseName, "_test.go") {
				stats.GoTestLines += lines
				stats.GoTestFiles++
			} else {
				stats.GoSourceLines += lines
				stats.GoSourceFiles++
				stats.TotalCodeLines += lines
			}
			stats.TotalFiles++
		}

		// Count TypeScript/JavaScript files
		if ext == ".ts" || ext == ".js" || ext == ".tsx" || ext == ".jsx" {
			lines, err := countLines(path)
			if err != nil {
				return nil
			}

			stats.TSJSLines += lines
			stats.TSJSFiles++
			stats.TotalLines += lines
			stats.TotalCodeLines += lines
			stats.TotalWithTests += lines
			stats.TotalFiles++
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
		os.Exit(1)
	}

	// Generate report
	report := fmt.Sprintf(`Thống kê dòng code dự án Fluxor
Tổng quan
Tổng số dòng code: %d dòng
Chi tiết theo loại file
Go (source code):
Dòng code: %d dòng
Số file: %d files
Dòng test: %d dòng
Số file test: %d files
TypeScript/JavaScript:
Dòng code: %d dòng
Số file: %d files
Tổng kết
Tổng số file code: %d files
Tổng dòng code (không bao gồm test): %d dòng
Tổng dòng code (bao gồm test): %d dòng
Lưu ý: Đã loại trừ các thư mục node_modules, vendor, .git, dist, và coverage.
Dự án chủ yếu là Go (~%.1f%% codebase) với một phần nhỏ TypeScript/JavaScript cho web frontend.`,
		stats.TotalWithTests,
		stats.GoSourceLines,
		stats.GoSourceFiles,
		stats.GoTestLines,
		stats.GoTestFiles,
		stats.TSJSLines,
		stats.TSJSFiles,
		stats.TotalFiles,
		stats.TotalCodeLines,
		stats.TotalWithTests,
		float64(stats.GoSourceLines+stats.GoTestLines)/float64(stats.TotalWithTests)*100)

	fmt.Println(report)

	// Write to statistic.log
	err = os.WriteFile("statistic.log", []byte(report), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing statistic.log: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✅ Đã cập nhật statistic.log")
}

func countLines(filePath string) (int, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(content), "\n")
	return len(lines), nil
}
