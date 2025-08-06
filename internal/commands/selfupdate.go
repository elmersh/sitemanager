package commands

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/elmersh/sitemanager/internal/config"
	"github.com/spf13/cobra"
)

const (
	githubAPI      = "https://api.github.com/repos/elmersh/sitemanager/releases/latest"
	downloadURL    = "https://github.com/elmersh/sitemanager/releases/download/%s/sitemanager-%s-%s-%s.tar.gz"
	updateCheckURL = "https://api.github.com/repos/elmersh/sitemanager/releases"
)

type GitHubRelease struct {
	TagName    string    `json:"tag_name"`
	Name       string    `json:"name"`
	Body       string    `json:"body"`
	Draft      bool      `json:"draft"`
	Prerelease bool      `json:"prerelease"`
	CreatedAt  time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
	Assets     []GitHubAsset `json:"assets"`
}

type GitHubAsset struct {
	Name               string `json:"name"`
	Size               int    `json:"size"`
	DownloadCount      int    `json:"download_count"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Variables inyectadas desde main
var (
	Version   string
	BuildTime string
	GitCommit string
)

// SetVersionInfo establece la información de versión desde main
func SetVersionInfo(version, buildTime, gitCommit string) {
	Version = version
	BuildTime = buildTime
	GitCommit = gitCommit
}

// AddSelfUpdateCommand agrega el comando self-update al root command
func AddSelfUpdateCommand(rootCmd *cobra.Command, cfg *config.Config) {
	selfUpdateCmd := &cobra.Command{
		Use:   "self-update",
		Short: "Actualiza SiteManager a la última versión",
		Long: `Descarga e instala automáticamente la última versión de SiteManager desde GitHub.
Este comando requiere permisos de sudo y conexión a internet.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSelfUpdate()
		},
	}

	// Comando para verificar actualizaciones
	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Verificar si hay actualizaciones disponibles",
		Long: `Verifica si hay una versión más reciente de SiteManager disponible
sin instalarla. Muestra información sobre la nueva versión si está disponible.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return checkForUpdates(false)
		},
	}

	selfUpdateCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(selfUpdateCmd)

	// También agregar como subcomando de version
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Mostrar información de versión",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("SiteManager %s\n", Version)
			fmt.Printf("Compilado: %s\n", BuildTime)
			fmt.Printf("Git commit: %s\n", GitCommit)
			fmt.Printf("Go: %s\n", runtime.Version())
			fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}

	checkVersionCmd := &cobra.Command{
		Use:   "check",
		Short: "Verificar actualizaciones disponibles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return checkForUpdates(true)
		},
	}

	versionCmd.AddCommand(checkVersionCmd)
	rootCmd.AddCommand(versionCmd)
}

func runSelfUpdate() error {
	fmt.Println("🔄 Verificando actualizaciones...")

	// Verificar permisos de sudo
	if os.Geteuid() != 0 {
		return fmt.Errorf("❌ se requieren permisos de sudo para actualizar")
	}

	// Obtener la información de la última release
	release, err := getLatestRelease()
	if err != nil {
		return fmt.Errorf("❌ error al obtener información de la última versión: %v", err)
	}

	currentVersion := strings.TrimPrefix(Version, "v")
	latestVersion := strings.TrimPrefix(release.TagName, "v")

	if currentVersion == latestVersion {
		fmt.Printf("✅ Ya tienes la última versión (%s)\n", currentVersion)
		return nil
	}

	fmt.Printf("📦 Nueva versión disponible: %s -> %s\n", currentVersion, latestVersion)
	fmt.Printf("📅 Fecha de lanzamiento: %s\n", release.PublishedAt.Format("2006-01-02 15:04:05"))

	if release.Body != "" && len(release.Body) > 0 {
		fmt.Println("\n📋 Notas de la versión:")
		// Mostrar solo las primeras 5 líneas del changelog
		lines := strings.Split(release.Body, "\n")
		maxLines := 5
		if len(lines) < maxLines {
			maxLines = len(lines)
		}
		for i := 0; i < maxLines; i++ {
			if strings.TrimSpace(lines[i]) != "" {
				fmt.Printf("  %s\n", lines[i])
			}
		}
		if len(lines) > maxLines {
			fmt.Println("  ...")
		}
	}

	fmt.Print("\n¿Deseas continuar con la actualización? [y/N]: ")
	var response string
	fmt.Scanln(&response)
	
	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println("❌ Actualización cancelada")
		return nil
	}

	// Descargar y actualizar
	return downloadAndUpdate(release)
}

func checkForUpdates(verbose bool) error {
	if verbose {
		fmt.Println("🔍 Verificando actualizaciones...")
	}

	release, err := getLatestRelease()
	if err != nil {
		return fmt.Errorf("❌ error al verificar actualizaciones: %v", err)
	}

	currentVersion := strings.TrimPrefix(Version, "v")
	latestVersion := strings.TrimPrefix(release.TagName, "v")

	if verbose {
		fmt.Printf("📋 Versión actual: %s\n", currentVersion)
		fmt.Printf("📋 Versión disponible: %s\n", latestVersion)
	}

	if currentVersion == latestVersion {
		fmt.Printf("✅ Tienes la última versión (%s)\n", currentVersion)
	} else {
		fmt.Printf("🚀 Nueva versión disponible: %s\n", latestVersion)
		fmt.Printf("📅 Fecha: %s\n", release.PublishedAt.Format("2006-01-02"))
		if verbose {
			fmt.Println("\n📋 Para actualizar ejecuta:")
			fmt.Println("  sudo sm self-update")
		}
	}

	return nil
}

func getLatestRelease() (*GitHubRelease, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	req, err := http.NewRequest("GET", githubAPI, nil)
	if err != nil {
		return nil, err
	}

	// Agregar User-Agent
	req.Header.Set("User-Agent", fmt.Sprintf("SiteManager/%s", Version))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API respondió con código %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func downloadAndUpdate(release *GitHubRelease) error {
	// Determinar el archivo a descargar
	arch := runtime.GOARCH
	goos := runtime.GOOS
	version := strings.TrimPrefix(release.TagName, "v")
	
	fileName := fmt.Sprintf("sitemanager-%s-%s-%s.tar.gz", version, goos, arch)
	downloadURL := fmt.Sprintf("https://github.com/elmersh/sitemanager/releases/download/%s/%s", release.TagName, fileName)

	fmt.Printf("📥 Descargando %s...\n", fileName)

	// Crear directorio temporal
	tmpDir := filepath.Join(os.TempDir(), "sitemanager-update")
	if err := os.RemoveAll(tmpDir); err != nil {
		return fmt.Errorf("error al limpiar directorio temporal: %v", err)
	}
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("error al crear directorio temporal: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Descargar archivo
	tarPath := filepath.Join(tmpDir, fileName)
	if err := downloadFile(downloadURL, tarPath); err != nil {
		return fmt.Errorf("error al descargar: %v", err)
	}

	// Extraer archivo
	fmt.Println("📦 Extrayendo archivo...")
	extractPath := filepath.Join(tmpDir, "extracted")
	if err := extractTarGz(tarPath, extractPath); err != nil {
		return fmt.Errorf("error al extraer: %v", err)
	}

	// Buscar el binario extraído
	var newBinaryPath string
	err := filepath.Walk(extractPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "sm" && !info.IsDir() {
			newBinaryPath = path
			return filepath.SkipDir
		}
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("error al buscar binario: %v", err)
	}
	
	if newBinaryPath == "" {
		return fmt.Errorf("no se encontró el binario sm en el archivo descargado")
	}

	// Obtener ruta actual del binario
	currentBinaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error al obtener ruta del binario actual: %v", err)
	}

	// Hacer backup del binario actual
	backupPath := currentBinaryPath + ".backup"
	fmt.Println("💾 Creando backup del binario actual...")
	if err := copyFile(currentBinaryPath, backupPath); err != nil {
		return fmt.Errorf("error al crear backup: %v", err)
	}

	// Reemplazar binario
	fmt.Println("🔄 Actualizando binario...")
	if err := copyFile(newBinaryPath, currentBinaryPath); err != nil {
		// Restaurar backup si falla
		copyFile(backupPath, currentBinaryPath)
		return fmt.Errorf("error al actualizar binario: %v", err)
	}

	// Dar permisos de ejecución
	if err := os.Chmod(currentBinaryPath, 0755); err != nil {
		return fmt.Errorf("error al configurar permisos: %v", err)
	}

	// Limpiar backup
	os.Remove(backupPath)

	fmt.Printf("✅ SiteManager actualizado exitosamente a la versión %s\n", release.TagName)
	fmt.Println("🔄 Reinicia tu terminal para usar la nueva versión")
	
	return nil
}

func downloadFile(url, filepath string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("SiteManager/%s", Version))

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error de descarga: código HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractTarGz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()

			if err := os.Chmod(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}