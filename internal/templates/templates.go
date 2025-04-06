package templates

import (
	"embed"
	"io/fs"
)

//go:embed nginx/*.conf.tmpl ssl/*.conf.tmpl
var templateFS embed.FS

// GetTemplateContent lee el contenido de una plantilla por su ruta
func GetTemplateContent(path string) (string, error) {
	data, err := templateFS.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ListTemplates lista todas las plantillas disponibles
func ListTemplates() ([]string, error) {
	var templates []string
	err := fs.WalkDir(templateFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && fs.ValidPath(path) {
			templates = append(templates, path)
		}
		return nil
	})
	return templates, err
}
