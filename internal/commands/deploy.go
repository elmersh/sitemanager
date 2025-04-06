package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/elmersh/sitemanager/internal/config"
	"github.com/spf13/cobra"
)

// DeployOptions contiene las opciones para el comando deploy
type DeployOptions struct {
	Domain      string
	Repository  string
	Branch      string
	Type        string
	Environment string
	User        string
	HomeDir     string
	AppDir      string
}

// AddDeployCommand agrega el comando deploy al comando raíz
func AddDeployCommand(rootCmd *cobra.Command, cfg *config.Config) {
	// Opciones del comando
	var opts DeployOptions

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

			// Configurar usuario y directorios
			opts.User = strings.Split(opts.Domain, ".")[0]
			opts.HomeDir = filepath.Join("/home", opts.Domain)
			opts.AppDir = filepath.Join(opts.HomeDir, "app")

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

	// Marcar flags obligatorios
	deployCmd.MarkFlagRequired("domain")
	deployCmd.MarkFlagRequired("repo")

	// Agregar comando al comando raíz
	rootCmd.AddCommand(deployCmd)
}

// cloneRepository clona el repositorio Git
func cloneRepository(opts *DeployOptions) error {
	// Verificar si el directorio app ya existe
	if _, err := os.Stat(opts.AppDir); err == nil {
		// Eliminar directorio existente
		fmt.Printf("Eliminando directorio app existente...\n")
		if err := os.RemoveAll(opts.AppDir); err != nil {
			return fmt.Errorf("error al eliminar directorio app: %v", err)
		}
	}

	// Clonar repositorio
	fmt.Printf("Clonando repositorio %s en %s...\n", opts.Repository, opts.AppDir)
	cmd := exec.Command(
		"git", "clone",
		"--branch", opts.Branch,
		"--single-branch",
		opts.Repository,
		opts.AppDir,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al clonar repositorio: %v\n%s", err, output)
	}

	// Cambiar propietario del directorio app
	cmd = exec.Command("chown", "-R", fmt.Sprintf("%s:%s", opts.User, opts.User), opts.AppDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario: %v\n%s", err, output)
	}

	fmt.Printf("Repositorio clonado correctamente en %s\n", opts.AppDir)
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

	// Ejecutar comandos como el usuario del sitio
	commands := []string{
		// Instalar dependencias
		"composer install --no-dev --optimize-autoloader",
		// Copiar .env.example a .env si no existe
		"[ -f .env ] || cp .env.example .env",
		// Generar clave de aplicación
		"php artisan key:generate",
		// Ejecutar migraciones
		"php artisan migrate --force",
		// Optimizaciones
		"php artisan config:cache",
		"php artisan route:cache",
		"php artisan view:cache",
	}

	for _, cmdStr := range commands {
		cmd := exec.Command("su", "-c", cmdStr, opts.User)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al ejecutar comando '%s': %v\n%s", cmdStr, err, output)
		}
	}

	// Crear enlace simbólico de public_html a app/public
	publicHtmlPath := filepath.Join(opts.HomeDir, "public_html")
	appPublicPath := filepath.Join(opts.AppDir, "public")

	// Eliminar public_html existente
	os.RemoveAll(publicHtmlPath)

	// Crear enlace simbólico
	if err := os.Symlink(appPublicPath, publicHtmlPath); err != nil {
		return fmt.Errorf("error al crear enlace simbólico: %v", err)
	}

	// Cambiar propietario del enlace
	cmd := exec.Command("chown", "-h", fmt.Sprintf("%s:%s", opts.User, opts.User), publicHtmlPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario del enlace: %v\n%s", err, output)
	}

	fmt.Printf("Aplicación Laravel desplegada correctamente en %s\n", opts.Domain)
	return nil
}

// deployNodejs configura y despliega una aplicación Node.js
func deployNodejs(opts *DeployOptions) error {
	fmt.Printf("Desplegando aplicación Node.js en %s...\n", opts.Domain)

	// Cambiar al directorio de la aplicación
	if err := os.Chdir(opts.AppDir); err != nil {
		return fmt.Errorf("error al cambiar al directorio de la aplicación: %v", err)
	}

	// Ejecutar comandos como el usuario del sitio
	commands := []string{
		// Instalar dependencias
		"npm install --production",
		// Construir aplicación si hay un script de build
		"npm run build || echo 'No build script found'",
	}

	for _, cmdStr := range commands {
		cmd := exec.Command("su", "-c", cmdStr, opts.User)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al ejecutar comando '%s': %v\n%s", cmdStr, err, output)
		}
	}

	// Crear archivo de configuración para PM2
	pmConfig := fmt.Sprintf(`{
  "apps": [{
    "name": "%s",
    "script": "%s/app.js",
    "cwd": "%s",
    "env": {
      "NODE_ENV": "%s",
      "PORT": "3000"
    }
  }]
}`, opts.Domain, opts.AppDir, opts.AppDir, opts.Environment)

	configPath := filepath.Join(opts.HomeDir, "pm2.config.json")
	if err := os.WriteFile(configPath, []byte(pmConfig), 0644); err != nil {
		return fmt.Errorf("error al crear archivo de configuración PM2: %v", err)
	}

	// Cambiar propietario del archivo de configuración
	cmd := exec.Command("chown", fmt.Sprintf("%s:%s", opts.User, opts.User), configPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario: %v\n%s", err, output)
	}

	// Iniciar aplicación con PM2
	startCmd := exec.Command(
		"pm2", "start", configPath,
		"--uid", opts.User,
		"--gid", opts.User,
	)

	if output, err := startCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al iniciar aplicación con PM2: %v\n%s", err, output)
	}

	// Guardar configuración de PM2
	saveCmd := exec.Command("pm2", "save")
	if output, err := saveCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al guardar configuración PM2: %v\n%s", err, output)
	}

	fmt.Printf("Aplicación Node.js desplegada correctamente en %s\n", opts.Domain)
	return nil
}
