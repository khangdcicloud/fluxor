package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const version = "0.1.0"

func main() {
	newCmd := flag.NewFlagSet("new", flag.ExitOnError)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "new":
		newCmd.Parse(os.Args[2:])
		if newCmd.NArg() == 0 {
			fmt.Fprintf(os.Stderr, "Error: app name is required\n\n")
			fmt.Fprintf(os.Stderr, "Usage: fluxorcli new <appname>\n")
			os.Exit(1)
		}
		appName := newCmd.Arg(0)
		if err := createNewApp(appName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Successfully created Fluxor application '%s'\n", appName)
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  cd %s\n", appName)
		fmt.Printf("  go mod tidy\n")
		fmt.Printf("  go run .\n\n")
	case "version", "-v", "--version":
		fmt.Printf("fluxorcli version %s\n", version)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Fluxor CLI - Create and manage Fluxor applications\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  fluxorcli new <appname>    Create a new Fluxor application\n")
	fmt.Fprintf(os.Stderr, "  fluxorcli version          Show version\n\n")
}

func createNewApp(appName string) error {
	// Validate app name
	if appName == "" {
		return fmt.Errorf("app name cannot be empty")
	}

	// Check if directory already exists
	if _, err := os.Stat(appName); err == nil {
		return fmt.Errorf("directory '%s' already exists", appName)
	}

	// Create directory
	if err := os.MkdirAll(appName, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create main.go
	if err := createMainGo(appName); err != nil {
		return fmt.Errorf("failed to create main.go: %w", err)
	}

	// Create config.json
	if err := createConfigJson(appName); err != nil {
		return fmt.Errorf("failed to create config.json: %w", err)
	}

	// Create go.mod
	if err := createGoMod(appName); err != nil {
		return fmt.Errorf("failed to create go.mod: %w", err)
	}

	// Create README.md
	if err := createReadme(appName); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}

	return nil
}

func createMainGo(appName string) error {
	tmpl := `package main

import (
	"log"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/fluxor"
	"github.com/fluxorio/fluxor/pkg/web"
)

func main() {
	// Create MainVerticle with config
	app, err := fluxor.NewMainVerticle("config.json")
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Deploy API verticle
	_, err = app.DeployVerticle(NewApiVerticle())
	if err != nil {
		log.Fatalf("Failed to deploy API verticle: %v", err)
	}

	// Start application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}

// ApiVerticle handles HTTP endpoints
type ApiVerticle struct {
	server *web.FastHTTPServer
}

func NewApiVerticle() *ApiVerticle {
	return &ApiVerticle{}
}

func (v *ApiVerticle) Start(ctx core.FluxorContext) error {
	log.Println("API Verticle started")

	// Get HTTP address from config
	addr := ":8080"
	if val, ok := ctx.Config()["http_addr"].(string); ok && val != "" {
		addr = val
	}

	// Create FastHTTPServer using context's GoCMD
	cfg := web.DefaultFastHTTPServerConfig(addr)
	v.server = web.NewFastHTTPServer(ctx.GoCMD(), cfg)

	// Setup routes
	router := v.server.FastRouter()

	// Health check endpoint
	router.GETFast("/health", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]interface{}{
			"status":  "UP",
			"service": "{{.AppName}}",
		})
	})

	// Example API endpoint
	router.GETFast("/api/hello", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]interface{}{
			"message": "Hello from Fluxor!",
		})
	})

	// Start server
	go func() {
		log.Printf("HTTP server starting on %s", addr)
		if err := v.server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	log.Println("API endpoints registered:")
	log.Println("  GET /health - Health check")
	log.Println("  GET /api/hello - Hello endpoint")

	return nil
}

func (v *ApiVerticle) Stop(ctx core.FluxorContext) error {
	log.Println("API Verticle stopped")
	if v.server != nil {
		return v.server.Stop()
	}
	return nil
}
`

	t, err := template.New("main.go").Parse(tmpl)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(appName, "main.go"))
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, map[string]string{
		"AppName": appName,
	})
}

func createConfigJson(appName string) error {
	content := `{
  "http_addr": ":8080"
}
`
	return os.WriteFile(filepath.Join(appName, "config.json"), []byte(content), 0644)
}

func createGoMod(appName string) error {
	content := `module ` + appName + `

go 1.24.0

require github.com/fluxorio/fluxor v0.0.0
`
	return os.WriteFile(filepath.Join(appName, "go.mod"), []byte(content), 0644)
}

func createReadme(appName string) error {
	content := `# ` + appName + ` - Fluxor Application

A simple Fluxor application starter template.

## Getting Started

1. **Install dependencies:**
   ` + "```bash" + `
   go mod tidy
   ` + "```" + `

2. **Run the application:**
   ` + "```bash" + `
   go run .
   ` + "```" + `

3. **Test the endpoints:**
   ` + "```bash" + `
   curl http://localhost:8080/health
   curl http://localhost:8080/api/hello
   ` + "```" + `

## Project Structure

` + "```" + `
` + appName + `/
├── main.go          # Application entry point
├── config.json      # Configuration file
├── go.mod          # Go module file
└── README.md       # This file
` + "```" + `

## Next Steps

- Add more verticles in separate files (see ` + "`verticles/`" + ` directory pattern)
- Create event contracts in ` + "`contracts/`" + ` directory
- Add database connections using ` + "`pkg/db`" + `
- Add middleware for authentication, logging, etc.

## Resources

- [Fluxor Documentation](https://github.com/fluxorio/fluxor/blob/main/DOCUMENTATION.md)
- [Primary Pattern Guide](https://github.com/fluxorio/fluxor/blob/main/docs/PRIMARY_PATTERN.md)
- [Examples](https://github.com/fluxorio/fluxor/tree/main/examples)
`
	return os.WriteFile(filepath.Join(appName, "README.md"), []byte(content), 0644)
}
