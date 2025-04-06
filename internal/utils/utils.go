// internal/utils/utils.go
package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// ReadTemplateFile lee el contenido de un archivo de plantilla
func ReadTemplateFile(tmplPath string) (string, error) {
	// Buscar el archivo en varias ubicaciones posibles
	possiblePaths := []string{
		tmplPath,
		filepath.Join("/etc/sitemanager", tmplPath),
		filepath.Join("/usr/local/share/sitemanager", tmplPath),
		filepath.Join("/usr/share/sitemanager", tmplPath),
	}

	// Buscar en el directorio actual
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		possiblePaths = append(possiblePaths, filepath.Join(execDir, tmplPath))
	}

	// Buscar en las posibles ubicaciones
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			content, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("error al leer plantilla %s: %v", path, err)
			}
			return string(content), nil
		}
	}

	return "", fmt.Errorf("plantilla %s no encontrada", tmplPath)
}

// PathExists verifica si una ruta existe
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureDirectory asegura que un directorio exista
func EnsureDirectory(dir string) error {
	if !PathExists(dir) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// AvailableMemory devuelve la memoria disponible en el sistema en MB
func AvailableMemory() (uint64, error) {
	// Implementación básica para Linux
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var available uint64
	_, err = fmt.Fscanf(file, "MemAvailable: %d kB", &available)
	if err != nil {
		return 0, err
	}

	// Convertir de kB a MB
	available /= 1024

	return available, nil
}

// SystemInfo contiene información del sistema
type SystemInfo struct {
	CPU       int
	Memory    uint64 // En MB
	Hostname  string
	IPAddress string
}

// GetSystemInfo obtiene información del sistema
func GetSystemInfo() (*SystemInfo, error) {
	// Implementación básica
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	memory, err := AvailableMemory()
	if err != nil {
		memory = 0
	}

	return &SystemInfo{
		CPU:       4, // Valor por defecto
		Memory:    memory,
		Hostname:  hostname,
		IPAddress: "127.0.0.1", // Valor por defecto
	}, nil
}
