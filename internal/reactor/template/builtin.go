package template

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	
	"claude-reactor/pkg"
)

// initializeBuiltinTemplates creates default templates for major languages
func (m *manager) initializeBuiltinTemplates() error {
	m.logger.Infof("Creating built-in templates...")

	templates := []*pkg.ProjectTemplate{
		m.createGoAPITemplate(),
		m.createGoCLITemplate(),
		m.createRustCLITemplate(),
		m.createRustLibTemplate(),
		m.createNodeAPITemplate(),
		m.createReactAppTemplate(),
		m.createPythonAPITemplate(),
		m.createPythonCLITemplate(),
		m.createJavaSpringTemplate(),
	}

	for _, template := range templates {
		if err := m.saveBuiltinTemplate(template); err != nil {
			m.logger.Warnf("Failed to save template %s: %v", template.Name, err)
			continue
		}
		m.logger.Infof("Created built-in template: %s", template.Name)
	}

	return nil
}

// createGoAPITemplate creates a Go REST API template
func (m *manager) createGoAPITemplate() *pkg.ProjectTemplate {
	return &pkg.ProjectTemplate{
		Name:        "go-api",
		Description: "Go REST API with Gorilla Mux and structured logging",
		Language:    "go",
		Framework:   "gorilla/mux",
		Variant:     "go",
		Version:     "1.0.0",
		Author:      "claude-reactor",
		Tags:        []string{"go", "api", "rest", "builtin"},
		DevContainer: true,
		Files: []pkg.TemplateFile{
			{
				Path:     "main.go",
				Template: true,
				Content: `package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()
	
	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/health", healthHandler).Methods("GET")
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("{{.PROJECT_NAME}} server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(` + "`" + `{"message": "Hello from {{.PROJECT_NAME}}!"}` + "`" + `))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(` + "`" + `{"status": "healthy"}` + "`" + `))
}
`,
			},
			{
				Path:     "go.mod",
				Template: true,
				Content: `module {{.PROJECT_NAME_LOWER}}

go 1.23

require (
	github.com/gorilla/mux v1.8.0
)
`,
			},
			{
				Path:     "README.md",
				Template: true,
				Content: `# {{.PROJECT_NAME}}

A Go REST API built with Gorilla Mux.

## Getting Started

1. Install dependencies:
   ` + "```" + `
   go mod download
   ` + "```" + `

2. Run the server:
   ` + "```" + `
   go run main.go
   ` + "```" + `

3. Test the API:
   ` + "```" + `
   curl http://localhost:8080/
   curl http://localhost:8080/health
   ` + "```" + `

## Development

This project is configured for development with Claude Reactor and VS Code Dev Containers.
`,
			},
			{
				Path:     "Dockerfile",
				Template: true,
				Content: `FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o {{.PROJECT_NAME_LOWER}} main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/{{.PROJECT_NAME_LOWER}} .

EXPOSE 8080

CMD ["./{{.PROJECT_NAME_LOWER}}"]
`,
			},
		},
		Variables: []pkg.TemplateVar{
			{
				Name:        "PORT",
				Description: "Server port",
				Type:        "string",
				Default:     "8080",
			},
		},
		GitIgnore: []string{
			"# Go",
			"*.exe",
			"*.exe~",
			"*.dll",
			"*.so",
			"*.dylib",
			"*.test",
			"*.out",
			"go.work",
			"",
			"# IDE",
			".vscode/",
			".idea/",
		},
		PostCreate: []string{
			"go mod download",
		},
	}
}

// createGoCLITemplate creates a Go CLI application template
func (m *manager) createGoCLITemplate() *pkg.ProjectTemplate {
	return &pkg.ProjectTemplate{
		Name:        "go-cli",
		Description: "Go CLI application with Cobra framework",
		Language:    "go",
		Framework:   "cobra",
		Variant:     "go",
		Version:     "1.0.0",
		Author:      "claude-reactor",
		Tags:        []string{"go", "cli", "cobra", "builtin"},
		DevContainer: true,
		Files: []pkg.TemplateFile{
			{
				Path:     "main.go",
				Template: true,
				Content: `package main

import (
	"{{.PROJECT_NAME_LOWER}}/cmd"
)

func main() {
	cmd.Execute()
}
`,
			},
			{
				Path:     "cmd/root.go",
				Template: true,
				Content: `package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "{{.PROJECT_NAME_LOWER}}",
	Short: "{{.PROJECT_NAME}} - A CLI tool",
	Long:  ` + "`" + `{{.PROJECT_NAME}} is a command-line tool built with Go and Cobra.` + "`" + `,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello from {{.PROJECT_NAME}}!")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")
}
`,
			},
			{
				Path:     "go.mod",
				Template: true,
				Content: `module {{.PROJECT_NAME_LOWER}}

go 1.23

require (
	github.com/spf13/cobra v1.8.0
)
`,
			},
			{
				Path:     "README.md",
				Template: true,
				Content: `# {{.PROJECT_NAME}}

A Go CLI application built with Cobra.

## Installation

` + "```" + `
go build -o {{.PROJECT_NAME_LOWER}} main.go
` + "```" + `

## Usage

` + "```" + `
./{{.PROJECT_NAME_LOWER}} --help
` + "```" + `
`,
			},
		},
		GitIgnore: []string{
			"# Go",
			"*.exe",
			"*.exe~",
			"*.dll",
			"*.so",
			"*.dylib",
			"*.test",
			"*.out",
			"go.work",
			"{{.PROJECT_NAME_LOWER}}",
		},
		PostCreate: []string{
			"go mod download",
		},
	}
}

// createRustCLITemplate creates a Rust CLI application template
func (m *manager) createRustCLITemplate() *pkg.ProjectTemplate {
	return &pkg.ProjectTemplate{
		Name:        "rust-cli",
		Description: "Rust CLI application with clap argument parser",
		Language:    "rust",
		Framework:   "clap",
		Variant:     "full",
		Version:     "1.0.0",
		Author:      "claude-reactor",
		Tags:        []string{"rust", "cli", "clap", "builtin"},
		DevContainer: true,
		Files: []pkg.TemplateFile{
			{
				Path:     "Cargo.toml",
				Template: true,
				Content: `[package]
name = "{{.PROJECT_NAME_LOWER}}"
version = "0.1.0"
edition = "2021"

[dependencies]
clap = { version = "4.0", features = ["derive"] }
`,
			},
			{
				Path:     "src/main.rs",
				Template: true,
				Content: `use clap::Parser;

/// {{.PROJECT_NAME}} - A CLI tool written in Rust
#[derive(Parser)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Name of the person to greet
    #[arg(short, long)]
    name: Option<String>,
    
    /// Number of times to greet
    #[arg(short, long, default_value_t = 1)]
    count: u8,
}

fn main() {
    let args = Args::parse();
    
    let name = args.name.unwrap_or_else(|| "World".to_string());
    
    for _ in 0..args.count {
        println!("Hello, {}! Welcome to {{.PROJECT_NAME}}!", name);
    }
}
`,
			},
			{
				Path:     "README.md",
				Template: true,
				Content: `# {{.PROJECT_NAME}}

A Rust CLI application built with clap.

## Installation

` + "```" + `
cargo build --release
` + "```" + `

## Usage

` + "```" + `
cargo run -- --help
cargo run -- --name "Claude" --count 3
` + "```" + `
`,
			},
		},
		GitIgnore: []string{
			"# Rust",
			"/target",
			"**/*.rs.bk",
			"Cargo.lock",
		},
		PostCreate: []string{
			"cargo check",
		},
	}
}

// createRustLibTemplate creates a Rust library template
func (m *manager) createRustLibTemplate() *pkg.ProjectTemplate {
	return &pkg.ProjectTemplate{
		Name:        "rust-lib",
		Description: "Rust library with comprehensive testing setup",
		Language:    "rust",
		Framework:   "",
		Variant:     "full",
		Version:     "1.0.0",
		Author:      "claude-reactor",
		Tags:        []string{"rust", "library", "builtin"},
		DevContainer: true,
		Files: []pkg.TemplateFile{
			{
				Path:     "Cargo.toml",
				Template: true,
				Content: `[package]
name = "{{.PROJECT_NAME_LOWER}}"
version = "0.1.0"
edition = "2021"
description = "{{.PROJECT_NAME}} - A Rust library"
license = "MIT"

[dependencies]

[dev-dependencies]
`,
			},
			{
				Path:     "src/lib.rs",
				Template: true,
				Content: `//! # {{.PROJECT_NAME}}
//!
//! {{.PROJECT_NAME}} is a Rust library that provides...

/// Adds two numbers together.
///
/// # Examples
///
/// ` + "```" + `
/// use {{.PROJECT_NAME_LOWER}}::add;
///
/// assert_eq!(add(2, 3), 5);
/// ` + "```" + `
pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_add() {
        assert_eq!(add(2, 3), 5);
    }
}
`,
			},
			{
				Path:     "README.md",
				Template: true,
				Content: `# {{.PROJECT_NAME}}

A Rust library.

## Usage

Add this to your ` + "`Cargo.toml`" + `:

` + "```toml" + `
[dependencies]
{{.PROJECT_NAME_LOWER}} = "0.1.0"
` + "```" + `

## Development

` + "```" + `
cargo test
cargo doc --open
` + "```" + `
`,
			},
		},
		GitIgnore: []string{
			"# Rust",
			"/target",
			"**/*.rs.bk",
			"Cargo.lock",
		},
		PostCreate: []string{
			"cargo check",
		},
	}
}

// createNodeAPITemplate creates a Node.js REST API template
func (m *manager) createNodeAPITemplate() *pkg.ProjectTemplate {
	return &pkg.ProjectTemplate{
		Name:        "node-api",
		Description: "Node.js REST API with Express and TypeScript",
		Language:    "node",
		Framework:   "express",
		Variant:     "base",
		Version:     "1.0.0",
		Author:      "claude-reactor",
		Tags:        []string{"node", "api", "express", "typescript", "builtin"},
		DevContainer: true,
		Files: []pkg.TemplateFile{
			{
				Path:     "package.json",
				Template: true,
				Content: `{
  "name": "{{.PROJECT_NAME_LOWER}}",
  "version": "1.0.0",
  "description": "{{.PROJECT_NAME}} - A Node.js REST API",
  "main": "dist/index.js",
  "scripts": {
    "build": "tsc",
    "start": "node dist/index.js",
    "dev": "ts-node-dev --respawn --transpile-only src/index.ts",
    "test": "jest"
  },
  "keywords": ["api", "rest", "express", "typescript"],
  "dependencies": {
    "express": "^4.18.0",
    "cors": "^2.8.5"
  },
  "devDependencies": {
    "@types/express": "^4.17.0",
    "@types/cors": "^2.8.0",
    "@types/node": "^20.0.0",
    "typescript": "^5.0.0",
    "ts-node-dev": "^2.0.0"
  }
}
`,
			},
			{
				Path:     "src/index.ts",
				Template: true,
				Content: `import express from 'express';
import cors from 'cors';

const app = express();
const port = process.env.PORT || 3000;

app.use(cors());
app.use(express.json());

app.get('/', (req, res) => {
  res.json({ message: 'Hello from {{.PROJECT_NAME}}!' });
});

app.get('/health', (req, res) => {
  res.json({ status: 'healthy' });
});

app.listen(port, () => {
  console.log(` + "`Server running on port ${port}`" + `);
});
`,
			},
			{
				Path:     "tsconfig.json",
				Template: false,
				Content: `{
  "compilerOptions": {
    "target": "es2020",
    "module": "commonjs",
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
`,
			},
			{
				Path:     "README.md",
				Template: true,
				Content: `# {{.PROJECT_NAME}}

A Node.js REST API built with Express and TypeScript.

## Getting Started

1. Install dependencies:
   ` + "```" + `
   npm install
   ` + "```" + `

2. Start development server:
   ` + "```" + `
   npm run dev
   ` + "```" + `

3. Build for production:
   ` + "```" + `
   npm run build
   npm start
   ` + "```" + `
`,
			},
		},
		GitIgnore: []string{
			"# Node.js",
			"node_modules/",
			"npm-debug.log*",
			"yarn-debug.log*",
			"yarn-error.log*",
			"dist/",
			".env",
		},
		PostCreate: []string{
			"npm install",
		},
	}
}

// Additional template creation functions continue...
func (m *manager) createReactAppTemplate() *pkg.ProjectTemplate {
	return &pkg.ProjectTemplate{
		Name:        "react-app",
		Description: "React application with TypeScript and modern tooling",
		Language:    "node",
		Framework:   "react",
		Variant:     "base",
		Version:     "1.0.0",
		Author:      "claude-reactor",
		Tags:        []string{"react", "typescript", "frontend", "builtin"},
		DevContainer: true,
		Files: []pkg.TemplateFile{
			{
				Path:     "package.json",
				Template: true,
				Content: `{
  "name": "{{.PROJECT_NAME_LOWER}}",
  "version": "0.1.0",
  "private": true,
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "typescript": "^5.0.0",
    "web-vitals": "^3.0.0"
  },
  "scripts": {
    "start": "react-scripts start",
    "build": "react-scripts build",
    "test": "react-scripts test",
    "eject": "react-scripts eject"
  },
  "devDependencies": {
    "@types/react": "^18.0.0",
    "@types/react-dom": "^18.0.0",
    "react-scripts": "5.0.1"
  }
}
`,
			},
		},
		GitIgnore: []string{
			"# React",
			"node_modules/",
			"build/",
			".env.local",
			".env.development.local",
			".env.test.local",
			".env.production.local",
			"npm-debug.log*",
			"yarn-debug.log*",
			"yarn-error.log*",
		},
		PostCreate: []string{
			"npm install",
		},
	}
}

func (m *manager) createPythonAPITemplate() *pkg.ProjectTemplate {
	return &pkg.ProjectTemplate{
		Name:        "python-api",
		Description: "Python REST API with FastAPI and modern tooling",
		Language:    "python",
		Framework:   "fastapi",
		Variant:     "base",
		Version:     "1.0.0",
		Author:      "claude-reactor",
		Tags:        []string{"python", "api", "fastapi", "builtin"},
		DevContainer: true,
		Files: []pkg.TemplateFile{
			{
				Path:     "main.py",
				Template: true,
				Content: `from fastapi import FastAPI
import uvicorn

app = FastAPI(title="{{.PROJECT_NAME}}", version="1.0.0")

@app.get("/")
async def root():
    return {"message": "Hello from {{.PROJECT_NAME}}!"}

@app.get("/health")
async def health():
    return {"status": "healthy"}

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
`,
			},
			{
				Path:     "requirements.txt",
				Template: false,
				Content: `fastapi==0.104.1
uvicorn[standard]==0.24.0
`,
			},
			{
				Path:     "README.md",
				Template: true,
				Content: `# {{.PROJECT_NAME}}

A Python REST API built with FastAPI.

## Getting Started

1. Install dependencies:
   ` + "```" + `
   pip install -r requirements.txt
   ` + "```" + `

2. Start development server:
   ` + "```" + `
   python main.py
   ` + "```" + `

3. Test the API:
   ` + "```" + `
   curl http://localhost:8000/
   curl http://localhost:8000/health
   ` + "```" + `

## Development

This project is configured for development with Claude Reactor and VS Code Dev Containers.
`,
			},
		},
		GitIgnore: []string{
			"# Python",
			"__pycache__/",
			"*.py[cod]",
			"*$py.class",
			"*.so",
			".Python",
			"env/",
			"venv/",
			".venv/",
			".env",
		},
		PostCreate: []string{
			"python -m pip install -r requirements.txt",
		},
	}
}

func (m *manager) createPythonCLITemplate() *pkg.ProjectTemplate {
	return &pkg.ProjectTemplate{
		Name:        "python-cli",
		Description: "Python CLI application with Click framework",
		Language:    "python",
		Framework:   "click",
		Variant:     "base",
		Version:     "1.0.0",
		Author:      "claude-reactor",
		Tags:        []string{"python", "cli", "click", "builtin"},
		DevContainer: true,
		Files: []pkg.TemplateFile{
			{
				Path:     "main.py",
				Template: true,
				Content: `#!/usr/bin/env python3
import click

@click.command()
@click.option('--name', default='World', help='Name to greet')
@click.option('--count', default=1, help='Number of greetings')
def hello(name, count):
    """{{.PROJECT_NAME}} - A Python CLI tool"""
    for _ in range(count):
        click.echo(f'Hello, {name}! Welcome to {{.PROJECT_NAME}}!')

if __name__ == '__main__':
    hello()
`,
				Executable: true,
			},
			{
				Path:     "requirements.txt",
				Template: false,
				Content: `click==8.1.7
`,
			},
		},
		GitIgnore: []string{
			"# Python",
			"__pycache__/",
			"*.py[cod]",
			"*$py.class",
			"*.so",
			".Python",
			"env/",
			"venv/",
			".venv/",
			".env",
		},
		PostCreate: []string{
			"python -m pip install -r requirements.txt",
		},
	}
}

func (m *manager) createJavaSpringTemplate() *pkg.ProjectTemplate {
	return &pkg.ProjectTemplate{
		Name:        "java-spring",
		Description: "Java Spring Boot REST API",
		Language:    "java",
		Framework:   "spring-boot",
		Variant:     "full",
		Version:     "1.0.0",
		Author:      "claude-reactor",
		Tags:        []string{"java", "spring", "api", "builtin"},
		DevContainer: true,
		Files: []pkg.TemplateFile{
			{
				Path:     "pom.xml",
				Template: true,
				Content: `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 
         http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    
    <parent>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-parent</artifactId>
        <version>3.2.0</version>
        <relativePath/>
    </parent>
    
    <groupId>com.example</groupId>
    <artifactId>{{.PROJECT_NAME_LOWER}}</artifactId>
    <version>1.0.0</version>
    <packaging>jar</packaging>
    
    <name>{{.PROJECT_NAME}}</name>
    <description>{{.PROJECT_NAME}} - A Spring Boot application</description>
    
    <properties>
        <java.version>17</java.version>
    </properties>
    
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
        </dependency>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-test</artifactId>
            <scope>test</scope>
        </dependency>
    </dependencies>
    
    <build>
        <plugins>
            <plugin>
                <groupId>org.springframework.boot</groupId>
                <artifactId>spring-boot-maven-plugin</artifactId>
            </plugin>
        </plugins>
    </build>
</project>
`,
			},
		},
		GitIgnore: []string{
			"# Java",
			"*.class",
			"*.log",
			"*.ctxt",
			".mtj.tmp/",
			"*.jar",
			"*.war",
			"*.nar",
			"*.ear",
			"*.zip",
			"*.tar.gz",
			"*.rar",
			"target/",
			".idea/",
			"*.iws",
			"*.iml",
			"*.ipr",
		},
		PostCreate: []string{
			"mvn clean compile",
		},
	}
}

// saveBuiltinTemplate saves a builtin template to the filesystem
func (m *manager) saveBuiltinTemplate(template *pkg.ProjectTemplate) error {
	templateDir := filepath.Join(m.templatesDir, template.Name)
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		return fmt.Errorf("failed to create template directory: %w", err)
	}

	templateFile := filepath.Join(templateDir, "template.yaml")
	data, err := yaml.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	if err := os.WriteFile(templateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to save template: %w", err)
	}

	return nil
}