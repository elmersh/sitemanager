package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// InitSkelDir inicializa el directorio skel con la estructura básica necesaria
func InitSkelDir(skelDir string) error {
	// Verificar si el directorio skel ya existe
	if _, err := os.Stat(skelDir); err == nil {
		fmt.Printf("El directorio skel ya existe: %s\n", skelDir)
	} else {
		// Crear el directorio skel
		if err := os.MkdirAll(skelDir, 0755); err != nil {
			return fmt.Errorf("error al crear el directorio skel: %v", err)
		}
		fmt.Printf("Directorio skel creado en %s\n", skelDir)
	}

	// Verificar si el directorio skel tiene contenido
	entries, err := os.ReadDir(skelDir)
	if err != nil {
		return fmt.Errorf("error al leer el directorio skel: %v", err)
	}

	// Si el directorio está vacío o no tiene la estructura básica, crearla
	if len(entries) == 0 || !hasBasicStructure(skelDir) {
		fmt.Println("Inicializando estructura básica del directorio skel...")

		// Crear la estructura básica del directorio skel
		dirs := []string{
			"public_html",
			"nginx",
			"logs",
			"apps",
		}

		for _, dir := range dirs {
			path := filepath.Join(skelDir, dir)
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("error al crear el directorio %s: %v", path, err)
			}
		}

		// Crear index.html de prueba
		indexFile := filepath.Join(skelDir, "public_html", "index.html")
		indexContent := `<html><body><h1>Bienvenido a tu sitio</h1><p>Sitio configurado con SiteManager</p></body></html>`
		if err := os.WriteFile(indexFile, []byte(indexContent), 0644); err != nil {
			return fmt.Errorf("error al crear el archivo index.html: %v", err)
		}

		// Establecer permisos
		cmd := exec.Command("chmod", "-R", "755", skelDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al establecer permisos: %v\n%s", err, output)
		}
	}

	fmt.Printf("Directorio skel inicializado correctamente en %s\n", skelDir)
	return nil
}

// hasBasicStructure verifica si el directorio skel tiene la estructura básica necesaria
func hasBasicStructure(skelDir string) bool {
	requiredDirs := []string{
		"public_html",
		"nginx",
		"logs",
		"apps",
	}

	for _, dir := range requiredDirs {
		path := filepath.Join(skelDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return false
		}
	}

	// Verificar si existe el archivo index.html
	indexFile := filepath.Join(skelDir, "public_html", "index.html")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		return false
	}

	return true
}
