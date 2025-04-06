package commands

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/elmersh/sitemanager/internal/config"
	"github.com/elmersh/sitemanager/internal/utils"
	"github.com/spf13/cobra"
)

// DeployOptions contiene las opciones para el comando deploy
type DeployOptions struct {
	Domain       string
	Repository   string
	Branch       string
	Type         string
	Environment  string
	User         string
	HomeDir      string
	AppDir       string
	UseSSH       bool
	SSHKeyPath   string
	IsSubdomain  bool
	ParentDomain string
	RepoOwner    string
	RepoName     string
}

// AddDeployCommand agrega el comando deploy al comando raíz
func AddDeployCommand(rootCmd *cobra.Command, cfg *config.Config) {
	// Inicializar semilla para números aleatorios
	// rand.Seed(time.Now().UnixNano()) // Deprecated in Go 1.20+

	// Opciones del comando
	var opts DeployOptions
	var useSSH bool

	// Crear comando deploy
	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Desplegar una aplicación web",
		Long:  `Despliega una aplicación web desde un repositorio Git y configura el entorno necesario.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Configurar opciones
			if opts.Domain == "" {
				return fmt.Errorf("el dominio es obligatorio")
			}

			if opts.Repository == "" {
				return fmt.Errorf("el repositorio Git es obligatorio")
			}

			// Usar valores por defecto si no se especifican
			if opts.Branch == "" {
				opts.Branch = "main"
			}

			if opts.Type == "" {
				opts.Type = cfg.DefaultTemplate
			}

			if opts.Environment == "" {
				opts.Environment = "production"
			}

			// Determinar si es un subdominio
			domainParts := strings.Split(opts.Domain, ".")
			if len(domainParts) > 2 && domainParts[0] != "www" {
				opts.IsSubdomain = true
				opts.ParentDomain = strings.Join(domainParts[1:], ".")
				fmt.Printf("Detectado subdominio de %s\n", opts.ParentDomain)

				// Usar el usuario del dominio principal para subdominios
				opts.User = strings.Split(opts.ParentDomain, ".")[0]
				opts.HomeDir = filepath.Join("/home", opts.ParentDomain)
				// Usar el dominio completo como directorio principal
				opts.AppDir = filepath.Join(opts.HomeDir, "apps", opts.Domain)
			} else {
				// No es subdominio, configuración normal
				opts.User = domainParts[0]
				opts.HomeDir = filepath.Join("/home", opts.Domain)
				opts.AppDir = filepath.Join(opts.HomeDir, "apps", opts.Domain)
			}

			// Configurar opciones SSH
			opts.UseSSH = useSSH

			// Procesar la URL del repositorio para obtener propietario y nombre
			if useSSH {
				// Formato: git@github.com:propietario/repo.git
				if strings.HasPrefix(opts.Repository, "git@github.com:") {
					repoPath := strings.TrimPrefix(opts.Repository, "git@github.com:")
					repoPath = strings.TrimSuffix(repoPath, ".git")
					repoParts := strings.Split(repoPath, "/")
					if len(repoParts) == 2 {
						opts.RepoOwner = repoParts[0]
						opts.RepoName = repoParts[1]
					}
				} else {
					return fmt.Errorf("formato de URL SSH no válido, debe ser git@github.com:propietario/repo.git")
				}
			} else {
				// Formato: https://github.com/propietario/repo.git
				if strings.HasPrefix(opts.Repository, "https://github.com/") {
					repoPath := strings.TrimPrefix(opts.Repository, "https://github.com/")
					repoPath = strings.TrimSuffix(repoPath, ".git")
					repoParts := strings.Split(repoPath, "/")
					if len(repoParts) == 2 {
						opts.RepoOwner = repoParts[0]
						opts.RepoName = repoParts[1]
					}
				}
			}

			// Verificar si el directorio de la aplicación existe
			if _, err := os.Stat(opts.AppDir); os.IsNotExist(err) {
				return fmt.Errorf("el directorio de la aplicación no existe: %s", opts.AppDir)
			}

			// Generar nombre para la clave SSH
			if opts.UseSSH {
				// Sanitizar nombres para usarlos en el nombre de archivo
				domainSafe := strings.ReplaceAll(opts.Domain, ".", "_")
				ownerSafe := strings.ReplaceAll(opts.RepoOwner, "-", "_")
				repoSafe := strings.ReplaceAll(opts.RepoName, "-", "_")

				keyName := fmt.Sprintf("%s_%s_%s", domainSafe, ownerSafe, repoSafe)
				opts.SSHKeyPath = filepath.Join(opts.HomeDir, ".ssh", keyName)

				// Verificar si el sitio existe
				if !siteExists(opts.Domain, cfg) {
					return fmt.Errorf("el sitio %s no existe, primero crea el sitio con 'sm site'", opts.Domain)
				}

				// Clonar repositorio
				if err := cloneRepository(&opts); err != nil {
					return err
				}

				// Configurar entorno según el tipo de aplicación
				switch opts.Type {
				case "laravel":
					if err := deployLaravel(&opts); err != nil {
						return err
					}
				case "nodejs":
					if err := deployNodejs(&opts); err != nil {
						return err
					}
				default:
					return fmt.Errorf("tipo de aplicación no soportado: %s", opts.Type)
				}
			}

			fmt.Printf("Aplicación desplegada correctamente en %s\n", opts.Domain)
			return nil
		},
	}
	// Agregar flags
	deployCmd.Flags().StringVarP(&opts.Domain, "domain", "d", "", "Dominio del sitio (obligatorio)")
	deployCmd.Flags().StringVarP(&opts.Repository, "repo", "r", "", "Repositorio Git (obligatorio)")
	deployCmd.Flags().StringVarP(&opts.Branch, "branch", "b", "main", "Rama del repositorio")
	deployCmd.Flags().StringVarP(&opts.Type, "type", "t", "", "Tipo de aplicación (laravel, nodejs)")
	deployCmd.Flags().StringVarP(&opts.Environment, "env", "e", "production", "Entorno (development, production)")
	deployCmd.Flags().BoolVarP(&useSSH, "ssh", "s", false, "Usar SSH para clonar el repositorio")

	// Marcar flags obligatorios
	deployCmd.MarkFlagRequired("domain")
	deployCmd.MarkFlagRequired("repo")

	// Validación de requisitos antes de ejecutar
	deployCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// Validar dominio
		if err := utils.ValidateDomain(opts.Domain); err != nil {
			return err
		}

		// Validar repositorio
		if err := utils.ValidateRepository(opts.Repository, useSSH); err != nil {
			return err
		}

		// Verificar requisitos
		requirements := map[string]string{
			"template": opts.Type,
		}

		return utils.CheckRequirements("deploy", requirements)
	}

	// Crear subcomando reset-pm2
	resetPM2Cmd := &cobra.Command{
		Use:   "reset-pm2",
		Short: "Reconfigurar PM2 para una aplicación existente",
		Long:  `Reconfigura PM2 para una aplicación existente, actualizando la configuración y reiniciando el servicio.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Configurar opciones
			if opts.Domain == "" {
				return fmt.Errorf("el dominio es obligatorio")
			}

			// Determinar si es un subdominio
			domainParts := strings.Split(opts.Domain, ".")
			if len(domainParts) > 2 && domainParts[0] != "www" {
				opts.IsSubdomain = true
				opts.ParentDomain = strings.Join(domainParts[1:], ".")
				fmt.Printf("Detectado subdominio de %s\n", opts.ParentDomain)

				// Usar el usuario del dominio principal para subdominios
				opts.User = strings.Split(opts.ParentDomain, ".")[0]
				opts.HomeDir = filepath.Join("/home", opts.ParentDomain)
				opts.AppDir = filepath.Join(opts.HomeDir, "apps", opts.Domain)
			} else {
				// No es subdominio, configuración normal
				opts.User = domainParts[0]
				opts.HomeDir = filepath.Join("/home", opts.Domain)
				opts.AppDir = filepath.Join(opts.HomeDir, "apps", opts.Domain)
			}

			// Verificar si el directorio de la aplicación existe
			if _, err := os.Stat(opts.AppDir); os.IsNotExist(err) {
				return fmt.Errorf("el directorio de la aplicación no existe: %s", opts.AppDir)
			}

			return resetPM2(&opts)
		},
	}

	// Agregar flags al subcomando reset-pm2
	resetPM2Cmd.Flags().StringVarP(&opts.Domain, "domain", "d", "", "Dominio del sitio (obligatorio)")
	resetPM2Cmd.MarkFlagRequired("domain")

	// Agregar subcomando reset-pm2 al comando deploy
	deployCmd.AddCommand(resetPM2Cmd)

	// Agregar comando al comando raíz
	rootCmd.AddCommand(deployCmd)
}

// cloneRepository clona el repositorio Git
func cloneRepository(opts *DeployOptions) error {
	// Extraer el nombre del repositorio de la URL
	repoName := opts.RepoName
	if repoName == "" {
		// Si no tenemos el nombre del repositorio, intentar extraerlo de la URL
		if strings.HasPrefix(opts.Repository, "git@github.com:") {
			parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(opts.Repository, "git@github.com:"), ".git"), "/")
			if len(parts) == 2 {
				repoName = parts[1]
			}
		} else if strings.HasPrefix(opts.Repository, "https://github.com/") {
			parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(opts.Repository, "https://github.com/"), ".git"), "/")
			if len(parts) == 2 {
				repoName = parts[1]
			}
		}
	}

	// Si aún no tenemos un nombre de repositorio, usar el subdominio como fallback
	if repoName == "" {
		repoName = strings.Split(opts.Domain, ".")[0]
	}

	// Construir la ruta del directorio de la aplicación
	// Usar el dominio completo como directorio principal y el nombre del repositorio como subdirectorio
	appDir := filepath.Join(opts.HomeDir, "apps", opts.Domain, repoName)
	opts.AppDir = appDir

	// Verificar si el directorio de la aplicación ya existe
	if _, err := os.Stat(appDir); err == nil {
		// Eliminar solo el directorio de esta aplicación específica
		fmt.Printf("Eliminando directorio de la aplicación existente en %s...\n", appDir)
		if err := os.RemoveAll(appDir); err != nil {
			return fmt.Errorf("error al eliminar directorio de la aplicación: %v", err)
		}
	}

	// Asegurarse de que el directorio padre existe y tiene los permisos correctos
	parentDir := filepath.Dir(appDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("error al crear directorio padre: %v", err)
	}

	// Cambiar propietario del directorio padre
	chownCmd := exec.Command("chown", fmt.Sprintf("%s:%s", opts.User, opts.User), parentDir)
	if output, err := chownCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario del directorio padre: %v\n%s", err, output)
	}

	// Si vamos a usar SSH, necesitamos manejar las claves
	if opts.UseSSH {
		if err := setupSSHKey(opts); err != nil {
			return err
		}
	}

	// Clonar repositorio
	fmt.Printf("Clonando repositorio %s en %s...\n", opts.Repository, appDir)

	var gitCmd *exec.Cmd

	if opts.UseSSH {
		// Usar el usuario para ejecutar git con la clave SSH correcta
		gitCmdStr := fmt.Sprintf("GIT_SSH_COMMAND='ssh -i %s -o IdentitiesOnly=yes -o StrictHostKeyChecking=no' git clone --branch %s --single-branch %s %s",
			opts.SSHKeyPath, opts.Branch, opts.Repository, appDir)

		gitCmd = exec.Command("su", "-c", gitCmdStr, opts.User)
	} else {
		// Clonar usando HTTPS
		gitCmd = exec.Command(
			"git", "clone",
			"--branch", opts.Branch,
			"--single-branch",
			opts.Repository,
			appDir,
		)
	}

	if output, err := gitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al clonar repositorio: %v\n%s", err, output)
	}

	// Cambiar propietario del directorio app
	finalChownCmd := exec.Command("chown", "-R", fmt.Sprintf("%s:%s", opts.User, opts.User), appDir)
	if output, err := finalChownCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario: %v\n%s", err, output)
	}

	fmt.Printf("Repositorio clonado correctamente en %s\n", appDir)
	return nil
}

// setupSSHKey configura la clave SSH para el despliegue
func setupSSHKey(opts *DeployOptions) error {
	// Directorio .ssh
	sshDir := filepath.Join(opts.HomeDir, ".ssh")

	// Verificar si el directorio .ssh existe, si no, crearlo
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("error al crear directorio .ssh: %v", err)
	}

	// Cambiar propietario del directorio .ssh
	cmd := exec.Command("chown", opts.User, sshDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario del directorio .ssh: %v\n%s", err, output)
	}

	// Verificar si la clave ya existe
	keyExists := false
	if _, err := os.Stat(opts.SSHKeyPath); err == nil {
		// La clave ya existe
		fmt.Printf("Clave SSH ya existe en %s\n", opts.SSHKeyPath)
		keyExists = true
	}

	if !keyExists {
		// Generar una nueva clave SSH
		fmt.Printf("Generando nueva clave SSH para %s...\n", opts.Domain)

		//keyFile := filepath.Base(opts.SSHKeyPath)
		keyComment := fmt.Sprintf("%s@%s", opts.User, opts.Domain)

		cmd := exec.Command("ssh-keygen",
			"-t", "ed25519",
			"-f", opts.SSHKeyPath,
			"-C", keyComment,
			"-N", "", // Sin passphrase
		)

		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al generar clave SSH: %v\n%s", err, output)
		}

		// Cambiar propietario de las claves
		cmd = exec.Command("chown", fmt.Sprintf("%s:%s", opts.User, opts.User), opts.SSHKeyPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al cambiar propietario de la clave privada: %v\n%s", err, output)
		}

		cmd = exec.Command("chown", fmt.Sprintf("%s:%s", opts.User, opts.User), opts.SSHKeyPath+".pub")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al cambiar propietario de la clave pública: %v\n%s", err, output)
		}

		// Mostrar la clave pública para que el usuario pueda agregarla a GitHub
		pubKeyContent, err := os.ReadFile(opts.SSHKeyPath + ".pub")
		if err != nil {
			return fmt.Errorf("error al leer clave pública: %v", err)
		}

		fmt.Printf("\n¡IMPORTANTE! Agrega esta clave a las Deploy Keys de tu repositorio:\n\n%s\n\n", pubKeyContent)
		fmt.Print("Presiona Enter para continuar cuando hayas agregado la clave en GitHub...")
		fmt.Scanln() // Esperar a que el usuario presione Enter

		// Agregar entrada al config de SSH
		configPath := filepath.Join(sshDir, "config")
		hostName := fmt.Sprintf("github-%s-%s", opts.RepoOwner, opts.RepoName)

		configContent := fmt.Sprintf(`
# Añadido por SiteManager para %s
Host %s
    Hostname github.com
    User git
    IdentityFile %s
    IdentitiesOnly yes
    StrictHostKeyChecking no
`, opts.Domain, hostName, opts.SSHKeyPath)

		// Verificar si el archivo config existe
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// Crear archivo config
			if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
				return fmt.Errorf("error al crear archivo config de SSH: %v", err)
			}
		} else {
			// Añadir al archivo config existente
			f, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				return fmt.Errorf("error al abrir archivo config de SSH: %v", err)
			}
			defer f.Close()

			if _, err = f.WriteString(configContent); err != nil {
				return fmt.Errorf("error al escribir en archivo config de SSH: %v", err)
			}
		}

		// Cambiar propietario del archivo config
		cmd = exec.Command("chown", fmt.Sprintf("%s:%s", opts.User, opts.User), configPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al cambiar propietario del archivo config: %v\n%s", err, output)
		}
	}

	return nil
}

// deployLaravel configura y despliega una aplicación Laravel
func deployLaravel(opts *DeployOptions) error {
	// Esta es una implementación básica, puede ser expandida según necesidades
	fmt.Printf("Desplegando aplicación Laravel en %s...\n", opts.Domain)

	// Cambiar al directorio de la aplicación
	if err := os.Chdir(opts.AppDir); err != nil {
		return fmt.Errorf("error al cambiar al directorio de la aplicación: %v", err)
	}

	// Verificar si es una aplicación Laravel
	if _, err := os.Stat(filepath.Join(opts.AppDir, "artisan")); os.IsNotExist(err) {
		return fmt.Errorf("no se encontró el archivo artisan en %s, ¿es una aplicación Laravel?", opts.AppDir)
	}

	// Crear directorios necesarios
	dirsToCreate := []string{
		filepath.Join(opts.AppDir, "bootstrap/cache"),
		filepath.Join(opts.AppDir, "storage/app"),
		filepath.Join(opts.AppDir, "storage/app/public"),
		filepath.Join(opts.AppDir, "storage/framework"),
		filepath.Join(opts.AppDir, "storage/framework/cache"),
		filepath.Join(opts.AppDir, "storage/framework/sessions"),
		filepath.Join(opts.AppDir, "storage/framework/views"),
		filepath.Join(opts.AppDir, "storage/logs"),
	}

	for _, dir := range dirsToCreate {
		if err := os.MkdirAll(dir, 0775); err != nil {
			return fmt.Errorf("error al crear directorio %s: %v", dir, err)
		}
	}

	// Cambiar propietario de los directorios
	cmd := exec.Command("chown", "-R", fmt.Sprintf("%s:%s", opts.User, opts.User), opts.AppDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario: %v\n%s", err, output)
	}

	// Ejecutar comandos como el usuario del sitio
	commands := []string{
		// Instalar dependencias
		"composer install --no-dev --optimize-autoloader",
		// Copiar .env.example a .env si no existe
		"[ -f .env ] || cp .env.example .env",
		// Generar clave de aplicación
		"php artisan key:generate",
		// Crear enlace simbólico para storage
		"php artisan storage:link",
		// Ejecutar migraciones, pero permitir que falle (por si la BD no está configurada)
		"php artisan migrate --force || echo 'Error en migraciones. Verifica la configuración de la base de datos'",
		// Optimizaciones
		"php artisan config:cache",
		"php artisan route:cache || echo 'No se pudieron cachear las rutas'",
		"php artisan view:cache",
	}

	for _, cmdStr := range commands {
		cmd := exec.Command("su", "-c", cmdStr, opts.User)
		cmd.Dir = opts.AppDir // Establecer el directorio de trabajo
		output, err := cmd.CombinedOutput()
		fmt.Printf("Ejecutando: %s\n", cmdStr)
		fmt.Printf("Salida: %s\n", output)

		// Para algunos comandos, queremos continuar incluso si fallan
		if err != nil && !strings.Contains(cmdStr, "echo 'Error") && !strings.Contains(cmdStr, "echo 'No se") {
			return fmt.Errorf("error al ejecutar comando '%s': %v\n%s", cmdStr, err, output)
		}
	}

	// Configurar directorios públicos
	var publicHtmlPath string
	if opts.IsSubdomain {
		// Para subdominios, usar una carpeta específica dentro de public_html
		publicHtmlPath = filepath.Join(opts.HomeDir, "public_html", opts.Domain)
		// Asegurarse de que el directorio existe
		if err := os.MkdirAll(filepath.Dir(publicHtmlPath), 0755); err != nil {
			return fmt.Errorf("error al crear directorio para subdominio: %v", err)
		}
	} else {
		// Para dominio principal, usar public_html directamente
		publicHtmlPath = filepath.Join(opts.HomeDir, "public_html")
	}

	appPublicPath := filepath.Join(opts.AppDir, "public")

	// Eliminar carpeta existente
	if err := os.RemoveAll(publicHtmlPath); err != nil {
		return fmt.Errorf("error al eliminar directorio público existente: %v", err)
	}

	// Crear enlace simbólico
	if err := os.Symlink(appPublicPath, publicHtmlPath); err != nil {
		return fmt.Errorf("error al crear enlace simbólico: %v", err)
	}

	// Cambiar propietario del enlace
	cmd = exec.Command("chown", "-h", fmt.Sprintf("%s:%s", opts.User, opts.User), publicHtmlPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario del enlace: %v\n%s", err, output)
	}

	fmt.Printf("Aplicación Laravel desplegada correctamente en %s\n", opts.Domain)
	return nil
}

// Modificación en deployNodejs
func deployNodejs(opts *DeployOptions) error {
	fmt.Printf("Desplegando aplicación Node.js en %s...\n", opts.Domain)

	// Cambiar al directorio de la aplicación
	if err := os.Chdir(opts.AppDir); err != nil {
		return fmt.Errorf("error al cambiar al directorio de la aplicación: %v", err)
	}

	// Verificar si hay package.json
	if _, err := os.Stat(filepath.Join(opts.AppDir, "package.json")); os.IsNotExist(err) {
		return fmt.Errorf("no se encontró package.json en %s", opts.AppDir)
	}

	// Detectar framework Node.js y obtener información del proyecto
	projectInfo, err := utils.DetectNodeJSFramework(opts.AppDir)
	if err != nil {
		return fmt.Errorf("error al detectar framework: %v", err)
	}

	fmt.Printf("Framework detectado: %s\n", projectInfo.Framework)
	if projectInfo.HasTypeScript {
		fmt.Println("Proyecto con TypeScript: Sí")
	}

	// Determinar puerto para la aplicación - MOVER ESTA SECCIÓN AQUÍ ARRIBA
	port := projectInfo.DefaultPort
	if opts.IsSubdomain {
		// Para subdominios, usar un puerto diferente basado en una función hash simple
		h := fnv.New32a()
		h.Write([]byte(opts.Domain))
		port = 3000 + int(h.Sum32()%1000) // Puertos entre 3000 y 3999
	}

	// Verificar si necesita variables de entorno
	if projectInfo.RequiresEnv {
		fmt.Println("El proyecto requiere variables de entorno, configurando...")

		// Variables de entorno predeterminadas según el framework
		userEnvVars := make(map[string]string)

		// Si es una aplicación con una base de datos
		if projectInfo.RequiresDatabase {
			fmt.Printf("El proyecto requiere una base de datos (%s)\n", projectInfo.DBType)

			// Configurar base de datos según el tipo
			switch projectInfo.DBType {
			case "postgresql":
				// Verificar si PostgreSQL está disponible
				pgCheckCmd := exec.Command("pg_isready")
				if output, err := pgCheckCmd.CombinedOutput(); err != nil {
					fmt.Printf("Advertencia: PostgreSQL no está disponible: %v\n%s\n", err, output)
					fmt.Println("Se omitirá la configuración de la base de datos. Por favor, configure PostgreSQL manualmente.")
					// Continuar sin configurar la base de datos
					break
				}

				dbName := strings.ReplaceAll(opts.Domain, ".", "_")
				dbUser := strings.ReplaceAll(opts.User, "-", "_")
				dbPassword := generateRandomPassword(16)

				dbOpts := &utils.DatabaseOptions{
					Type:     utils.DBTypePostgreSQL,
					Host:     "localhost",
					Port:     5432,
					Name:     dbName,
					User:     dbUser,
					Password: dbPassword,
					Schema:   "public",
					SSLMode:  "prefer",
				}

				// Crear base de datos
				fmt.Println("Creando base de datos PostgreSQL...")
				if err := utils.CreateDatabase(dbOpts); err != nil {
					fmt.Printf("Advertencia: error al crear base de datos: %v\n", err)
					fmt.Println("Se omitirá la configuración de la base de datos. Por favor, configure PostgreSQL manualmente.")
					// Continuar aunque haya error en la creación de la base de datos
				} else {
					// Agregar URL de conexión a las variables de entorno
					userEnvVars["DATABASE_URL"] = utils.BuildDatabaseURL(dbOpts)
				}
			case "mysql":
				dbName := strings.ReplaceAll(opts.Domain, ".", "_")
				dbUser := strings.ReplaceAll(opts.User, "-", "_")
				dbPassword := generateRandomPassword(16)

				dbOpts := &utils.DatabaseOptions{
					Type:      utils.DBTypeMySQL,
					Host:      "localhost",
					Port:      3306,
					Name:      dbName,
					User:      dbUser,
					Password:  dbPassword,
					Charset:   "utf8mb4",
					Collation: "utf8mb4_unicode_ci",
				}

				// Crear base de datos
				fmt.Println("Creando base de datos MySQL...")
				if err := utils.CreateDatabase(dbOpts); err != nil {
					fmt.Printf("Advertencia: error al crear base de datos: %v\n", err)
					// Continuar aunque haya error en la creación de la base de datos
				} else {
					// Agregar URL de conexión a las variables de entorno
					userEnvVars["DATABASE_URL"] = utils.BuildDatabaseURL(dbOpts)
				}
			}
		}

		// Agregar variables específicas según el framework
		switch projectInfo.Framework {
		case utils.FrameworkNestJS:
			if _, ok := projectInfo.EnvVars["JWT_SECRET"]; ok {
				userEnvVars["JWT_SECRET"] = generateRandomString(32)
			}
			if _, ok := projectInfo.EnvVars["JWT_EXPIRES_IN"]; ok {
				userEnvVars["JWT_EXPIRES_IN"] = "7d"
			}
			userEnvVars["NODE_ENV"] = "production"
			userEnvVars["PORT"] = fmt.Sprintf("%d", port) // Ahora port ya está definido
		case utils.FrameworkNextJS:
			userEnvVars["NODE_ENV"] = "production"
			userEnvVars["PORT"] = fmt.Sprintf("%d", port) // Ahora port ya está definido
			// Agregar variables específicas para NextJS
			if _, ok := projectInfo.EnvVars["NEXT_PUBLIC_API_URL"]; ok {
				// Si estamos en un subdominio, la API podría estar en otro subdominio
				if opts.IsSubdomain {
					apiSubdomain := "api." + opts.ParentDomain
					userEnvVars["NEXT_PUBLIC_API_URL"] = "https://" + apiSubdomain
				} else {
					userEnvVars["NEXT_PUBLIC_API_URL"] = "https://api." + opts.Domain
				}
			}
		case utils.FrameworkExpress:
			userEnvVars["NODE_ENV"] = "production"
			userEnvVars["PORT"] = fmt.Sprintf("%d", port) // Ahora port ya está definido
		}

		// Configurar archivo .env
		if err := utils.ConfigureNodeJSEnv(opts.AppDir, projectInfo, userEnvVars); err != nil {
			fmt.Printf("Advertencia: error al configurar archivo .env: %v\n", err)
			// Continuamos aunque haya error en el .env
		} else {
			fmt.Println("Archivo .env configurado correctamente")

			// Cambiar propietario del archivo .env
			envFilePath := filepath.Join(opts.AppDir, ".env")
			chownCmd := exec.Command("chown", fmt.Sprintf("%s:%s", opts.User, opts.User), envFilePath)
			if output, err := chownCmd.CombinedOutput(); err != nil {
				fmt.Printf("Advertencia: error al cambiar propietario del archivo .env: %v\n%s\n", err, output)
			}
		}

		// Si es un proyecto con Prisma, ejecutar prisma generate y migrate
		if projectInfo.HasPrisma {
			fmt.Println("Proyecto con Prisma detectado, ejecutando migraciones...")

			// Verificar si PostgreSQL está disponible antes de ejecutar comandos de Prisma
			pgCheckCmd := exec.Command("pg_isready")
			if output, err := pgCheckCmd.CombinedOutput(); err != nil {
				fmt.Printf("Advertencia: PostgreSQL no está disponible: %v\n%s\n", err, output)
				fmt.Println("Se omitirán las operaciones de Prisma. Por favor, configure PostgreSQL manualmente y ejecute las migraciones después.")
			} else {
				// Ejecutar comandos de Prisma
				prismaCommands := []string{
					// Generar cliente Prisma
					"npx prisma generate",
					// Ejecutar migraciones (solo si existe la carpeta migrations)
					"[ -d prisma/migrations ] && npx prisma migrate deploy || echo 'No hay migraciones para aplicar'",
				}

				for _, cmdStr := range prismaCommands {
					cmd := exec.Command("su", "-c", cmdStr, opts.User)
					cmd.Dir = opts.AppDir
					fmt.Printf("Ejecutando: %s\n", cmdStr)
					if output, err := cmd.CombinedOutput(); err != nil {
						// Para migraciones, podemos continuar aunque haya errores
						if strings.Contains(cmdStr, "migrate") {
							fmt.Printf("Advertencia: error en migraciones de Prisma (no crítico): %v\n%s\n", err, output)
						} else {
							fmt.Printf("Error al ejecutar comando Prisma: %v\n%s\n", err, output)
							// Para otros comandos, podría ser más crítico pero continuamos
						}
					} else if len(output) > 0 {
						fmt.Printf("Salida: %s\n", output)
					}
				}
			}
		}
	}

	// Ejecutar comandos como el usuario del sitio
	commands := []string{
		// Instalar dependencias
		"npm ci || npm install",
	}

	// Si hay comando de build, agregarlo
	buildCmd := utils.GetNodeJSBuildCommand(projectInfo)
	if buildCmd != "" {
		commands = append(commands, buildCmd)
	}

	for _, cmdStr := range commands {
		cmd := exec.Command("su", "-c", cmdStr, opts.User)
		cmd.Dir = opts.AppDir // Establecer el directorio de trabajo
		fmt.Printf("Ejecutando: %s\n", cmdStr)
		output, err := cmd.CombinedOutput()

		if len(output) > 0 {
			fmt.Printf("Salida: %s\n", output)
		}

		if err != nil {
			// Para npm install, ignoramos errores que son comunes debido a dependencias opcionales
			if strings.Contains(cmdStr, "npm") && strings.Contains(cmdStr, "install") {
				fmt.Printf("Advertencia: hubo algunos errores en la instalación de dependencias, pero continuaremos: %v\n", err)
			} else {
				return fmt.Errorf("error al ejecutar comando '%s': %v\n%s", cmdStr, err, output)
			}
		}
	}

	// ELIMINAR ESTA SECCIÓN YA QUE LA MOVIMOS ARRIBA
	// Determinar puerto para la aplicación
	// port := projectInfo.DefaultPort
	// if opts.IsSubdomain {
	// 	// Para subdominios, usar un puerto diferente basado en una función hash simple
	// 	h := fnv.New32a()
	// 	h.Write([]byte(opts.Domain))
	// 	port = 3000 + int(h.Sum32()%1000) // Puertos entre 3000 y 3999
	// }

	// Determinar comando para iniciar la aplicación
	startCommand := utils.GetNodeJSStartCommand(projectInfo)

	// Para frameworks específicos, configurar comando de inicio
	switch projectInfo.Framework {
	case utils.FrameworkNestJS:
		if projectInfo.HasTypeScript {
			// Para NestJS con TypeScript, asegurarnos de que usamos la versión compilada
			startCommand = "node dist/main.js"
		}
	case utils.FrameworkNextJS:
		// Para Next.js, necesitamos un puerto específico
		startCommand = fmt.Sprintf("next start -p %d", port)
	}

	// Crear archivo de configuración para PM2
	pmConfig := fmt.Sprintf(`{
  "apps": [{
    "name": "%s",
    "script": "%s",
    "cwd": "%s",
    "env": {
      "NODE_ENV": "production",
      "PORT": "%d"
    },
    "error_file": "%s/logs/%s_error.log",
    "out_file": "%s/logs/%s_output.log",
    "merge_logs": true,
    "max_memory_restart": "200M",
    "restart_delay": 3000,
    "watch": false,
    "exec_mode": "fork",
    "instances": 1,
    "autorestart": true
  }]
}`, opts.Domain, startCommand, opts.AppDir, port, opts.HomeDir, opts.Domain, opts.HomeDir, opts.Domain)

	// Asegurarse de que el directorio de logs existe
	logsDir := filepath.Join(opts.HomeDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("error al crear directorio de logs: %v", err)
	}

	// Cambiar propietario del directorio de logs
	logChownCmd := exec.Command("chown", fmt.Sprintf("%s:%s", opts.User, opts.User), logsDir)
	if output, err := logChownCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario del directorio de logs: %v\n%s", err, output)
	}

	// Crear archivos de log con los permisos correctos
	errorLogPath := filepath.Join(logsDir, fmt.Sprintf("%s_error.log", opts.Domain))
	outputLogPath := filepath.Join(logsDir, fmt.Sprintf("%s_output.log", opts.Domain))

	// Crear archivos de log vacíos
	if err := os.WriteFile(errorLogPath, []byte(""), 0644); err != nil {
		return fmt.Errorf("error al crear archivo de log de error: %v", err)
	}
	if err := os.WriteFile(outputLogPath, []byte(""), 0644); err != nil {
		return fmt.Errorf("error al crear archivo de log de salida: %v", err)
	}

	// Cambiar propietario de los archivos de log
	logFilesChownCmd := exec.Command("chown", fmt.Sprintf("%s:%s", opts.User, opts.User), errorLogPath, outputLogPath)
	if output, err := logFilesChownCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario de los archivos de log: %v\n%s", err, output)
	}

	configPath := filepath.Join(opts.HomeDir, fmt.Sprintf("pm2.%s.config.json", opts.Domain))
	if err := os.WriteFile(configPath, []byte(pmConfig), 0644); err != nil {
		return fmt.Errorf("error al crear archivo de configuración PM2: %v", err)
	}

	// Cambiar propietario del archivo de configuración
	cmd := exec.Command("chown", fmt.Sprintf("%s:%s", opts.User, opts.User), configPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario: %v\n%s", err, output)
	}

	// Detener la aplicación si ya está en ejecución (tanto en root como en el usuario)
	exec.Command("pm2", "delete", opts.Domain).Run()                          // Ignorar errores aquí
	exec.Command("sudo", "-u", opts.User, "pm2", "delete", opts.Domain).Run() // Ignorar errores aquí

	// Asegurarse de que PM2 está configurado para el usuario
	startupCmd := exec.Command("sudo", "-u", opts.User, "pm2", "startup")
	if output, err := startupCmd.CombinedOutput(); err != nil {
		fmt.Printf("Advertencia: error al configurar PM2 startup (no crítico): %v\n", err)
		fmt.Printf("Salida de PM2 startup: %s\n", output)
	}

	// Iniciar la aplicación como el usuario
	startCmd := exec.Command("sudo", "-u", opts.User, "pm2", "start", configPath)

	output, err := startCmd.CombinedOutput()
	if len(output) > 0 {
		fmt.Printf("Salida de PM2: %s\n", output)
	}

	if err != nil {
		return fmt.Errorf("error al iniciar aplicación con PM2: %v\n%s", err, output)
	}

	// Guardar la configuración de PM2 para que persista después de reiniciar
	saveCmd := exec.Command("sudo", "-u", opts.User, "pm2", "save")
	if output, err := saveCmd.CombinedOutput(); err != nil {
		fmt.Printf("Advertencia: error al guardar configuración PM2 (no crítico): %v\n", err)
		fmt.Printf("Salida de PM2: %s\n", output)
	}

	// Actualizar la configuración de Nginx para usar el puerto correcto
	if opts.IsSubdomain {
		nginxConfPath := filepath.Join(opts.HomeDir, ".nginx", fmt.Sprintf("%s.conf", opts.Domain))
		if _, err := os.Stat(nginxConfPath); err == nil {
			// Leer el archivo
			confData, err := os.ReadFile(nginxConfPath)
			if err != nil {
				return fmt.Errorf("error al leer configuración Nginx: %v", err)
			}

			// Reemplazar el puerto
			newConf := strings.ReplaceAll(string(confData), "proxy_pass http://localhost:3000", fmt.Sprintf("proxy_pass http://localhost:%d", port))

			// Escribir el archivo
			if err := os.WriteFile(nginxConfPath, []byte(newConf), 0644); err != nil {
				return fmt.Errorf("error al escribir configuración Nginx: %v", err)
			}

			// Cambiar propietario del archivo de configuración
			chownCmd := exec.Command("chown", fmt.Sprintf("%s:%s", opts.User, opts.User), nginxConfPath)
			if output, err := chownCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("error al cambiar propietario del archivo de configuración: %v\n%s", err, output)
			}

			// Recargar Nginx
			reloadCmd := exec.Command("systemctl", "reload", "nginx")
			if output, err := reloadCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("error al recargar Nginx: %v\n%s", err, output)
			}
		}
	}

	fmt.Printf("Aplicación Node.js desplegada correctamente en %s\n", opts.Domain)
	fmt.Printf("Tipo: %s\n", projectInfo.Framework)
	fmt.Printf("Puerto: %d\n", port)
	fmt.Printf("Logs: %s/logs/%s_*.log\n", opts.HomeDir, opts.Domain)

	// Mostrar URL de la aplicación
	if opts.IsSubdomain {
		fmt.Printf("URL: http://%s\n", opts.Domain)
	} else {
		fmt.Printf("URL: http://%s\n", opts.Domain)
	}

	return nil
}

// generateRandomPassword genera una contraseña aleatoria
func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// generateRandomString genera una cadena aleatoria (sin caracteres especiales)
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// resetPM2 reinicia la configuración de PM2 para una aplicación existente
func resetPM2(opts *DeployOptions) error {
	fmt.Printf("Reconfigurando PM2 para %s...\n", opts.Domain)

	// Detectar framework Node.js y obtener información del proyecto
	projectInfo, err := utils.DetectNodeJSFramework(opts.AppDir)
	if err != nil {
		return fmt.Errorf("error al detectar framework: %v", err)
	}

	// Determinar puerto para la aplicación
	port := projectInfo.DefaultPort
	if opts.IsSubdomain {
		// Para subdominios, usar un puerto diferente basado en una función hash simple
		h := fnv.New32a()
		h.Write([]byte(opts.Domain))
		port = 3000 + int(h.Sum32()%1000) // Puertos entre 3000 y 3999
	}

	// Determinar comando para iniciar la aplicación
	startCommand := utils.GetNodeJSStartCommand(projectInfo)

	// Para frameworks específicos, configurar comando de inicio
	switch projectInfo.Framework {
	case utils.FrameworkNestJS:
		if projectInfo.HasTypeScript {
			// Para NestJS con TypeScript, asegurarnos de que usamos la versión compilada
			startCommand = "node dist/main.js"
		}
	case utils.FrameworkNextJS:
		// Para Next.js, necesitamos un puerto específico
		startCommand = fmt.Sprintf("next start -p %d", port)
	}

	// Crear archivo de configuración para PM2
	pmConfig := fmt.Sprintf(`{
  "apps": [{
    "name": "%s",
    "script": "%s",
    "cwd": "%s",
    "env": {
      "NODE_ENV": "production",
      "PORT": "%d"
    },
    "error_file": "%s/logs/%s_error.log",
    "out_file": "%s/logs/%s_output.log",
    "merge_logs": true,
    "max_memory_restart": "200M",
    "restart_delay": 3000,
    "watch": false,
    "exec_mode": "fork",
    "instances": 1,
    "autorestart": true
  }]
}`, opts.Domain, startCommand, opts.AppDir, port, opts.HomeDir, opts.Domain, opts.HomeDir, opts.Domain)

	// Asegurarse de que el directorio de logs existe
	logsDir := filepath.Join(opts.HomeDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("error al crear directorio de logs: %v", err)
	}

	// Cambiar propietario del directorio de logs
	logChownCmd := exec.Command("chown", fmt.Sprintf("%s:%s", opts.User, opts.User), logsDir)
	if output, err := logChownCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario del directorio de logs: %v\n%s", err, output)
	}

	// Crear archivos de log con los permisos correctos
	errorLogPath := filepath.Join(logsDir, fmt.Sprintf("%s_error.log", opts.Domain))
	outputLogPath := filepath.Join(logsDir, fmt.Sprintf("%s_output.log", opts.Domain))

	// Crear archivos de log vacíos
	if err := os.WriteFile(errorLogPath, []byte(""), 0644); err != nil {
		return fmt.Errorf("error al crear archivo de log de error: %v", err)
	}
	if err := os.WriteFile(outputLogPath, []byte(""), 0644); err != nil {
		return fmt.Errorf("error al crear archivo de log de salida: %v", err)
	}

	// Cambiar propietario de los archivos de log
	logFilesChownCmd := exec.Command("chown", fmt.Sprintf("%s:%s", opts.User, opts.User), errorLogPath, outputLogPath)
	if output, err := logFilesChownCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario de los archivos de log: %v\n%s", err, output)
	}

	configPath := filepath.Join(opts.HomeDir, fmt.Sprintf("pm2.%s.config.json", opts.Domain))
	if err := os.WriteFile(configPath, []byte(pmConfig), 0644); err != nil {
		return fmt.Errorf("error al crear archivo de configuración PM2: %v", err)
	}

	// Cambiar propietario del archivo de configuración
	cmd := exec.Command("chown", fmt.Sprintf("%s:%s", opts.User, opts.User), configPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario: %v\n%s", err, output)
	}

	// Detener la aplicación si ya está en ejecución (tanto en root como en el usuario)
	exec.Command("pm2", "delete", opts.Domain).Run()                          // Ignorar errores aquí
	exec.Command("sudo", "-u", opts.User, "pm2", "delete", opts.Domain).Run() // Ignorar errores aquí

	// Asegurarse de que PM2 está configurado para el usuario
	startupCmd := exec.Command("sudo", "-u", opts.User, "pm2", "startup")
	if output, err := startupCmd.CombinedOutput(); err != nil {
		fmt.Printf("Advertencia: error al configurar PM2 startup (no crítico): %v\n", err)
		fmt.Printf("Salida de PM2 startup: %s\n", output)
	}

	// Iniciar la aplicación como el usuario
	startCmd := exec.Command("sudo", "-u", opts.User, "pm2", "start", configPath)

	output, err := startCmd.CombinedOutput()
	if len(output) > 0 {
		fmt.Printf("Salida de PM2: %s\n", output)
	}

	if err != nil {
		return fmt.Errorf("error al iniciar aplicación con PM2: %v\n%s", err, output)
	}

	// Guardar la configuración de PM2 para que persista después de reiniciar
	saveCmd := exec.Command("sudo", "-u", opts.User, "pm2", "save")
	if output, err := saveCmd.CombinedOutput(); err != nil {
		fmt.Printf("Advertencia: error al guardar configuración PM2 (no crítico): %v\n", err)
		fmt.Printf("Salida de PM2: %s\n", output)
	}

	fmt.Printf("PM2 reconfigurado correctamente para %s\n", opts.Domain)
	return nil
}
