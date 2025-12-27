package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type FileStats struct {
	TotalLines   int
	CodeLines    int
	CommentLines int
	BlankLines   int
}

type LanguageStats struct {
	Files        int
	TotalLines   int
	CodeLines    int
	CommentLines int
	BlankLines   int
	TestFiles    int
	TestLines    int
}

type ProjectStats struct {
	Languages    map[string]*LanguageStats
	TotalFiles   int
	TotalLines   int
	CodeLines    int
	CommentLines int
	BlankLines   int
	StartTime    time.Time
}

var (
	excludedDirs = map[string]bool{
		"node_modules": true,
		"vendor":       true,
		".git":         true,
		"dist":         true,
		"coverage":     true,
		".cursor":      true,
		"bin":          true,
		"tmp":          true,
		"build":        true,
		"out":          true,
		".vscode":      true,
		".idea":        true,
	}

	fileExtensions = map[string]string{
		".go":     "Go",
		".ts":     "TypeScript",
		".tsx":    "TypeScript",
		".js":     "JavaScript",
		".jsx":    "JavaScript",
		".md":     "Markdown",
		".yaml":   "YAML",
		".yml":    "YAML",
		".json":   "JSON",
		".toml":   "TOML",
		".css":    "CSS",
		".html":   "HTML",
		".sh":     "Shell",
		".bat":    "Batch",
		".ps1":    "PowerShell",
		".sql":    "SQL",
		".proto":  "Protocol Buffers",
		".vue":    "Vue",
		".svelte": "Svelte",
	}

	commentPatterns = map[string]*regexp.Regexp{
		"Go":               regexp.MustCompile(`^\s*//|^\s*/\*|\*/`),
		"TypeScript":       regexp.MustCompile(`^\s*//|^\s*/\*|\*/`),
		"JavaScript":       regexp.MustCompile(`^\s*//|^\s*/\*|\*/`),
		"CSS":              regexp.MustCompile(`^\s*/\*|\*/`),
		"HTML":             regexp.MustCompile(`<!--|-->`),
		"Shell":            regexp.MustCompile(`^\s*#`),
		"Batch":            regexp.MustCompile(`^\s*REM|^\s*::`),
		"PowerShell":       regexp.MustCompile(`^\s*#`),
		"SQL":              regexp.MustCompile(`^\s*--|^\s*/\*|\*/`),
		"Protocol Buffers": regexp.MustCompile(`^\s*//`),
	}
)

func main() {
	var (
		outputJSON   = flag.Bool("json", false, "Output in JSON format")
		outputFile   = flag.String("output", "statistic.log", "Output file path")
		excludeTests = flag.Bool("no-tests", false, "Exclude test files from statistics")
		verbose      = flag.Bool("v", false, "Verbose output")
	)
	flag.Parse()

	stats := &ProjectStats{
		Languages: make(map[string]*LanguageStats),
		StartTime: time.Now(),
	}

	rootDir := "."
	if flag.NArg() > 0 {
		rootDir = flag.Arg(0)
	}

	if *verbose {
		fmt.Printf("Scanning directory: %s\n", rootDir)
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

		// Check if file extension is supported
		lang, ok := fileExtensions[ext]
		if !ok {
			return nil
		}

		// Skip test files if requested
		if *excludeTests && isTestFile(baseName, ext) {
			return nil
		}

		fileStats, err := analyzeFile(path, lang)
		if err != nil {
			if *verbose {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
			return nil
		}

		// Initialize language stats if needed
		if stats.Languages[lang] == nil {
			stats.Languages[lang] = &LanguageStats{}
		}

		langStats := stats.Languages[lang]
		langStats.Files++
		langStats.TotalLines += fileStats.TotalLines
		langStats.CodeLines += fileStats.CodeLines
		langStats.CommentLines += fileStats.CommentLines
		langStats.BlankLines += fileStats.BlankLines

		if isTestFile(baseName, ext) {
			langStats.TestFiles++
			langStats.TestLines += fileStats.TotalLines
		}

		stats.TotalFiles++
		stats.TotalLines += fileStats.TotalLines
		stats.CodeLines += fileStats.CodeLines
		stats.CommentLines += fileStats.CommentLines
		stats.BlankLines += fileStats.BlankLines

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
		os.Exit(1)
	}

	duration := time.Since(stats.StartTime)

	// Generate and output report
	var report string
	if *outputJSON {
		report = generateJSONReport(stats, duration)
	} else {
		report = generateTextReport(stats, duration)
	}

	fmt.Println(report)

	// Write to output file
	err = os.WriteFile(*outputFile, []byte(report), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", *outputFile, err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Statistics saved to %s (scanned in %v)\n", *outputFile, duration.Round(time.Millisecond))
}

func isTestFile(filename, ext string) bool {
	if ext == ".go" {
		return strings.HasSuffix(filename, "_test.go")
	}
	if ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" {
		return strings.Contains(filename, ".test.") ||
			strings.Contains(filename, ".spec.") ||
			strings.HasSuffix(filename, ".test.ts") ||
			strings.HasSuffix(filename, ".test.js")
	}
	return false
}

func analyzeFile(filePath, language string) (*FileStats, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	stats := &FileStats{
		TotalLines: len(lines),
	}

	commentPattern, hasComments := commentPatterns[language]
	inBlockComment := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for blank lines
		if trimmed == "" {
			stats.BlankLines++
			continue
		}

		// Check for comments
		isComment := false
		if hasComments {
			if inBlockComment {
				isComment = true
				if strings.Contains(line, "*/") {
					inBlockComment = false
				}
			} else if commentPattern.MatchString(line) {
				isComment = true
				if strings.Contains(line, "/*") && !strings.Contains(line, "*/") {
					inBlockComment = true
				}
			}
		}

		if isComment {
			stats.CommentLines++
		} else {
			stats.CodeLines++
		}
	}

	return stats, nil
}

func generateTextReport(stats *ProjectStats, duration time.Duration) string {
	var sb strings.Builder

	sb.WriteString("═══════════════════════════════════════════════════════════\n")
	sb.WriteString("  📊 Fluxor Project Code Statistics\n")
	sb.WriteString("═══════════════════════════════════════════════════════════\n\n")

	// Overall summary
	sb.WriteString("📈 OVERALL SUMMARY\n")
	sb.WriteString("───────────────────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("Total Files:        %6d files\n", stats.TotalFiles))
	sb.WriteString(fmt.Sprintf("Total Lines:        %6d lines\n", stats.TotalLines))
	sb.WriteString(fmt.Sprintf("  ├─ Code Lines:    %6d lines (%.1f%%)\n",
		stats.CodeLines, float64(stats.CodeLines)/float64(stats.TotalLines)*100))
	sb.WriteString(fmt.Sprintf("  ├─ Comment Lines: %6d lines (%.1f%%)\n",
		stats.CommentLines, float64(stats.CommentLines)/float64(stats.TotalLines)*100))
	sb.WriteString(fmt.Sprintf("  └─ Blank Lines:   %6d lines (%.1f%%)\n",
		stats.BlankLines, float64(stats.BlankLines)/float64(stats.TotalLines)*100))
	sb.WriteString("\n")

	// Language breakdown
	sb.WriteString("📋 LANGUAGE BREAKDOWN\n")
	sb.WriteString("───────────────────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("%-20s %8s %10s %10s %10s %8s %10s\n",
		"Language", "Files", "Total", "Code", "Comments", "Tests", "Test Lines"))
	sb.WriteString("───────────────────────────────────────────────────────────\n")

	// Sort languages by total lines (descending)
	type langEntry struct {
		name  string
		stats *LanguageStats
	}
	var sortedLangs []langEntry
	for name, langStats := range stats.Languages {
		sortedLangs = append(sortedLangs, langEntry{name, langStats})
	}

	// Simple sort by total lines
	for i := 0; i < len(sortedLangs); i++ {
		for j := i + 1; j < len(sortedLangs); j++ {
			if sortedLangs[i].stats.TotalLines < sortedLangs[j].stats.TotalLines {
				sortedLangs[i], sortedLangs[j] = sortedLangs[j], sortedLangs[i]
			}
		}
	}

	for _, entry := range sortedLangs {
		lang := entry.name
		langStats := entry.stats
		sb.WriteString(fmt.Sprintf("%-20s %8d %10d %10d %10d %8d %10d\n",
			lang,
			langStats.Files,
			langStats.TotalLines,
			langStats.CodeLines,
			langStats.CommentLines,
			langStats.TestFiles,
			langStats.TestLines))
	}

	sb.WriteString("───────────────────────────────────────────────────────────\n\n")

	// Code vs Test breakdown
	totalTestFiles := 0
	totalTestLines := 0
	totalSourceFiles := 0
	totalSourceLines := 0

	for _, langStats := range stats.Languages {
		totalTestFiles += langStats.TestFiles
		totalTestLines += langStats.TestLines
		totalSourceFiles += langStats.Files - langStats.TestFiles
		totalSourceLines += langStats.TotalLines - langStats.TestLines
	}

	sb.WriteString("🧪 CODE VS TESTS\n")
	sb.WriteString("───────────────────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("Source Code:\n"))
	sb.WriteString(fmt.Sprintf("  Files: %d files\n", totalSourceFiles))
	sb.WriteString(fmt.Sprintf("  Lines: %d lines (%.1f%%)\n",
		totalSourceLines, float64(totalSourceLines)/float64(stats.TotalLines)*100))
	sb.WriteString(fmt.Sprintf("Test Code:\n"))
	sb.WriteString(fmt.Sprintf("  Files: %d files\n", totalTestFiles))
	sb.WriteString(fmt.Sprintf("  Lines: %d lines (%.1f%%)\n",
		totalTestLines, float64(totalTestLines)/float64(stats.TotalLines)*100))
	sb.WriteString("\n")

	// Percentage breakdown
	if len(stats.Languages) > 0 {
		mainLang := sortedLangs[0]
		mainLangPercent := float64(mainLang.stats.TotalLines) / float64(stats.TotalLines) * 100
		sb.WriteString("📊 DISTRIBUTION\n")
		sb.WriteString("───────────────────────────────────────────────────────────\n")
		sb.WriteString(fmt.Sprintf("Primary Language: %s (%.1f%% of codebase)\n",
			mainLang.name, mainLangPercent))
		sb.WriteString("\n")
	}

	sb.WriteString("───────────────────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("⏱️  Scan completed in %v\n", duration.Round(time.Millisecond)))
	sb.WriteString("📝 Note: Excluded directories: node_modules, vendor, .git, dist, coverage, etc.\n")

	return sb.String()
}

func generateJSONReport(stats *ProjectStats, duration time.Duration) string {
	var sb strings.Builder
	sb.WriteString("{\n")
	sb.WriteString(fmt.Sprintf(`  "scan_duration_ms": %d,`+"\n", duration.Milliseconds()))
	sb.WriteString(fmt.Sprintf(`  "total_files": %d,`+"\n", stats.TotalFiles))
	sb.WriteString(fmt.Sprintf(`  "total_lines": %d,`+"\n", stats.TotalLines))
	sb.WriteString(fmt.Sprintf(`  "code_lines": %d,`+"\n", stats.CodeLines))
	sb.WriteString(fmt.Sprintf(`  "comment_lines": %d,`+"\n", stats.CommentLines))
	sb.WriteString(fmt.Sprintf(`  "blank_lines": %d,`+"\n", stats.BlankLines))
	sb.WriteString(`  "languages": {` + "\n")

	first := true
	for lang, langStats := range stats.Languages {
		if !first {
			sb.WriteString(",\n")
		}
		first = false
		sb.WriteString(fmt.Sprintf(`    "%s": {`+"\n", lang))
		sb.WriteString(fmt.Sprintf(`      "files": %d,`+"\n", langStats.Files))
		sb.WriteString(fmt.Sprintf(`      "total_lines": %d,`+"\n", langStats.TotalLines))
		sb.WriteString(fmt.Sprintf(`      "code_lines": %d,`+"\n", langStats.CodeLines))
		sb.WriteString(fmt.Sprintf(`      "comment_lines": %d,`+"\n", langStats.CommentLines))
		sb.WriteString(fmt.Sprintf(`      "blank_lines": %d,`+"\n", langStats.BlankLines))
		sb.WriteString(fmt.Sprintf(`      "test_files": %d,`+"\n", langStats.TestFiles))
		sb.WriteString(fmt.Sprintf(`      "test_lines": %d`+"\n", langStats.TestLines))
		sb.WriteString("    }")
	}

	sb.WriteString("\n  }\n}")
	return sb.String()
}
