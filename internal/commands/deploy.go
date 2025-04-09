package commands

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	Backup       bool
}

// AddDeployCommand agrega el comando deploy al comando raíz
func AddDeployCommand(rootCmd *cobra.Command, cfg *config.Config) {
	// Inicializar semilla para números aleatorios
	// rand.Seed(time.Now().UnixNano()) // Deprecated in Go 1.20+

	// Opciones del comando
	var opts DeployOptions
	var useSSH bool
	var dbType string // Declaración de la variable para el tipo de base de datos

	// Crear comando deploy
	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Desplegar una aplicación web",
		Long:  `Despliega una aplicación web desde un repositorio Git y configura el entorno necesario.`,
		RunE: func(cmd *cobra.Command, args []string) error {

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

			// Verificar si el sitio existe antes de continuar
			if !siteExists(opts.Domain, cfg) {
				return fmt.Errorf("el sitio %s no existe, primero crea el sitio con 'sm site'", opts.Domain)
			}

			// Crear la estructura de directorios necesaria
			fmt.Printf("Creando estructura de directorios en %s...\n", filepath.Dir(opts.AppDir))

			// Crear el directorio apps si no existe
			appsDir := filepath.Join(opts.HomeDir, "apps")
			if _, err := os.Stat(appsDir); os.IsNotExist(err) {
				fmt.Printf("Creando directorio apps en %s...\n", appsDir)
				if err := os.MkdirAll(appsDir, 0755); err != nil {
					return fmt.Errorf("error al crear directorio apps: %v", err)
				}
			}

			// Crear todos los directorios padres necesarios
			if err := os.MkdirAll(filepath.Dir(opts.AppDir), 0755); err != nil {
				return fmt.Errorf("error al crear directorios padres: %v", err)
			}

			// Cambiar propietario de los directorios padres
			chownCmd := exec.Command("chown", "-R", fmt.Sprintf("%s:%s", opts.User, opts.User), filepath.Join(opts.HomeDir, "apps"))
			if output, err := chownCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("error al cambiar propietario de los directorios padres: %v\n%s", err, output)
			}

			// Generar nombre para la clave SSH
			if opts.UseSSH {
				// Sanitizar nombres para usarlos en el nombre de archivo
				domainSafe := strings.ReplaceAll(opts.Domain, ".", "_")
				ownerSafe := strings.ReplaceAll(opts.RepoOwner, "-", "_")
				repoSafe := strings.ReplaceAll(opts.RepoName, "-", "_")

				keyName := fmt.Sprintf("%s_%s_%s", domainSafe, ownerSafe, repoSafe)
				opts.SSHKeyPath = filepath.Join(opts.HomeDir, ".ssh", keyName)

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
					if err := deployNodejs(&opts, dbType); err != nil {
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
	deployCmd.Flags().StringVar(&dbType, "database", "", "Tipo de base de datos a configurar (postgresql, mysql)")

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

	// Crear subcomando remove
	removeCmd := &cobra.Command{
		Use:   "remove",
		Short: "Eliminar un proyecto desplegado",
		Long:  `Elimina un proyecto desplegado, deteniendo PM2, eliminando la aplicación y opcionalmente haciendo backup de la carpeta del proyecto.`,
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

			return removeDeployedProject(&opts)
		},
	}

	// Agregar flags al subcomando remove
	removeCmd.Flags().StringVarP(&opts.Domain, "domain", "d", "", "Dominio del sitio (obligatorio)")
	removeCmd.Flags().BoolVar(&opts.Backup, "backup", false, "Hacer backup de la carpeta del proyecto antes de eliminar")
	removeCmd.MarkFlagRequired("domain")

	// Agregar subcomando remove al comando deploy
	deployCmd.AddCommand(removeCmd)

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
	var appDir string
	if opts.IsSubdomain {
		// Para subdominios, usar la estructura /home/parentdomain/apps/subdomain/reponame
		appDir = filepath.Join(opts.HomeDir, "apps", opts.Domain, repoName)
	} else {
		// Para dominios principales, usar la estructura /home/domain/apps/domain/reponame
		appDir = filepath.Join(opts.HomeDir, "apps", opts.Domain, repoName)
	}
	opts.AppDir = appDir

	// Crear la estructura de directorios necesaria
	parentDir := filepath.Dir(appDir)
	fmt.Printf("Creando estructura de directorios en %s...\n", parentDir)

	// Crear todos los directorios padres necesarios
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("error al crear directorios padres: %v", err)
	}

	// Cambiar propietario de los directorios padres
	chownCmd := exec.Command("chown", "-R", fmt.Sprintf("%s:%s", opts.User, opts.User), filepath.Join(opts.HomeDir, "apps"))
	if output, err := chownCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario de los directorios padres: %v\n%s", err, output)
	}

	// Verificar si el directorio de la aplicación ya existe
	if _, err := os.Stat(appDir); err == nil {
		// Eliminar solo el directorio de esta aplicación específica
		fmt.Printf("Eliminando directorio de la aplicación existente en %s...\n", appDir)
		if err := os.RemoveAll(appDir); err != nil {
			return fmt.Errorf("error al eliminar directorio de la aplicación: %v", err)
		}
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
func deployNodejs(opts *DeployOptions, dbType string) error {
	fmt.Printf("Desplegando aplicación Node.js en %s...\n", opts.Domain)

	// Verificar si el directorio de la aplicación existe
	if _, err := os.Stat(opts.AppDir); os.IsNotExist(err) {
		return fmt.Errorf("el directorio de la aplicación no existe: %s", opts.AppDir)
	}

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

	// AQUÍ ES DONDE DEBEMOS COLOCAR NUESTRA LÓGICA DE SELECCIÓN DE BASE DE DATOS
	// Si se especificó un tipo de base de datos mediante flag, forzar su uso
	if dbType != "" {
		switch dbType {
		case "postgresql", "postgres", "pg":
			projectInfo.RequiresDatabase = true
			projectInfo.DBType = "postgresql"
			fmt.Println("Forzando tipo de base de datos: PostgreSQL")
		case "mysql":
			projectInfo.RequiresDatabase = true
			projectInfo.DBType = "mysql"
			fmt.Println("Forzando tipo de base de datos: MySQL")
		default:
			fmt.Printf("Tipo de base de datos no reconocido: %s. Se usará la detección automática.\n", dbType)
		}
	}

	// Determinar puerto para la aplicación
	var port int
	if opts.IsSubdomain {
		// Para subdominios, usar un puerto aleatorio
		h := fnv.New32a()
		h.Write([]byte(opts.Domain))
		port = 3001 + int(h.Sum32()%999) // Puertos entre 3001 y 3999
		fmt.Printf("Generando puerto aleatorio para subdominio: %d\n", port)
	} else {
		// Para dominios principales, usar un puerto aleatorio
		h := fnv.New32a()
		h.Write([]byte(opts.Domain))
		port = 3001 + int(h.Sum32()%999) // Puertos entre 3001 y 3999
		fmt.Printf("Generando puerto aleatorio para dominio principal: %d\n", port)
	}

	// Actualizar la configuración de Nginx con el nuevo puerto
	nginxConfPath := filepath.Join(opts.HomeDir, "nginx", fmt.Sprintf("%s.conf", opts.Domain))
	if _, err := os.Stat(nginxConfPath); err == nil {
		// Leer el archivo
		confData, err := os.ReadFile(nginxConfPath)
		if err != nil {
			return fmt.Errorf("error al leer configuración Nginx: %v", err)
		}

		// Reemplazar el puerto en la configuración
		confStr := string(confData)
		var newConf string

		// Buscar y reemplazar cualquier proxy_pass existente
		if strings.Contains(confStr, "proxy_pass http://localhost:") {
			// Extraer el puerto actual
			portParts := strings.Split(confStr, "proxy_pass http://localhost:")
			if len(portParts) > 1 {
				portEnd := strings.Index(portParts[1], ";")
				if portEnd > 0 {
					oldPort := portParts[1][:portEnd]
					// Reemplazar el puerto antiguo con el nuevo
					newConf = strings.ReplaceAll(confStr, "proxy_pass http://localhost:"+oldPort, fmt.Sprintf("proxy_pass http://localhost:%d", port))
				} else {
					// Si no se puede extraer el puerto, simplemente reemplazar toda la línea
					newConf = strings.ReplaceAll(confStr, "proxy_pass http://localhost:", fmt.Sprintf("proxy_pass http://localhost:%d", port))
				}
			} else {
				// Si no se puede extraer el puerto, simplemente reemplazar toda la línea
				newConf = strings.ReplaceAll(confStr, "proxy_pass http://localhost:", fmt.Sprintf("proxy_pass http://localhost:%d", port))
			}
		} else {
			// Si no hay proxy_pass, agregar uno nuevo
			// Buscar la ubicación correcta para agregar el proxy_pass
			if strings.Contains(confStr, "location / {") {
				newConf = strings.ReplaceAll(confStr, "location / {", fmt.Sprintf("location / {\n        proxy_pass http://localhost:%d;", port))
			} else if strings.Contains(confStr, "location /") {
				newConf = strings.ReplaceAll(confStr, "location /", fmt.Sprintf("location / {\n        proxy_pass http://localhost:%d;", port))
			} else {
				// Si no se encuentra una ubicación adecuada, agregar al final
				newConf = confStr + fmt.Sprintf("\n    location / {\n        proxy_pass http://localhost:%d;\n    }", port)
			}
		}

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

		fmt.Printf("Configuración de Nginx actualizada para usar el puerto %d\n", port)
	} else {
		fmt.Printf("No se encontró archivo de configuración Nginx en %s\n", nginxConfPath)
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
				// Usar la función mejorada de configuración de PostgreSQL
				pgEnvVars, pgErr := setupPostgreSQLDatabase(opts, projectInfo)
				if pgErr != nil {
					fmt.Printf("Advertencia: error al configurar PostgreSQL: %v\n", pgErr)
					fmt.Println("Se omitirá la configuración automática de la base de datos.")
					fmt.Println("Por favor, configure PostgreSQL manualmente y actualice el archivo .env")

					// Agregar variables de entorno de respaldo para que Prisma no falle
					dbName := strings.ReplaceAll(opts.Domain, ".", "_")
					dbUser := opts.User
					dbPassword := "dev_password_please_change" // Contraseña temporal que debe cambiarse

					// Configurar variables de entorno de respaldo
					userEnvVars["DATABASE_URL"] = fmt.Sprintf("postgresql://%s:%s@localhost:5432/%s?schema=public",
						dbUser, dbPassword, dbName)
					userEnvVars["DB_CONNECTION"] = "pgsql"
					userEnvVars["DB_HOST"] = "localhost"
					userEnvVars["DB_PORT"] = "5432"
					userEnvVars["DB_DATABASE"] = dbName
					userEnvVars["DB_USERNAME"] = dbUser
					userEnvVars["DB_PASSWORD"] = dbPassword

					// Agregar comentario al principio del archivo .env
					userEnvVars["# IMPORTANTE"] = "La configuración de PostgreSQL no pudo completarse automáticamente."
					userEnvVars["# POR FAVOR"] = "Modifique las credenciales de la base de datos manualmente."
				} else {
					// Agregar variables de entorno de PostgreSQL
					for key, value := range pgEnvVars {
						userEnvVars[key] = value
					}
				}
			case "mysql":
				// Código existente para MySQL
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
					userEnvVars["DB_CONNECTION"] = "mysql"
					userEnvVars["DB_HOST"] = "localhost"
					userEnvVars["DB_PORT"] = "3306"
					userEnvVars["DB_DATABASE"] = dbName
					userEnvVars["DB_USERNAME"] = dbUser
					userEnvVars["DB_PASSWORD"] = dbPassword
				}
			default:
				fmt.Printf("Tipo de base de datos no soportado: %s\n", projectInfo.DBType)
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
			userEnvVars["PORT"] = fmt.Sprintf("%d", port) // Asegurarse de que el puerto se establece correctamente
		case utils.FrameworkNextJS:
			userEnvVars["NODE_ENV"] = "production"
			userEnvVars["PORT"] = fmt.Sprintf("%d", port) // Asegurarse de que el puerto se establece correctamente

			// Configurar variables específicas para NextJS
			if _, ok := projectInfo.EnvVars["NEXT_PUBLIC_API_URL"]; ok {
				// Si estamos en un subdominio, la API podría estar en otro subdominio
				if opts.IsSubdomain {
					apiSubdomain := "api." + opts.ParentDomain
					userEnvVars["NEXT_PUBLIC_API_URL"] = "https://" + apiSubdomain
				} else {
					userEnvVars["NEXT_PUBLIC_API_URL"] = "https://api." + opts.Domain
				}
			}

			// Configurar otras variables comunes de NextJS si existen en .env.example
			nextjsCommonVars := []string{
				"NEXT_PUBLIC_IMAGE_DOMAINS",
				"NEXT_PUBLIC_IMAGES_URL",
				"NEXT_PUBLIC_BODY_SIZE_LIMIT",
			}

			for _, varName := range nextjsCommonVars {
				if value, ok := projectInfo.EnvVars[varName]; ok {
					userEnvVars[varName] = value
				}
			}

			// Asegurar que PWA está habilitada en producción
			userEnvVars["NEXT_PUBLIC_PWA_ENABLED"] = "true"
		case utils.FrameworkExpress:
			userEnvVars["NODE_ENV"] = "production"
			userEnvVars["PORT"] = fmt.Sprintf("%d", port) // Asegurarse de que el puerto se establece correctamente
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
			pgCheckCmd := exec.Command("sudo", "-u", "postgres", "pg_isready")
			if pgOutput, pgErr := pgCheckCmd.CombinedOutput(); pgErr != nil {
				fmt.Printf("Advertencia: PostgreSQL no está disponible: %v\n%s\n", pgErr, pgOutput)
				fmt.Println("Se omitirán las operaciones de Prisma. Por favor, configure PostgreSQL manualmente y ejecute las migraciones después.")
			} else {
				// Verificar que DATABASE_URL existe en el archivo .env antes de continuar
				envFilePath := filepath.Join(opts.AppDir, ".env")
				if envContent, err := os.ReadFile(envFilePath); err == nil {
					envLines := strings.Split(string(envContent), "\n")
					hasDbUrl := false

					for _, line := range envLines {
						if strings.HasPrefix(line, "DATABASE_URL=") {
							hasDbUrl = true
							break
						}
					}

					if !hasDbUrl {
						fmt.Println("No se encontró DATABASE_URL en el archivo .env. Se omitirán las operaciones de Prisma.")
						return nil // Return nil to indicate no error
					}
				}

				// Ejecutar comandos de Prisma directamente sin instalar globalmente
				// Instalar @prisma/client y generar el cliente
				generateCmd := "npm install @prisma/client && npx prisma generate"
				cmd := exec.Command("su", "-c", generateCmd, opts.User)
				cmd.Dir = opts.AppDir
				fmt.Printf("Ejecutando: %s\n", generateCmd)
				if output, err := cmd.CombinedOutput(); err != nil {
					fmt.Printf("Error al ejecutar prisma generate: %v\n%s\n", err, output)
					// Intentar una alternativa
					alternativeCmd := "cd " + opts.AppDir + " && npm install @prisma/client && npx prisma generate"
					altCmd := exec.Command("su", "-c", alternativeCmd, opts.User)
					fmt.Printf("Intentando comando alternativo: %s\n", alternativeCmd)
					if altOutput, altErr := altCmd.CombinedOutput(); altErr != nil {
						fmt.Printf("Error con el comando alternativo: %v\n%s\n", altErr, altOutput)
					} else {
						fmt.Printf("Comando alternativo exitoso: %s\n", altOutput)
					}
				} else {
					fmt.Printf("Prisma generate completado: %s\n", output)
				}

				// Verificar si existen migraciones antes de intentar ejecutarlas
				migrationsPath := filepath.Join(opts.AppDir, "prisma", "migrations")
				if _, err := os.Stat(migrationsPath); err == nil {
					// Ejecutar prisma migrate deploy
					migrateCmd := "npx prisma migrate deploy"
					cmd = exec.Command("su", "-c", migrateCmd, opts.User)
					cmd.Dir = opts.AppDir
					fmt.Printf("Ejecutando: %s\n", migrateCmd)
					if output, err := cmd.CombinedOutput(); err != nil {
						fmt.Printf("Advertencia: error en migraciones de Prisma (no crítico): %v\n%s\n", err, output)
					} else {
						fmt.Printf("Migraciones de Prisma completadas: %s\n", output)
					}
				} else {
					fmt.Println("No se encontraron migraciones de Prisma. Omitiendo prisma migrate deploy.")
				}
			}
		}
	}

	// Ejecutar comandos como el usuario del sitio
	commands := []string{
		// Instalar dependencias con fallback options
		"npm ci || npm install || npm install --legacy-peer-deps || npm install --force",
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
			// Para npm install, intentar con diferentes opciones si hay errores
			if strings.Contains(cmdStr, "npm") && strings.Contains(cmdStr, "install") {
				fmt.Printf("Advertencia: hubo algunos errores en la instalación de dependencias, intentando con opciones alternativas...\n")

				// Intentar con --legacy-peer-deps
				legacyCmd := exec.Command("su", "-c", "npm install --legacy-peer-deps", opts.User)
				legacyCmd.Dir = opts.AppDir
				if legacyOutput, legacyErr := legacyCmd.CombinedOutput(); legacyErr == nil {
					fmt.Printf("Instalación exitosa con --legacy-peer-deps\n")
					continue
				} else {
					fmt.Printf("Error con --legacy-peer-deps: %v\n%s\n", legacyErr, legacyOutput)
				}

				// Intentar con --force
				forceCmd := exec.Command("su", "-c", "npm install --force", opts.User)
				forceCmd.Dir = opts.AppDir
				if forceOutput, forceErr := forceCmd.CombinedOutput(); forceErr == nil {
					fmt.Printf("Instalación exitosa con --force\n")
					continue
				} else {
					fmt.Printf("Error con --force: %v\n%s\n", forceErr, forceOutput)
				}

				// Si todos los intentos fallan, mostrar advertencia pero continuar
				fmt.Printf("Advertencia: no se pudo instalar las dependencias correctamente, pero continuaremos: %v\n", err)
			} else {
				return fmt.Errorf("error al ejecutar comando '%s': %v\n%s", cmdStr, err, output)
			}
		}
	}

	// Determinar comando para iniciar la aplicación
	startCommand := utils.GetNodeJSStartCommand(projectInfo)

	// Para frameworks específicos, configurar comando de inicio
	switch projectInfo.Framework {
	case utils.FrameworkNestJS:
		if projectInfo.HasTypeScript {
			// Para NestJS con TypeScript, usar la ruta correcta o npm run start
			// Verificar si existe el script start en package.json
			packageJSONPath := filepath.Join(opts.AppDir, "package.json")
			if _, err := os.Stat(packageJSONPath); err == nil {
				// Leer package.json para verificar si tiene script start
				packageJSONContent, err := os.ReadFile(packageJSONPath)
				if err == nil {
					packageJSONStr := string(packageJSONContent)
					if strings.Contains(packageJSONStr, "\"start\":") {
						// Usar npm run start si está disponible
						startCommand = "npm run start"
					} else {
						// Usar la ruta correcta al archivo main.js
						startCommand = "node dist/src/main.js"
					}
				} else {
					// Fallback a la ruta correcta
					startCommand = "node dist/src/main.js"
				}
			} else {
				// Fallback a la ruta correcta
				startCommand = "node dist/src/main.js"
			}
		}
	case utils.FrameworkNextJS:
		// Para Next.js, necesitamos un puerto específico
		startCommand = fmt.Sprintf("npm run start -- -p %d", port)
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
		return fmt.Errorf("error al iniciar la aplicación con PM2: %v\n%s", err, output)
	}

	fmt.Printf("Aplicación iniciada correctamente en el puerto %d\n", port)
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
		// Para subdominios, leer el puerto de la configuración de Nginx
		nginxConfPath := filepath.Join(opts.HomeDir, "nginx", fmt.Sprintf("%s.conf", opts.Domain))
		if _, err := os.Stat(nginxConfPath); err == nil {
			// Leer el archivo
			confData, err := os.ReadFile(nginxConfPath)
			if err != nil {
				return fmt.Errorf("error al leer configuración Nginx: %v", err)
			}

			// Buscar el puerto en la configuración
			confStr := string(confData)
			if strings.Contains(confStr, "proxy_pass http://localhost:") {
				portStr := strings.Split(strings.Split(confStr, "proxy_pass http://localhost:")[1], ";")[0]
				if p, err := strconv.Atoi(portStr); err == nil {
					port = p
					fmt.Printf("Usando puerto %d de la configuración de Nginx\n", port)
				}
			}
		} else {
			// Si no se encuentra la configuración, usar un puerto basado en hash
			h := fnv.New32a()
			h.Write([]byte(opts.Domain))
			port = 3001 + int(h.Sum32()%999) // Puertos entre 3001 y 3999
			fmt.Printf("Generando puerto aleatorio: %d\n", port)
		}
	}

	// Determinar comando para iniciar la aplicación
	startCommand := utils.GetNodeJSStartCommand(projectInfo)

	// Para frameworks específicos, configurar comando de inicio
	switch projectInfo.Framework {
	case utils.FrameworkNestJS:
		if projectInfo.HasTypeScript {
			// Para NestJS con TypeScript, usar la ruta correcta o npm run start
			// Verificar si existe el script start en package.json
			packageJSONPath := filepath.Join(opts.AppDir, "package.json")
			if _, err := os.Stat(packageJSONPath); err == nil {
				// Leer package.json para verificar si tiene script start
				packageJSONContent, err := os.ReadFile(packageJSONPath)
				if err == nil {
					packageJSONStr := string(packageJSONContent)
					if strings.Contains(packageJSONStr, "\"start\":") {
						// Usar npm run start si está disponible
						startCommand = "npm run start"
					} else {
						// Usar la ruta correcta al archivo main.js
						startCommand = "node dist/src/main.js"
					}
				} else {
					// Fallback a la ruta correcta
					startCommand = "node dist/src/main.js"
				}
			} else {
				// Fallback a la ruta correcta
				startCommand = "node dist/src/main.js"
			}
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
		return fmt.Errorf("error al iniciar la aplicación con PM2: %v\n%s", err, output)
	}

	fmt.Printf("PM2 reconfigurado correctamente para %s\n", opts.Domain)
	return nil
}

// Función corregida para manejar la autenticación PostgreSQL correctamente

func setupPostgreSQLDatabase(opts *DeployOptions, projectInfo *utils.NodeJSProjectInfo) (map[string]string, error) {
	// Variables de entorno que se devolverán
	envVars := make(map[string]string)

	// Verificar si PostgreSQL está disponible
	pgCheckCmd := exec.Command("sudo", "-u", "postgres", "pg_isready")
	pgOutput, pgErr := pgCheckCmd.CombinedOutput()

	if pgErr != nil {
		fmt.Printf("PostgreSQL no está disponible: %v\n%s\n", pgErr, pgOutput)
		fmt.Println("¿Desea configurar PostgreSQL manualmente? (s/n)")

		var response string
		fmt.Scanln(&response)

		if strings.ToLower(response) == "s" {
			// Solicitar datos de configuración manual
			// [Código de configuración manual existente...]
			// ...
			return envVars, nil
		}

		// Si el usuario no desea configurar manualmente, devolver error
		return nil, fmt.Errorf("PostgreSQL no está disponible y se omitió la configuración manual")
	}

	// PostgreSQL está disponible, configurar automáticamente

	// Sanitizar nombre de dominio para usarlo como nombre de base de datos
	dbName := strings.ReplaceAll(opts.Domain, ".", "_")
	// Usar el usuario de sistema como usuario de base de datos
	dbUser := opts.User
	// Generar contraseña aleatoria
	dbPassword := generateRandomPassword(16)

	// Verificar si el usuario ya existe en PostgreSQL
	checkUserCmd := fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname='%s'", dbUser)
	cmd := exec.Command("sudo", "-u", "postgres", "psql", "-tAc", checkUserCmd)
	userOutput, userErr := cmd.CombinedOutput()

	userExists := false
	if userErr == nil && strings.TrimSpace(string(userOutput)) == "1" {
		userExists = true
		fmt.Printf("El usuario PostgreSQL '%s' ya existe\n", dbUser)
	}

	if !userExists {
		// Crear usuario de PostgreSQL
		createUserCmd := fmt.Sprintf("CREATE USER %s WITH ENCRYPTED PASSWORD '%s';", dbUser, dbPassword)
		cmd = exec.Command("sudo", "-u", "postgres", "psql", "-c", createUserCmd)

		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("Error al crear usuario PostgreSQL: %v\n%s\n", err, output)
			return nil, fmt.Errorf("error al crear usuario PostgreSQL: %v", err)
		}

		fmt.Printf("Usuario PostgreSQL '%s' creado exitosamente\n", dbUser)
	} else {
		// Si el usuario ya existe, cambiar su contraseña
		alterUserCmd := fmt.Sprintf("ALTER USER %s WITH ENCRYPTED PASSWORD '%s';", dbUser, dbPassword)
		cmd = exec.Command("sudo", "-u", "postgres", "psql", "-c", alterUserCmd)

		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("Error al cambiar contraseña del usuario PostgreSQL: %v\n%s\n", err, output)
			return nil, fmt.Errorf("error al cambiar contraseña del usuario PostgreSQL: %v", err)
		}

		fmt.Printf("Contraseña actualizada para el usuario PostgreSQL '%s'\n", dbUser)
	}

	// Verificar si la base de datos ya existe
	checkDbCmd := fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname='%s'", dbName)
	cmd = exec.Command("sudo", "-u", "postgres", "psql", "-tAc", checkDbCmd)
	dbOutput, dbErr := cmd.CombinedOutput()

	dbExists := false
	if dbErr == nil && strings.TrimSpace(string(dbOutput)) == "1" {
		dbExists = true
		fmt.Printf("La base de datos PostgreSQL '%s' ya existe\n", dbName)
	}

	if !dbExists {
		// Crear base de datos
		createDbCmd := fmt.Sprintf("CREATE DATABASE %s OWNER %s;", dbName, dbUser)
		cmd = exec.Command("sudo", "-u", "postgres", "psql", "-c", createDbCmd)

		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("Error al crear base de datos PostgreSQL: %v\n%s\n", err, output)
			return nil, fmt.Errorf("error al crear base de datos PostgreSQL: %v", err)
		}

		fmt.Printf("Base de datos PostgreSQL '%s' creada exitosamente\n", dbName)
	} else {
		// Si la base de datos ya existe, asignar permisos al usuario
		grantCmd := fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s;", dbName, dbUser)
		cmd = exec.Command("sudo", "-u", "postgres", "psql", "-c", grantCmd)

		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("Error al asignar permisos sobre la base de datos: %v\n%s\n", err, output)
			// No retornamos error aquí, continuamos con la configuración
		}
	}

	// Construir URL de conexión para diferentes formatos según el framework
	databaseURL := fmt.Sprintf("postgresql://%s:%s@localhost:5432/%s", dbUser, dbPassword, dbName)

	// Configurar variables de entorno según el framework detectado
	envVars["DATABASE_URL"] = databaseURL

	// Variables para Laravel
	envVars["DB_CONNECTION"] = "pgsql"
	envVars["DB_HOST"] = "localhost"
	envVars["DB_PORT"] = "5432"
	envVars["DB_DATABASE"] = dbName
	envVars["DB_USERNAME"] = dbUser
	envVars["DB_PASSWORD"] = dbPassword

	// Variables específicas para Prisma
	if projectInfo.HasPrisma {
		// Prisma usa DATABASE_URL en este formato
		envVars["DATABASE_URL"] = databaseURL + "?schema=public"
	}

	// Mostrar información de conexión
	fmt.Println("\n=== Configuración de PostgreSQL ===")
	fmt.Printf("Base de datos: %s\n", dbName)
	fmt.Printf("Usuario: %s\n", dbUser)
	fmt.Printf("Contraseña: %s\n", dbPassword)
	fmt.Printf("URL de conexión: %s\n", databaseURL)
	fmt.Println("===================================\n")

	return envVars, nil
}

// removeDeployedProject elimina un proyecto desplegado
func removeDeployedProject(opts *DeployOptions) error {
	fmt.Printf("Eliminando proyecto desplegado en %s...\n", opts.Domain)

	// Detener la aplicación en PM2
	fmt.Println("Deteniendo aplicación en PM2...")
	stopCmd := exec.Command("sudo", "-u", opts.User, "pm2", "stop", opts.Domain)
	if output, err := stopCmd.CombinedOutput(); err != nil {
		fmt.Printf("Advertencia: error al detener aplicación en PM2: %v\n%s\n", err, output)
		// Continuamos aunque haya error al detener
	}

	// Eliminar la aplicación de PM2
	fmt.Println("Eliminando aplicación de PM2...")
	deleteCmd := exec.Command("sudo", "-u", opts.User, "pm2", "delete", opts.Domain)
	if output, err := deleteCmd.CombinedOutput(); err != nil {
		fmt.Printf("Advertencia: error al eliminar aplicación de PM2: %v\n%s\n", err, output)
		// Continuamos aunque haya error al eliminar
	}

	// Guardar la configuración de PM2 para que persista después de reiniciar
	saveCmd := exec.Command("sudo", "-u", opts.User, "pm2", "save")
	if output, err := saveCmd.CombinedOutput(); err != nil {
		fmt.Printf("Advertencia: error al guardar configuración PM2: %v\n%s\n", err, output)
		// Continuamos aunque haya error al guardar
	}

	// Detener cualquier proceso de Node.js que esté ejecutándose en el directorio de la aplicación
	fmt.Println("Deteniendo procesos de Node.js...")
	killCmd := exec.Command("pkill", "-f", fmt.Sprintf("node.*%s", opts.AppDir))
	if output, err := killCmd.CombinedOutput(); err != nil {
		fmt.Printf("Advertencia: error al detener procesos de Node.js: %v\n%s\n", err, output)
		// Continuamos aunque haya error al detener
	}

	// Verificar si hay procesos de Node.js ejecutándose en el puerto 3000
	fmt.Println("Verificando procesos en el puerto 3000...")
	lsofCmd := exec.Command("lsof", "-i", ":3000", "-t")
	if output, err := lsofCmd.CombinedOutput(); err == nil && len(output) > 0 {
		// Si hay procesos, intentar detenerlos
		fmt.Println("Deteniendo procesos en el puerto 3000...")
		killPortCmd := exec.Command("kill", "-9", strings.TrimSpace(string(output)))
		if output, err := killPortCmd.CombinedOutput(); err != nil {
			fmt.Printf("Advertencia: error al detener procesos en el puerto 3000: %v\n%s\n", err, output)
		}
	}

	// Hacer backup si se solicita
	if opts.Backup {
		fmt.Println("Haciendo backup de la carpeta del proyecto...")
		backupDir := filepath.Join(opts.HomeDir, "backups")
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return fmt.Errorf("error al crear directorio de backups: %v", err)
		}

		// Cambiar propietario del directorio de backups
		chownCmd := exec.Command("chown", fmt.Sprintf("%s:%s", opts.User, opts.User), backupDir)
		if output, err := chownCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al cambiar propietario del directorio de backups: %v\n%s", err, output)
		}

		// Crear nombre de archivo de backup con timestamp
		timestamp := time.Now().Format("20060102150405")
		backupFileName := fmt.Sprintf("%s_%s.tar.gz", opts.Domain, timestamp)
		backupPath := filepath.Join(backupDir, backupFileName)

		// Crear archivo de backup
		tarCmd := exec.Command("tar", "-czf", backupPath, "-C", filepath.Dir(opts.AppDir), filepath.Base(opts.AppDir))
		if output, err := tarCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al crear backup: %v\n%s", err, output)
		}

		// Cambiar propietario del archivo de backup
		chownCmd = exec.Command("chown", fmt.Sprintf("%s:%s", opts.User, opts.User), backupPath)
		if output, err := chownCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al cambiar propietario del archivo de backup: %v\n%s", err, output)
		}

		fmt.Printf("Backup creado en %s\n", backupPath)
	}

	// Eliminar la carpeta del proyecto
	fmt.Printf("Eliminando carpeta del proyecto en %s...\n", opts.AppDir)
	if err := os.RemoveAll(opts.AppDir); err != nil {
		return fmt.Errorf("error al eliminar carpeta del proyecto: %v", err)
	}

	fmt.Printf("Proyecto eliminado correctamente: %s\n", opts.Domain)
	return nil
}
