// internal/utils/nodejs.go
package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
// Mejora de la función DetectNodeJSFramework en internal/utils/nodejs.go

// DetectNodeJSFramework detecta el framework y características de un proyecto Node.js
func DetectNodeJSFramework(appDir string) (*NodeJSProjectInfo, error) {
	// Información del proyecto
	info := &NodeJSProjectInfo{
		Framework:        FrameworkUnknown,
		DefaultPort:      3000,
		RequiresEnv:      false,
		RequiresDatabase: false,
		DBType:           "",
		EnvVars:          make(map[string]string),
		HasTypeScript:    false,
		HasPrisma:        false,
	}

	// Verificar si hay package.json
	packagePath := filepath.Join(appDir, "package.json")
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		return info, fmt.Errorf("no se encontró package.json")
	}

	// Leer package.json
	packageData, err := os.ReadFile(packagePath)
	if err != nil {
		return info, fmt.Errorf("error al leer package.json: %v", err)
	}

	// Parsear package.json
	var packageJSON map[string]interface{}
	if err := json.Unmarshal(packageData, &packageJSON); err != nil {
		return info, fmt.Errorf("error al parsear package.json: %v", err)
	}

	// Detectar si usa TypeScript
	if deps, ok := packageJSON["dependencies"].(map[string]interface{}); ok {
		if _, exists := deps["typescript"]; exists {
			info.HasTypeScript = true
		}
	}

	if devDeps, ok := packageJSON["devDependencies"].(map[string]interface{}); ok {
		if _, exists := devDeps["typescript"]; exists {
			info.HasTypeScript = true
		}
	}

	// Verificar si hay tsconfig.json (otra forma de detectar TypeScript)
	if _, err := os.Stat(filepath.Join(appDir, "tsconfig.json")); err == nil {
		info.HasTypeScript = true
	}

	// Detectar NestJS
	if deps, ok := packageJSON["dependencies"].(map[string]interface{}); ok {
		if _, exists := deps["@nestjs/core"]; exists {
			info.Framework = FrameworkNestJS
			info.DefaultPort = 3000
			info.RequiresEnv = true
		}
	}

	// Detectar NextJS
	if deps, ok := packageJSON["dependencies"].(map[string]interface{}); ok {
		if _, exists := deps["next"]; exists {
			info.Framework = FrameworkNextJS
			info.DefaultPort = 3000
			info.RequiresEnv = true
		}
	}

	// Detectar Express
	if deps, ok := packageJSON["dependencies"].(map[string]interface{}); ok {
		if _, exists := deps["express"]; exists && info.Framework == FrameworkUnknown {
			info.Framework = FrameworkExpress
			info.DefaultPort = 3000
			info.RequiresEnv = true
		}
	}

	// Detectar Prisma
	if deps, ok := packageJSON["dependencies"].(map[string]interface{}); ok {
		if _, exists := deps["@prisma/client"]; exists {
			info.HasPrisma = true
			info.RequiresDatabase = true

			// Detectar tipo de base de datos para Prisma
			prismaSchemaPath := filepath.Join(appDir, "prisma", "schema.prisma")
			if _, err := os.Stat(prismaSchemaPath); err == nil {
				// Leer schema.prisma
				schemaData, err := os.ReadFile(prismaSchemaPath)
				if err == nil {
					schemaContent := string(schemaData)

					// Detectar PostgreSQL
					if strings.Contains(schemaContent, "provider = \"postgresql\"") {
						info.DBType = "postgresql"
					} else if strings.Contains(schemaContent, "provider = \"mysql\"") {
						info.DBType = "mysql"
					} else if strings.Contains(schemaContent, "provider = \"sqlite\"") {
						info.DBType = "sqlite"
					} else if strings.Contains(schemaContent, "provider = \"mongodb\"") {
						info.DBType = "mongodb"
					}
				}
			}
		}
	}

	if devDeps, ok := packageJSON["devDependencies"].(map[string]interface{}); ok {
		if _, exists := devDeps["prisma"]; exists {
			info.HasPrisma = true
			info.RequiresDatabase = true
		}
	}

	// Verificar más detalladamente el tipo de base de datos
	// Si se detectó Prisma pero no se pudo determinar el tipo de base de datos
	if info.HasPrisma && info.DBType == "" {
		// Buscar en archivo .env.example o .env si existe
		envExamplePath := filepath.Join(appDir, ".env.example")
		envPath := filepath.Join(appDir, ".env")

		var envContent string

		// Intentar leer .env.example primero
		if _, err := os.Stat(envExamplePath); err == nil {
			if data, err := os.ReadFile(envExamplePath); err == nil {
				envContent = string(data)
			}
		}

		// Si no encontramos .env.example, intentar con .env
		if envContent == "" {
			if _, err := os.Stat(envPath); err == nil {
				if data, err := os.ReadFile(envPath); err == nil {
					envContent = string(data)
				}
			}
		}

		// Buscar DATABASE_URL en el contenido
		if envContent != "" {
			lines := strings.Split(envContent, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "DATABASE_URL=") || strings.HasPrefix(line, "DATABASE_URL =") {
					value := strings.SplitN(line, "=", 2)[1]
					value = strings.TrimSpace(value)

					// Eliminar comillas si las hay
					if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
						(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
						value = value[1 : len(value)-1]
					}

					// Detectar tipo de base de datos por la URL
					if strings.HasPrefix(value, "postgresql://") {
						info.DBType = "postgresql"
					} else if strings.HasPrefix(value, "mysql://") {
						info.DBType = "mysql"
					} else if strings.HasPrefix(value, "file:") || strings.HasPrefix(value, "sqlite:") {
						info.DBType = "sqlite"
					} else if strings.HasPrefix(value, "mongodb://") {
						info.DBType = "mongodb"
					}

					break
				}
			}
		}
	}

	// Detectar base de datos por otras dependencias si aún no se ha detectado
	if !info.RequiresDatabase {
		if deps, ok := packageJSON["dependencies"].(map[string]interface{}); ok {
			// PostgreSQL
			if _, exists := deps["pg"]; exists {
				info.RequiresDatabase = true
				info.DBType = "postgresql"
			} else if _, exists := deps["postgres"]; exists {
				info.RequiresDatabase = true
				info.DBType = "postgresql"
			} else if _, exists := deps["postgresql"]; exists {
				info.RequiresDatabase = true
				info.DBType = "postgresql"
			} else if _, exists := deps["mysql"]; exists {
				info.RequiresDatabase = true
				info.DBType = "mysql"
			} else if _, exists := deps["mysql2"]; exists {
				info.RequiresDatabase = true
				info.DBType = "mysql"
			} else if _, exists := deps["sqlite"]; exists {
				info.RequiresDatabase = true
				info.DBType = "sqlite"
			} else if _, exists := deps["sqlite3"]; exists {
				info.RequiresDatabase = true
				info.DBType = "sqlite"
			} else if _, exists := deps["mongodb"]; exists {
				info.RequiresDatabase = true
				info.DBType = "mongodb"
			} else if _, exists := deps["mongoose"]; exists {
				info.RequiresDatabase = true
				info.DBType = "mongodb"
			}
		}
	}

	// Verificar si hay archivos .env o .env.example
	if _, err := os.Stat(filepath.Join(appDir, ".env")); err == nil {
		info.RequiresEnv = true
	}

	if _, err := os.Stat(filepath.Join(appDir, ".env.example")); err == nil {
		info.RequiresEnv = true
	}

	// Leer variables de .env.example si existe
	envExamplePath := filepath.Join(appDir, ".env.example")
	if _, err := os.Stat(envExamplePath); err == nil {
		// Leer el archivo .env.example
		data, err := os.ReadFile(envExamplePath)
		if err == nil {
			// Parsear variables
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}

				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])

					// Eliminar comillas si las hay
					if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
						(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
						value = value[1 : len(value)-1]
					}

					info.EnvVars[key] = value
				}
			}
		}
	}

	return info, nil
}

// ConfigureNodeJSEnv configura el archivo .env para una aplicación Node.js
func ConfigureNodeJSEnv(appDir string, projectInfo *NodeJSProjectInfo, userValues map[string]string) error {
	// Ruta del archivo .env.example
	examplePath := filepath.Join(appDir, ".env.example")

	// Ruta del archivo .env
	envPath := filepath.Join(appDir, ".env")

	// Variables a configurar
	envVars := make(map[string]string)

	// Si ya existe un archivo .env, leerlo primero
	if _, err := os.Stat(envPath); err == nil {
		// El archivo .env ya existe, leerlo
		data, err := os.ReadFile(envPath)
		if err != nil {
			return fmt.Errorf("error al leer archivo .env existente: %v", err)
		}

		// Parsear variables existentes
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// Eliminar comillas alrededor del valor si existen
				if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
					(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
					value = value[1 : len(value)-1]
				}

				envVars[key] = value
			}
		}
	}

	// Verificar si hay un archivo .env.example
	var exampleVars map[string]string
	if _, err := os.Stat(examplePath); err == nil {
		// El archivo .env.example existe, leerlo
		data, err := os.ReadFile(examplePath)
		if err != nil {
			return fmt.Errorf("error al leer archivo .env.example: %v", err)
		}

		// Parsear variables de ejemplo
		exampleVars = make(map[string]string)
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// Eliminar comillas alrededor del valor si existen
				if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
					(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
					value = value[1 : len(value)-1]
				}

				// Solo agregar a exampleVars si la clave no está ya en envVars
				if _, exists := envVars[key]; !exists {
					exampleVars[key] = value
				}
			}
		}

		// Llenar projectInfo.EnvVars con las variables de ejemplo
		for key, value := range exampleVars {
			projectInfo.EnvVars[key] = value
		}
	}

	// Agregar valores proporcionados por el usuario
	for key, value := range userValues {
		envVars[key] = value
	}

	// Ahora tenemos todas las variables, incluidas las existentes y las nuevas
	// Generar contenido del archivo .env
	var content strings.Builder

	// Agregar encabezado
	content.WriteString("# Archivo generado por SiteManager\n\n")

	// Priorizar las variables relacionadas con la base de datos
	dbKeys := []string{
		"DATABASE_URL",
		"DB_CONNECTION",
		"DB_HOST",
		"DB_PORT",
		"DB_DATABASE",
		"DB_USERNAME",
		"DB_PASSWORD",
	}

	// Agregar primero las variables de base de datos
	content.WriteString("# Configuración de base de datos\n")
	for _, key := range dbKeys {
		if value, exists := envVars[key]; exists {
			// Verificar si el valor necesita comillas
			if strings.Contains(value, " ") || strings.Contains(value, "#") {
				content.WriteString(fmt.Sprintf("%s=\"%s\"\n", key, value))
			} else {
				content.WriteString(fmt.Sprintf("%s=%s\n", key, value))
			}
			// Eliminar la clave del mapa para no repetirla después
			delete(envVars, key)
		}
	}
	content.WriteString("\n")

	// Agregar variables de NextJS si existen
	nextjsKeys := []string{
		"NEXT_PUBLIC_API_URL",
		"NEXT_PUBLIC_IMAGE_DOMAINS",
		"NEXT_PUBLIC_IMAGES_URL",
		"NEXT_PUBLIC_PWA_ENABLED",
		"NEXT_PUBLIC_BODY_SIZE_LIMIT",
	}

	hasNextJSVars := false
	for _, key := range nextjsKeys {
		if _, exists := envVars[key]; exists {
			if !hasNextJSVars {
				content.WriteString("# Configuración de NextJS\n")
				hasNextJSVars = true
			}
			value := envVars[key]
			if strings.Contains(value, " ") || strings.Contains(value, "#") {
				content.WriteString(fmt.Sprintf("%s=\"%s\"\n", key, value))
			} else {
				content.WriteString(fmt.Sprintf("%s=%s\n", key, value))
			}
			delete(envVars, key)
		}
	}
	if hasNextJSVars {
		content.WriteString("\n")
	}

	// Luego agregar el resto de variables
	content.WriteString("# Otras configuraciones\n")

	// Ordenar claves para una salida predecible
	keys := make([]string, 0, len(envVars))
	for key := range envVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := envVars[key]
		// Verificar si el valor necesita comillas
		if strings.Contains(value, " ") || strings.Contains(value, "#") {
			content.WriteString(fmt.Sprintf("%s=\"%s\"\n", key, value))
		} else {
			content.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		}
	}

	// Escribir el archivo .env
	if err := os.WriteFile(envPath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("error al escribir archivo .env: %v", err)
	}

	return nil
}

// GetNodeJSStartCommand devuelve el comando para iniciar una aplicación Node.js
func GetNodeJSStartCommand(info *NodeJSProjectInfo) string {
	// Si hay un comando start explícito, usarlo
	if info.StartCommand != "" {
		return "npm run start"
	}

	// Si no hay comando start, usar el framework para determinar el comando
	switch info.Framework {
	case FrameworkNestJS:
		if info.HasTypeScript {
			return "node dist/src/main.js"
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
		return "npm run build"
	case FrameworkReactJS:
		return "react-scripts build"
	default:
		// No hay comando build por defecto para otros frameworks
		return ""
	}
}
