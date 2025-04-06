// internal/utils/nodejs.go
package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NodeJSFrameworkType representa el tipo de framework de Node.js
type NodeJSFrameworkType string

const (
	// FrameworkUnknown representa un framework desconocido
	FrameworkUnknown NodeJSFrameworkType = "unknown"
	// FrameworkExpress representa Express.js
	FrameworkExpress NodeJSFrameworkType = "express"
	// FrameworkNestJS representa NestJS
	FrameworkNestJS NodeJSFrameworkType = "nestjs"
	// FrameworkNextJS representa Next.js
	FrameworkNextJS NodeJSFrameworkType = "nextjs"
	// FrameworkReactJS representa React.js
	FrameworkReactJS NodeJSFrameworkType = "reactjs"
	// FrameworkVueJS representa Vue.js
	FrameworkVueJS NodeJSFrameworkType = "vuejs"
	// FrameworkNuxtJS representa Nuxt.js
	FrameworkNuxtJS NodeJSFrameworkType = "nuxtjs"
)

// NodeJSProjectInfo contiene información sobre un proyecto Node.js
type NodeJSProjectInfo struct {
	Framework        NodeJSFrameworkType
	MainFile         string
	HasTypeScript    bool
	BuildCommand     string
	StartCommand     string
	DevCommand       string
	HasPrisma        bool
	RequiresEnv      bool
	EnvVars          map[string]string
	DefaultPort      int
	HasPackageJSON   bool
	HasNodeModules   bool
	RequiresDatabase bool
	DBType           string // postgresql, mysql, mongodb, etc.
}

// DetectNodeJSFramework detecta el framework de Node.js utilizado en un proyecto
func DetectNodeJSFramework(projectDir string) (*NodeJSProjectInfo, error) {
	info := &NodeJSProjectInfo{
		Framework:        FrameworkUnknown,
		MainFile:         "",
		HasTypeScript:    false,
		BuildCommand:     "",
		StartCommand:     "",
		DevCommand:       "",
		HasPrisma:        false,
		RequiresEnv:      false,
		EnvVars:          make(map[string]string),
		DefaultPort:      3000,
		HasPackageJSON:   false,
		HasNodeModules:   false,
		RequiresDatabase: false,
		DBType:           "",
	}

	// Verificar si hay package.json
	packageJSONPath := filepath.Join(projectDir, "package.json")
	if _, err := os.Stat(packageJSONPath); err == nil {
		info.HasPackageJSON = true

		// Leer package.json
		packageJSONData, err := os.ReadFile(packageJSONPath)
		if err != nil {
			return nil, fmt.Errorf("error al leer package.json: %v", err)
		}

		var packageJSON map[string]interface{}
		if err := json.Unmarshal(packageJSONData, &packageJSON); err != nil {
			return nil, fmt.Errorf("error al parsear package.json: %v", err)
		}

		// Verificar el main file
		if main, ok := packageJSON["main"].(string); ok && main != "" {
			info.MainFile = main
		}

		// Verificar scripts
		if scripts, ok := packageJSON["scripts"].(map[string]interface{}); ok {
			if build, ok := scripts["build"].(string); ok {
				info.BuildCommand = build
			}
			if start, ok := scripts["start"].(string); ok {
				info.StartCommand = start
			}
			if dev, ok := scripts["dev"].(string); ok {
				info.DevCommand = dev
			}
		}

		// Verificar dependencias para determinar el framework
		if deps, ok := packageJSON["dependencies"].(map[string]interface{}); ok {
			// Detectar NestJS
			if _, ok := deps["@nestjs/core"]; ok {
				info.Framework = FrameworkNestJS
				info.DefaultPort = 3000
			}

			// Detectar Next.js
			if _, ok := deps["next"]; ok {
				info.Framework = FrameworkNextJS
				info.DefaultPort = 3000
			}

			// Detectar Express.js
			if _, ok := deps["express"]; ok && info.Framework == FrameworkUnknown {
				info.Framework = FrameworkExpress
				info.DefaultPort = 3000
			}

			// Detectar React.js
			if _, ok := deps["react"]; ok && info.Framework == FrameworkUnknown {
				info.Framework = FrameworkReactJS
				info.DefaultPort = 3000
			}

			// Detectar Vue.js
			if _, ok := deps["vue"]; ok && info.Framework == FrameworkUnknown {
				info.Framework = FrameworkVueJS
				info.DefaultPort = 8080
			}

			// Detectar Nuxt.js
			if _, ok := deps["nuxt"]; ok {
				info.Framework = FrameworkNuxtJS
				info.DefaultPort = 3000
			}

			// Detectar Prisma
			if _, ok := deps["@prisma/client"]; ok {
				info.HasPrisma = true
				info.RequiresDatabase = true
				info.DBType = "postgresql" // Por defecto, puede ser cambiado después
			}
		}

		// Verificar devDependencies
		if devDeps, ok := packageJSON["devDependencies"].(map[string]interface{}); ok {
			// Verificar TypeScript
			if _, ok := devDeps["typescript"]; ok {
				info.HasTypeScript = true
			}

			// Detectar Prisma en devDependencies
			if _, ok := devDeps["prisma"]; ok {
				info.HasPrisma = true
				info.RequiresDatabase = true
			}
		}
	}

	// Verificar si hay node_modules
	nodeModulesPath := filepath.Join(projectDir, "node_modules")
	if _, err := os.Stat(nodeModulesPath); err == nil {
		info.HasNodeModules = true
	}

	// Verificar archivos tsconfig.json para confirmar TypeScript
	tsconfigPath := filepath.Join(projectDir, "tsconfig.json")
	if _, err := os.Stat(tsconfigPath); err == nil {
		info.HasTypeScript = true
	}

	// Verificar archivos .env.example o .env para determinar variables de entorno
	envExamplePath := filepath.Join(projectDir, ".env.example")
	envPath := filepath.Join(projectDir, ".env")

	var envFileToRead string
	if _, err := os.Stat(envExamplePath); err == nil {
		envFileToRead = envExamplePath
		info.RequiresEnv = true
	} else if _, err := os.Stat(envPath); err == nil {
		envFileToRead = envPath
		info.RequiresEnv = true
	}

	if envFileToRead != "" {
		// Leer archivo .env o .env.example
		envData, err := os.ReadFile(envFileToRead)
		if err == nil {
			envLines := strings.Split(string(envData), "\n")
			for _, line := range envLines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}

				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					// Quitar comillas si existen
					value = strings.Trim(value, "\"'")
					info.EnvVars[key] = value

					// Detectar puerto
					if key == "PORT" {
						if port, err := fmt.Sscanf(value, "%d", &info.DefaultPort); err != nil || port == 0 {
							// Si no se puede parsear, mantener el valor por defecto
						}
					}

					// Detectar variables de base de datos
					if key == "DATABASE_URL" {
						info.RequiresDatabase = true
						// Intentar determinar el tipo de BD
						if strings.Contains(value, "postgresql") {
							info.DBType = "postgresql"
						} else if strings.Contains(value, "mysql") {
							info.DBType = "mysql"
						} else if strings.Contains(value, "mongodb") {
							info.DBType = "mongodb"
						}
					}
				}
			}
		}
	}

	// Si no se ha detectado un framework pero hay archivo main, establecer como Express genérico
	if info.Framework == FrameworkUnknown && info.MainFile != "" {
		info.Framework = FrameworkExpress
	}

	// Si estamos utilizando NestJS, buscar el archivo main.js o main.ts
	if info.Framework == FrameworkNestJS {
		// Verificar estructuras de directorios típicas de NestJS
		possibleMainFiles := []string{
			filepath.Join(projectDir, "src", "main.ts"),
			filepath.Join(projectDir, "src", "main.js"),
			filepath.Join(projectDir, "dist", "main.js"),
		}

		for _, file := range possibleMainFiles {
			if _, err := os.Stat(file); err == nil {
				// Encontrado el archivo principal
				if strings.HasSuffix(file, ".ts") {
					info.MainFile = strings.TrimPrefix(file, projectDir+"/")
				} else if strings.HasSuffix(file, ".js") {
					// Para archivos JS, preferir la versión compilada en dist
					if strings.Contains(file, "/dist/") {
						info.MainFile = strings.TrimPrefix(file, projectDir+"/")
					}
				}
			}
		}
	}

	// Para NextJS, no necesitamos un archivo principal explícito
	if info.Framework == FrameworkNextJS {
		info.MainFile = ""
	}

	return info, nil
}

// ConfigureNodeJSEnv configura el archivo .env para un proyecto Node.js
func ConfigureNodeJSEnv(projectDir string, info *NodeJSProjectInfo, userEnvVars map[string]string) error {
	// Si el proyecto no requiere variables de entorno, no hacer nada
	if !info.RequiresEnv {
		return nil
	}

	// Combinar variables de entorno del proyecto con las proporcionadas por el usuario
	envVars := make(map[string]string)
	for k, v := range info.EnvVars {
		envVars[k] = v
	}
	for k, v := range userEnvVars {
		envVars[k] = v
	}

	// Si es necesario, añadir o actualizar variables estándar
	if info.Framework == FrameworkNestJS || info.Framework == FrameworkExpress {
		// Para APIs, asegurar que tenemos NODE_ENV y PORT
		if _, ok := envVars["NODE_ENV"]; !ok {
			envVars["NODE_ENV"] = "production"
		}
		if _, ok := envVars["PORT"]; !ok {
			envVars["PORT"] = fmt.Sprintf("%d", info.DefaultPort)
		}
	}

	// Construir contenido del archivo .env
	var envContent strings.Builder
	for k, v := range envVars {
		// Añadir comillas si el valor contiene espacios o caracteres especiales
		if strings.ContainsAny(v, " #'\"\n\t") {
			v = fmt.Sprintf("\"%s\"", v)
		}
		envContent.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}

	// Escribir archivo .env
	envPath := filepath.Join(projectDir, ".env")
	if err := os.WriteFile(envPath, []byte(envContent.String()), 0644); err != nil {
		return fmt.Errorf("error al escribir archivo .env: %v", err)
	}

	return nil
}

// GetNodeJSStartCommand devuelve el comando para iniciar una aplicación Node.js
func GetNodeJSStartCommand(info *NodeJSProjectInfo) string {
	// Si hay un comando start explícito, usarlo
	if info.StartCommand != "" {
		return fmt.Sprintf("npm run start")
	}

	// Si no hay comando start, usar el framework para determinar el comando
	switch info.Framework {
	case FrameworkNestJS:
		if info.HasTypeScript {
			return "node dist/main.js"
		}
		return "node src/main.js"
	case FrameworkNextJS:
		return "next start"
	case FrameworkExpress:
		if info.MainFile != "" {
			return fmt.Sprintf("node %s", info.MainFile)
		}
		return "node index.js"
	case FrameworkReactJS:
		return "serve -s build"
	default:
		// Comando genérico
		if info.MainFile != "" {
			return fmt.Sprintf("node %s", info.MainFile)
		}
		return "node index.js"
	}
}

// GetNodeJSBuildCommand devuelve el comando para compilar una aplicación Node.js
func GetNodeJSBuildCommand(info *NodeJSProjectInfo) string {
	// Si hay un comando build explícito, usarlo
	if info.BuildCommand != "" {
		return fmt.Sprintf("npm run build")
	}

	// Si no hay comando build, usar el framework para determinar el comando
	switch info.Framework {
	case FrameworkNestJS:
		if info.HasTypeScript {
			return "npm run build"
		}
		return ""
	case FrameworkNextJS:
		return "next build"
	case FrameworkReactJS:
		return "react-scripts build"
	default:
		// No hay comando build por defecto para otros frameworks
		return ""
	}
}
