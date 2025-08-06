// internal/commands/env.go
package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/elmersh/sitemanager/internal/config"
	"github.com/elmersh/sitemanager/internal/utils"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// EnvOptions contiene las opciones para el comando env
type EnvOptions struct {
	Domain      string
	EnvVars     []string
	Interactive bool
	File        string
}

// AddEnvCommand agrega el comando env al comando raíz
func AddEnvCommand(rootCmd *cobra.Command, cfg *config.Config) {
	// Opciones del comando
	var opts EnvOptions

	// Crear comando env
	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Configurar variables de entorno para un sitio",
		Long:  `Configura variables de entorno para un sitio, creando o modificando el archivo .env.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Cargar configuración si no se ha pasado
			if cfg == nil {
				var err error
				cfg, err = config.LoadConfig()
				if err != nil {
					return fmt.Errorf("error al cargar la configuración: %v", err)
				}
			}

			// Configurar opciones
			if opts.Domain == "" {
				return fmt.Errorf("el dominio es obligatorio")
			}

			// Determinar si es un subdominio
			domainParts := strings.Split(opts.Domain, ".")
			var homeDir, user, appDir string
			if len(domainParts) > 2 && domainParts[0] != "www" {
				// Es un subdominio
				parentDomain := strings.Join(domainParts[1:], ".")
				user = strings.Split(parentDomain, ".")[0]
				homeDir = filepath.Join("/home", parentDomain)
				appDir = filepath.Join(homeDir, "apps", domainParts[0])
			} else {
				// No es subdominio
				user = domainParts[0]
				homeDir = filepath.Join("/home", opts.Domain)
				appDir = filepath.Join(homeDir, "app")
			}

			// Verificar si el sitio existe
			if !utils.PathExists(homeDir) {
				return fmt.Errorf("el sitio %s no existe, primero crea el sitio con 'sm site'", opts.Domain)
			}

			// Verificar si la aplicación existe
			if !utils.PathExists(appDir) {
				return fmt.Errorf("la aplicación no existe en %s, primero despliega la aplicación con 'sm deploy'", appDir)
			}

			// Crear o actualizar archivo .env
			if err := configureEnvFile(opts, appDir, user); err != nil {
				return err
			}

			fmt.Printf("Variables de entorno configuradas correctamente para %s\n", opts.Domain)
			return nil
		},
	}

	// Agregar flags
	envCmd.Flags().StringVarP(&opts.Domain, "domain", "d", "", "Dominio del sitio (obligatorio)")
	envCmd.Flags().StringArrayVarP(&opts.EnvVars, "env", "e", []string{}, "Variables de entorno en formato KEY=VALUE")
	envCmd.Flags().BoolVarP(&opts.Interactive, "interactive", "i", false, "Modo interactivo para configurar variables de entorno")
	envCmd.Flags().StringVarP(&opts.File, "file", "f", "", "Archivo .env a importar")

	// Marcar flags obligatorios
	envCmd.MarkFlagRequired("domain")

	// Agregar comando al comando raíz
	rootCmd.AddCommand(envCmd)
}

// configureEnvFile crea o actualiza el archivo .env para una aplicación
func configureEnvFile(opts EnvOptions, appDir, user string) error {
	envFilePath := filepath.Join(appDir, ".env")
	exampleEnvPath := filepath.Join(appDir, ".env.example")

	// Variables de entorno actuales
	envVars := make(map[string]string)

	// Intentar leer archivo .env existente
	if utils.PathExists(envFilePath) {
		data, err := os.ReadFile(envFilePath)
		if err == nil {
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
					// Quitar comillas si existen
					value = strings.Trim(value, "\"'")
					envVars[key] = value
				}
			}
		}
	} else if utils.PathExists(exampleEnvPath) {
		// Si no existe .env pero existe .env.example, leer de ahí
		data, err := os.ReadFile(exampleEnvPath)
		if err == nil {
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
					// Quitar comillas si existen
					value = strings.Trim(value, "\"'")
					envVars[key] = value
				}
			}
		}
	}

	// Si se especifica archivo a importar
	if opts.File != "" {
		data, err := os.ReadFile(opts.File)
		if err != nil {
			return fmt.Errorf("error al leer archivo %s: %v", opts.File, err)
		}

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
				// Quitar comillas si existen
				value = strings.Trim(value, "\"'")
				envVars[key] = value
			}
		}
	}

	// Agregar variables de entorno especificadas en la línea de comandos
	for _, env := range opts.EnvVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			envVars[key] = value
		}
	}

	// Modo interactivo
	if opts.Interactive {
		reader := bufio.NewReader(os.Stdin)

		// Detectar si hay archivo .env.example para usar como base
		var exampleVars []string
		if utils.PathExists(exampleEnvPath) {
			data, err := os.ReadFile(exampleEnvPath)
			if err == nil {
				lines := strings.Split(string(data), "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" || strings.HasPrefix(line, "#") {
						continue
					}

					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						exampleVars = append(exampleVars, key)
					}
				}
			}
		}

		// Si no hay variables en .env.example, usar algunas variables comunes
		if len(exampleVars) == 0 {
			exampleVars = []string{
				"NODE_ENV",
				"PORT",
				"DATABASE_URL",
				"JWT_SECRET",
				"JWT_EXPIRES_IN",
			}
		}

		fmt.Println("Configuración interactiva de variables de entorno")
		fmt.Println("Presiona Enter para mantener el valor actual o déjalo en blanco para omitir")

		// Preguntar por cada variable
		for _, key := range exampleVars {
			currentValue, exists := envVars[key]
			var prompt string
			if exists {
				prompt = fmt.Sprintf("%s [%s]: ", key, currentValue)
			} else {
				prompt = fmt.Sprintf("%s: ", key)
			}

			fmt.Print(prompt)

			// Para contraseñas o secretos, ocultar la entrada
			if strings.Contains(strings.ToLower(key), "password") ||
				strings.Contains(strings.ToLower(key), "secret") ||
				strings.Contains(strings.ToLower(key), "key") {

				fmt.Println("(entrada oculta)")
				bytePassword, err := term.ReadPassword(int(syscall.Stdin))
				fmt.Println() // Nueva línea después de la entrada

				if err != nil {
					// Si hay error, usar entrada normal
					fmt.Print(prompt)
					input, err := reader.ReadString('\n')
					if err != nil {
						return fmt.Errorf("error al leer entrada: %v", err)
					}
					input = strings.TrimSpace(input)
					if input != "" {
						envVars[key] = input
					}
				} else {
					input := string(bytePassword)
					if input != "" {
						envVars[key] = input
					}
				}
			} else {
				input, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("error al leer entrada: %v", err)
				}
				input = strings.TrimSpace(input)
				if input != "" {
					envVars[key] = input
				}
			}
		}

		// Preguntar si desea añadir más variables
		fmt.Print("¿Deseas añadir más variables? (s/n): ")
		answer, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error al leer entrada: %v", err)
		}
		answer = strings.TrimSpace(answer)

		if strings.ToLower(answer) == "s" || strings.ToLower(answer) == "si" || strings.ToLower(answer) == "y" || strings.ToLower(answer) == "yes" {
			for {
				fmt.Print("Nombre de la variable (o Enter para terminar): ")
				key, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("error al leer entrada: %v", err)
				}
				key = strings.TrimSpace(key)
				if key == "" {
					break
				}

				fmt.Printf("%s: ", key)
				value, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("error al leer entrada: %v", err)
				}
				value = strings.TrimSpace(value)
				envVars[key] = value
			}
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
	if err := os.WriteFile(envFilePath, []byte(envContent.String()), 0644); err != nil {
		return fmt.Errorf("error al escribir archivo .env: %v", err)
	}

	// Cambiar propietario del archivo
	cmd := exec.Command("chown", fmt.Sprintf("%s:%s", user, user), envFilePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario del archivo .env: %v\n%s", err, output)
	}

	return nil
}

// detectDatabaseSettings detecta la configuración de base de datos a partir de DATABASE_URL
func detectDatabaseSettings(projectInfo *utils.NodeJSProjectInfo) (*utils.DatabaseOptions, error) {
	// Buscar variable DATABASE_URL
	databaseURL, ok := projectInfo.EnvVars["DATABASE_URL"]
	if !ok {
		return nil, fmt.Errorf("no se encontró variable DATABASE_URL")
	}

	// Analizar URL
	return utils.ParseDatabaseURL(databaseURL)
}

// setupDatabase configura la base de datos para una aplicación
func setupDatabase(projectInfo *utils.NodeJSProjectInfo) error {
	// Si el proyecto no requiere base de datos, no hacer nada
	if !projectInfo.RequiresDatabase {
		return nil
	}

	// Detectar configuración de base de datos
	dbOpts, err := detectDatabaseSettings(projectInfo)
	if err != nil {
		return fmt.Errorf("error al detectar configuración de base de datos: %v", err)
	}

	// Crear base de datos si no existe
	if err := utils.CreateDatabase(dbOpts); err != nil {
		return fmt.Errorf("error al crear base de datos: %v", err)
	}

	return nil
}
