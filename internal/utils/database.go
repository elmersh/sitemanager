// internal/utils/database.go
package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

// DatabaseType representa el tipo de base de datos
type DatabaseType string

const (
	// DBTypePostgreSQL representa PostgreSQL
	DBTypePostgreSQL DatabaseType = "postgresql"
	// DBTypeMySQL representa MySQL
	DBTypeMySQL DatabaseType = "mysql"
	// DBTypeMongoDB representa MongoDB
	DBTypeMongoDB DatabaseType = "mongodb"
	// DBTypeSQLite representa SQLite
	DBTypeSQLite DatabaseType = "sqlite"
)

// DatabaseOptions contiene las opciones para una base de datos
type DatabaseOptions struct {
	Type      DatabaseType
	Host      string
	Port      int
	Name      string
	User      string
	Password  string
	SSLMode   string
	Schema    string
	Timezone  string
	Charset   string
	Collation string
}

// CreateDatabase crea una base de datos si no existe
func CreateDatabase(opts *DatabaseOptions) error {
	switch opts.Type {
	case DBTypePostgreSQL:
		return createPostgreSQLDatabase(opts)
	case DBTypeMySQL:
		return createMySQLDatabase(opts)
	case DBTypeMongoDB:
		// MongoDB crea automáticamente las bases de datos, no es necesario crearlas explícitamente
		return nil
	case DBTypeSQLite:
		// SQLite crea automáticamente el archivo de base de datos
		return nil
	default:
		return fmt.Errorf("tipo de base de datos no soportado: %s", opts.Type)
	}
}

// createPostgreSQLDatabase crea una base de datos PostgreSQL si no existe
func createPostgreSQLDatabase(opts *DatabaseOptions) error {
	// Verificar si la base de datos ya existe
	checkCmd := fmt.Sprintf("psql -U postgres -c \"SELECT 1 FROM pg_database WHERE datname='%s'\" | grep -q 1", opts.Name)
	cmd := exec.Command("bash", "-c", checkCmd)

	if err := cmd.Run(); err == nil {
		// La base de datos ya existe
		fmt.Printf("Base de datos PostgreSQL '%s' ya existe\n", opts.Name)
		return nil
	}

	// Crear base de datos
	createCmd := fmt.Sprintf("psql -U postgres -c \"CREATE DATABASE %s;\"", opts.Name)
	cmd = exec.Command("bash", "-c", createCmd)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al crear base de datos PostgreSQL: %v\n%s", err, output)
	}

	// Crear usuario si no existe
	checkUserCmd := fmt.Sprintf("psql -U postgres -c \"SELECT 1 FROM pg_roles WHERE rolname='%s'\" | grep -q 1", opts.User)
	cmd = exec.Command("bash", "-c", checkUserCmd)

	if err := cmd.Run(); err != nil {
		// El usuario no existe, crearlo
		createUserCmd := fmt.Sprintf("psql -U postgres -c \"CREATE USER %s WITH ENCRYPTED PASSWORD '%s';\"", opts.User, opts.Password)
		cmd = exec.Command("bash", "-c", createUserCmd)

		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al crear usuario PostgreSQL: %v\n%s", err, output)
		}
	}

	// Asignar privilegios
	grantCmd := fmt.Sprintf("psql -U postgres -c \"GRANT ALL PRIVILEGES ON DATABASE %s TO %s;\"", opts.Name, opts.User)
	cmd = exec.Command("bash", "-c", grantCmd)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al asignar privilegios: %v\n%s", err, output)
	}

	// Crear schema si se especifica
	if opts.Schema != "" && opts.Schema != "public" {
		schemaCmd := fmt.Sprintf("psql -U postgres -d %s -c \"CREATE SCHEMA IF NOT EXISTS %s AUTHORIZATION %s;\"", opts.Name, opts.Schema, opts.User)
		cmd = exec.Command("bash", "-c", schemaCmd)

		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al crear schema: %v\n%s", err, output)
		}
	}

	fmt.Printf("Base de datos PostgreSQL '%s' creada correctamente\n", opts.Name)
	return nil
}

// createMySQLDatabase crea una base de datos MySQL si no existe
func createMySQLDatabase(opts *DatabaseOptions) error {
	// Verificar si la base de datos ya existe
	checkCmd := fmt.Sprintf("mysql -u root -e \"SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME='%s'\" | grep -q %s", opts.Name, opts.Name)
	cmd := exec.Command("bash", "-c", checkCmd)

	if err := cmd.Run(); err == nil {
		// La base de datos ya existe
		fmt.Printf("Base de datos MySQL '%s' ya existe\n", opts.Name)
		return nil
	}

	// Crear base de datos
	charset := "utf8mb4"
	collation := "utf8mb4_unicode_ci"

	if opts.Charset != "" {
		charset = opts.Charset
	}

	if opts.Collation != "" {
		collation = opts.Collation
	}

	createCmd := fmt.Sprintf("mysql -u root -e \"CREATE DATABASE %s CHARACTER SET %s COLLATE %s;\"", opts.Name, charset, collation)
	cmd = exec.Command("bash", "-c", createCmd)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al crear base de datos MySQL: %v\n%s", err, output)
	}

	// Crear usuario si no existe
	checkUserCmd := fmt.Sprintf("mysql -u root -e \"SELECT User FROM mysql.user WHERE User='%s'\" | grep -q %s", opts.User, opts.User)
	cmd = exec.Command("bash", "-c", checkUserCmd)

	if err := cmd.Run(); err != nil {
		// El usuario no existe, crearlo
		createUserCmd := fmt.Sprintf("mysql -u root -e \"CREATE USER '%s'@'localhost' IDENTIFIED BY '%s';\"", opts.User, opts.Password)
		cmd = exec.Command("bash", "-c", createUserCmd)

		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al crear usuario MySQL: %v\n%s", err, output)
		}
	}

	// Asignar privilegios
	grantCmd := fmt.Sprintf("mysql -u root -e \"GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost'; FLUSH PRIVILEGES;\"", opts.Name, opts.User)
	cmd = exec.Command("bash", "-c", grantCmd)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al asignar privilegios: %v\n%s", err, output)
	}

	fmt.Printf("Base de datos MySQL '%s' creada correctamente\n", opts.Name)
	return nil
}

// ParseDatabaseURL analiza una URL de conexión a base de datos y devuelve las opciones
func ParseDatabaseURL(url string) (*DatabaseOptions, error) {
	opts := &DatabaseOptions{
		Host:     "localhost",
		SSLMode:  "prefer",
		Schema:   "public",
		Timezone: "UTC",
		Charset:  "utf8mb4",
	}

	// Determinar el tipo de base de datos
	if strings.HasPrefix(url, "postgresql://") {
		opts.Type = DBTypePostgreSQL
		opts.Port = 5432
	} else if strings.HasPrefix(url, "mysql://") {
		opts.Type = DBTypeMySQL
		opts.Port = 3306
	} else if strings.HasPrefix(url, "mongodb://") {
		opts.Type = DBTypeMongoDB
		opts.Port = 27017
	} else if strings.HasPrefix(url, "sqlite://") {
		opts.Type = DBTypeSQLite
		// SQLite no tiene puerto
		return opts, nil
	} else {
		return nil, fmt.Errorf("URL de base de datos no reconocida: %s", url)
	}

	// Para PostgreSQL
	if opts.Type == DBTypePostgreSQL {
		// Formato: postgresql://username:password@localhost:5432/database?schema=public
		// Extraer usuario y contraseña
		if idx := strings.Index(url, "://"); idx >= 0 {
			url = url[idx+3:]

			// Extraer usuario y contraseña
			if idx = strings.Index(url, "@"); idx >= 0 {
				userPass := url[:idx]
				url = url[idx+1:]

				if idx = strings.Index(userPass, ":"); idx >= 0 {
					opts.User = userPass[:idx]
					opts.Password = userPass[idx+1:]
				} else {
					opts.User = userPass
				}
			}

			// Extraer host y puerto
			if idx = strings.Index(url, "/"); idx >= 0 {
				hostPort := url[:idx]
				url = url[idx+1:]

				if idx = strings.Index(hostPort, ":"); idx >= 0 {
					opts.Host = hostPort[:idx]
					fmt.Sscanf(hostPort[idx+1:], "%d", &opts.Port)
				} else {
					opts.Host = hostPort
				}
			}

			// Extraer nombre de base de datos y parámetros
			if idx = strings.Index(url, "?"); idx >= 0 {
				opts.Name = url[:idx]
				params := url[idx+1:]

				// Extraer parámetros
				for _, param := range strings.Split(params, "&") {
					if idx = strings.Index(param, "="); idx >= 0 {
						key := param[:idx]
						value := param[idx+1:]

						switch key {
						case "schema":
							opts.Schema = value
						case "sslmode":
							opts.SSLMode = value
						}
					}
				}
			} else {
				opts.Name = url
			}
		}
	}

	// Para MySQL
	if opts.Type == DBTypeMySQL {
		// Formato: mysql://username:password@localhost:3306/database
		// Extraer usuario y contraseña
		if idx := strings.Index(url, "://"); idx >= 0 {
			url = url[idx+3:]

			// Extraer usuario y contraseña
			if idx = strings.Index(url, "@"); idx >= 0 {
				userPass := url[:idx]
				url = url[idx+1:]

				if idx = strings.Index(userPass, ":"); idx >= 0 {
					opts.User = userPass[:idx]
					opts.Password = userPass[idx+1:]
				} else {
					opts.User = userPass
				}
			}

			// Extraer host y puerto
			if idx = strings.Index(url, "/"); idx >= 0 {
				hostPort := url[:idx]
				url = url[idx+1:]

				if idx = strings.Index(hostPort, ":"); idx >= 0 {
					opts.Host = hostPort[:idx]
					fmt.Sscanf(hostPort[idx+1:], "%d", &opts.Port)
				} else {
					opts.Host = hostPort
				}
			}

			// Extraer nombre de base de datos
			if idx = strings.Index(url, "?"); idx >= 0 {
				opts.Name = url[:idx]
				// Ignoramos los parámetros por ahora
			} else {
				opts.Name = url
			}
		}
	}

	return opts, nil
}

// BuildDatabaseURL construye una URL de conexión a base de datos a partir de las opciones
func BuildDatabaseURL(opts *DatabaseOptions) string {
	switch opts.Type {
	case DBTypePostgreSQL:
		url := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s", opts.User, opts.Password, opts.Host, opts.Port, opts.Name)
		if opts.Schema != "" && opts.Schema != "public" {
			url += fmt.Sprintf("?schema=%s", opts.Schema)
		}
		if opts.SSLMode != "" && opts.SSLMode != "prefer" {
			if strings.Contains(url, "?") {
				url += fmt.Sprintf("&sslmode=%s", opts.SSLMode)
			} else {
				url += fmt.Sprintf("?sslmode=%s", opts.SSLMode)
			}
		}
		return url
	case DBTypeMySQL:
		return fmt.Sprintf("mysql://%s:%s@%s:%d/%s", opts.User, opts.Password, opts.Host, opts.Port, opts.Name)
	case DBTypeMongoDB:
		return fmt.Sprintf("mongodb://%s:%s@%s:%d/%s", opts.User, opts.Password, opts.Host, opts.Port, opts.Name)
	case DBTypeSQLite:
		return fmt.Sprintf("sqlite:///%s", opts.Name)
	default:
		return ""
	}
}
