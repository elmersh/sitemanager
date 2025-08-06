package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/elmersh/sitemanager/internal/config"
	"github.com/elmersh/sitemanager/internal/utils"
	"github.com/spf13/cobra"
)

// SiteOptions contiene las opciones para el comando site
type SiteOptions struct {
	Domain       string
	Type         string
	PHP          string
	Port         int
	User         string
	HomeDir      string
	NginxDir     string
	IsSubdomain  bool
	ParentDomain string
	SkelDir      string
}

// AddSiteCommand agrega el comando site al comando raíz
func AddSiteCommand(rootCmd *cobra.Command, cfg *config.Config) {
	// Opciones del comando
	var opts SiteOptions
	var port int

	// Crear comando site
	siteCmd := &cobra.Command{
		Use:   "site",
		Short: "Configurar un nuevo sitio web",
		Long:  `Configura un nuevo sitio web creando un usuario, directorios y configuración de Nginx.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Configurar opciones
			if opts.Domain == "" {
				return fmt.Errorf("el dominio es obligatorio")
			}

			// Usar valores por defecto si no se especifican
			if opts.Type == "" {
				opts.Type = cfg.DefaultTemplate
			}

			// Verificar que el tipo de sitio es válido
			if _, ok := cfg.Templates[opts.Type]; !ok {
				return fmt.Errorf("tipo de sitio no válido: %s", opts.Type)
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
			} else {
				// No es subdominio, configuración normal
				opts.User = domainParts[0]
				opts.HomeDir = filepath.Join("/home", opts.Domain)
			}

			opts.NginxDir = filepath.Join(opts.HomeDir, "nginx")
			opts.Port = port
			opts.SkelDir = cfg.SkelDir

			// Crear usuario y directorios
			if err := createUserAndDirs(&opts); err != nil {
				return err
			}

			// Crear contenido específico según el tipo de sitio
			if err := createSiteContent(&opts); err != nil {
				return err
			}

			// Generar configuración de Nginx
			if err := generateNginxConfig(&opts, cfg); err != nil {
				return err
			}

			// Crear enlaces simbólicos
			if err := createSymlinks(&opts, cfg); err != nil {
				return err
			}

			// Recargar configuración de Nginx
			if err := reloadNginx(); err != nil {
				return err
			}

			fmt.Printf("Sitio %s configurado correctamente\n", opts.Domain)
			return nil
		},
	}

	// Agregar flags
	siteCmd.Flags().StringVarP(&opts.Domain, "domain", "d", "", "Dominio del sitio (obligatorio)")
	siteCmd.Flags().StringVarP(&opts.Type, "type", "t", "", "Tipo de sitio (laravel, nodejs, static)")
	siteCmd.Flags().StringVarP(&opts.PHP, "php", "p", "8.1", "Versión de PHP (para sitios Laravel)")
	siteCmd.Flags().IntVarP(&port, "port", "P", 3000, "Puerto (para sitios Node.js)")

	// Marcar flags obligatorios
	siteCmd.MarkFlagRequired("domain")

	// Validación de requisitos antes de ejecutar
	siteCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// Validar dominio
		if err := utils.ValidateDomain(opts.Domain); err != nil {
			return err
		}

		// Verificar requisitos
		requirements := map[string]string{
			"template": opts.Type,
			"php":      opts.PHP,
		}

		return utils.CheckRequirements("site", requirements)
	}

	// Agregar comando al comando raíz
	rootCmd.AddCommand(siteCmd)
}

// createUserAndDirs crea el usuario y los directorios necesarios
func createUserAndDirs(opts *SiteOptions) error {
	// Verificar si el usuario ya existe
	if _, err := exec.Command("id", opts.User).Output(); err == nil {
		fmt.Printf("Usuario %s ya existe\n", opts.User)
	} else {
		// Solo crear usuario si no es un subdominio (los subdominios usan el usuario del dominio principal)
		if !opts.IsSubdomain {
			// Crear usuario
			fmt.Printf("Creando usuario %s...\n", opts.User)
			cmd := exec.Command("useradd", "-m", "-d", opts.HomeDir, "-s", "/bin/bash", opts.User)
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("error al crear usuario: %v\n%s", err, output)
			}
		} else {
			return fmt.Errorf("el usuario %s no existe, primero debe crear el dominio principal %s", opts.User, opts.ParentDomain)
		}
	}

	// Si es un subdominio, no necesitamos crear la estructura de directorios
	// ya que usará la del dominio principal
	if opts.IsSubdomain {
		return nil
	}

	// Verificar si existe el directorio skel
	if _, err := os.Stat(opts.SkelDir); os.IsNotExist(err) {
		// Si no existe, crearlo con la estructura básica
		fmt.Printf("El directorio skel no existe, creándolo en %s...\n", opts.SkelDir)
		if err := os.MkdirAll(opts.SkelDir, 0755); err != nil {
			return fmt.Errorf("error al crear el directorio skel: %v", err)
		}

		// Crear la estructura básica del directorio skel
		dirs := []string{
			"public_html",
			"nginx",
			"logs",
			"apps",
		}

		for _, dir := range dirs {
			path := filepath.Join(opts.SkelDir, dir)
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("error al crear el directorio %s: %v", path, err)
			}
		}

		// Crear index.html de prueba
		indexFile := filepath.Join(opts.SkelDir, "public_html", "index.html")
		indexContent := fmt.Sprintf("<html><body><h1>Bienvenido a %s</h1><p>Sitio configurado con SiteManager</p></body></html>", opts.Domain)
		if err := os.WriteFile(indexFile, []byte(indexContent), 0644); err != nil {
			return fmt.Errorf("error al crear el archivo index.html: %v", err)
		}

		// Establecer permisos
		cmd := exec.Command("chmod", "-R", "755", opts.SkelDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al establecer permisos: %v\n%s", err, output)
		}
	}

	// Verificar si el directorio skel tiene contenido
	entries, err := os.ReadDir(opts.SkelDir)
	if err != nil {
		return fmt.Errorf("error al leer el directorio skel: %v", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("el directorio skel está vacío: %s", opts.SkelDir)
	}

	// Copiar la estructura del directorio skel al directorio home del usuario
	fmt.Printf("Copiando estructura del directorio skel a %s...\n", opts.HomeDir)

	// Usar rsync si está disponible, de lo contrario usar cp
	if _, err := exec.LookPath("rsync"); err == nil {
		cmd := exec.Command("rsync", "-av", opts.SkelDir+"/", opts.HomeDir+"/")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al copiar estructura del directorio skel con rsync: %v\n%s", err, output)
		}
	} else {
		// Usar cp como alternativa
		cmd := exec.Command("cp", "-r", opts.SkelDir+"/.", opts.HomeDir+"/")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al copiar estructura del directorio skel con cp: %v\n%s", err, output)
		}
	}

	// Configurar permisos y grupos
	// 1. Agregar usuario www-data al grupo del usuario del sitio
	cmd := exec.Command("usermod", "-a", "-G", opts.User, "www-data")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al agregar www-data al grupo: %v\n%s", err, output)
	}

	// 2. Establecer permisos del directorio home
	cmd = exec.Command("chmod", "750", opts.HomeDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al configurar permisos del directorio home: %v\n%s", err, output)
	}

	// 3. Establecer permisos recursivos para el grupo
	cmd = exec.Command("chmod", "-R", "g+rX", opts.HomeDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al configurar permisos recursivos: %v\n%s", err, output)
	}

	// 4. Cambiar propietario de los directorios
	cmd = exec.Command("chown", "-R", fmt.Sprintf("%s:%s", opts.User, opts.User), opts.HomeDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario: %v\n%s", err, output)
	}

	return nil
}

// generateNginxConfig genera la configuración de Nginx para el sitio
func generateNginxConfig(opts *SiteOptions, cfg *config.Config) error {
	// Si es un subdominio, no necesitamos generar la configuración de Nginx
	// ya que usará la del dominio principal
	if opts.IsSubdomain {
		return nil
	}

	// Determinar qué plantilla usar según si es subdominio o no
	var tmplPath string
	if opts.IsSubdomain {
		if path, ok := cfg.SubdomainTemplates[opts.Type]; ok {
			tmplPath = path
		} else {
			// Fallback a plantilla normal si no hay específica para subdominio
			tmplPath = cfg.Templates[opts.Type]
		}
	} else {
		tmplPath = cfg.Templates[opts.Type]
	}

	// Verificar que la ruta no esté vacía
	if tmplPath == "" {
		return fmt.Errorf("no se encontró una plantilla para el tipo de sitio: %s", opts.Type)
	}

	// Intentar usar la plantilla del sistema primero
	systemTmplPath := filepath.Join("/etc/nginx/templates", tmplPath)
	if _, err := os.Stat(systemTmplPath); err == nil {
		tmplPath = systemTmplPath
		fmt.Printf("Usando plantilla del sistema: %s\n", tmplPath)
	} else {
		fmt.Printf("Usando plantilla interna: %s\n", tmplPath)
	}

	tmplContent, err := utils.ReadTemplateFile(tmplPath)
	if err != nil {
		return err
	}

	// Crear plantilla
	tmpl, err := template.New("nginx").Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("error al parsear plantilla: %v", err)
	}

	// Datos para la plantilla
	data := map[string]interface{}{
		"Domain":   opts.Domain,
		"RootDir":  filepath.Join(opts.HomeDir, "public_html"),
		"PHP":      strings.TrimPrefix(opts.PHP, "php"), // Asegurar que no tenga el prefijo "php"
		"Port":     opts.Port,
		"User":     opts.User,
		"HomeDir":  opts.HomeDir,
		"NginxDir": opts.NginxDir,
	}

	// Archivo de configuración
	confFile := filepath.Join(opts.NginxDir, fmt.Sprintf("%s.conf", opts.Domain))
	file, err := os.Create(confFile)
	if err != nil {
		return fmt.Errorf("error al crear archivo de configuración: %v", err)
	}
	defer file.Close()

	// Ejecutar plantilla
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("error al ejecutar plantilla: %v", err)
	}

	// Cerrar el archivo antes de cambiar el propietario
	file.Close()

	// Cambiar propietario del archivo de configuración
	cmd := exec.Command("chown", fmt.Sprintf("%s:%s", opts.User, opts.User), confFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario del archivo de configuración: %v\n%s", err, output)
	}

	fmt.Printf("Configuración de Nginx generada en %s\n", confFile)
	return nil
}

// createSymlinks crea los enlaces simbólicos en los directorios de Nginx
func createSymlinks(opts *SiteOptions, cfg *config.Config) error {
	// Origen
	confFile := filepath.Join(opts.NginxDir, fmt.Sprintf("%s.conf", opts.Domain))

	// Destino en sites-available
	availableLink := filepath.Join(cfg.SitesAvailable, fmt.Sprintf("%s.conf", opts.Domain))

	// Eliminar enlace existente si existe
	os.Remove(availableLink)

	// Crear enlace en sites-available
	if err := os.Symlink(confFile, availableLink); err != nil {
		return fmt.Errorf("error al crear enlace en sites-available: %v", err)
	}

	// Destino en sites-enabled
	enabledLink := filepath.Join(cfg.SitesEnabled, fmt.Sprintf("%s.conf", opts.Domain))

	// Eliminar enlace existente si existe
	os.Remove(enabledLink)

	// Crear enlace en sites-enabled
	if err := os.Symlink(confFile, enabledLink); err != nil {
		return fmt.Errorf("error al crear enlace en sites-enabled: %v", err)
	}

	fmt.Println("Enlaces simbólicos creados correctamente")
	return nil
}

// reloadNginx recarga la configuración de Nginx
func reloadNginx() error {
	fmt.Println("Recargando configuración de Nginx...")
	cmd := exec.Command("systemctl", "reload", "nginx")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al recargar Nginx: %v\n%s", err, output)
	}
	return nil
}

// createSiteContent crea contenido específico según el tipo de sitio
func createSiteContent(opts *SiteOptions) error {
	if opts.Type == "static" {
		return createStaticSiteContent(opts)
	}
	// Para otros tipos (laravel, nodejs) no necesitamos contenido específico por ahora
	return nil
}

// createStaticSiteContent crea un sitio estático básico con HTML, CSS y JS
func createStaticSiteContent(opts *SiteOptions) error {
	publicDir := filepath.Join(opts.HomeDir, "public_html")
	
	// Crear directorios adicionales para sitio estático
	dirs := []string{
		filepath.Join(publicDir, "css"),
		filepath.Join(publicDir, "js"),
		filepath.Join(publicDir, "img"),
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error al crear directorio %s: %v", dir, err)
		}
	}
	
	// Crear index.html más completo
	indexFile := filepath.Join(publicDir, "index.html")
	indexContent := fmt.Sprintf(`<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Bienvenido a %s</title>
    <link rel="stylesheet" href="css/style.css">
</head>
<body>
    <header>
        <nav>
            <h1>%s</h1>
            <ul>
                <li><a href="#inicio">Inicio</a></li>
                <li><a href="#about">Acerca de</a></li>
                <li><a href="#contact">Contacto</a></li>
            </ul>
        </nav>
    </header>

    <main>
        <section id="inicio" class="hero">
            <h2>¡Bienvenido a tu nuevo sitio web!</h2>
            <p>Este sitio ha sido configurado automáticamente con SiteManager.</p>
            <p>Puedes editar los archivos en <code>/home/%s/public_html/</code></p>
            <button onclick="showMessage()">¡Haz clic aquí!</button>
        </section>

        <section id="about" class="content">
            <h3>Acerca de este sitio</h3>
            <p>Este es un sitio web estático generado automáticamente. Incluye:</p>
            <ul>
                <li>HTML5 semántico</li>
                <li>CSS3 responsive</li>
                <li>JavaScript básico</li>
                <li>Estructura de directorios organizada</li>
            </ul>
        </section>

        <section id="contact" class="content">
            <h3>¿Listo para personalizar?</h3>
            <p>Edita estos archivos para personalizar tu sitio:</p>
            <ul>
                <li><strong>index.html</strong> - Contenido principal</li>
                <li><strong>css/style.css</strong> - Estilos</li>
                <li><strong>js/script.js</strong> - Funcionalidad</li>
            </ul>
        </section>
    </main>

    <footer>
        <p>&copy; 2024 %s. Sitio generado con SiteManager.</p>
    </footer>

    <script src="js/script.js"></script>
</body>
</html>`, opts.Domain, opts.Domain, opts.Domain, opts.Domain)
	
	if err := os.WriteFile(indexFile, []byte(indexContent), 0644); err != nil {
		return fmt.Errorf("error al crear index.html: %v", err)
	}
	
	// Crear archivo CSS básico
	cssFile := filepath.Join(publicDir, "css", "style.css")
	cssContent := `/* Estilos básicos para el sitio */
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    line-height: 1.6;
    color: #333;
    background-color: #f4f4f4;
}

header {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    padding: 1rem 0;
    box-shadow: 0 2px 5px rgba(0,0,0,0.1);
}

nav {
    max-width: 1200px;
    margin: 0 auto;
    padding: 0 20px;
    display: flex;
    justify-content: space-between;
    align-items: center;
}

nav h1 {
    font-size: 1.8rem;
}

nav ul {
    display: flex;
    list-style: none;
}

nav ul li {
    margin-left: 2rem;
}

nav ul li a {
    color: white;
    text-decoration: none;
    font-weight: 500;
    transition: opacity 0.3s ease;
}

nav ul li a:hover {
    opacity: 0.8;
}

main {
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
}

.hero {
    background: white;
    padding: 3rem 2rem;
    border-radius: 10px;
    text-align: center;
    margin-bottom: 2rem;
    box-shadow: 0 5px 15px rgba(0,0,0,0.1);
}

.hero h2 {
    color: #667eea;
    margin-bottom: 1rem;
    font-size: 2.2rem;
}

.hero p {
    margin-bottom: 1rem;
    font-size: 1.1rem;
}

.hero code {
    background-color: #f8f9fa;
    padding: 2px 6px;
    border-radius: 3px;
    font-family: 'Courier New', monospace;
}

button {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    border: none;
    padding: 12px 24px;
    font-size: 1rem;
    border-radius: 5px;
    cursor: pointer;
    transition: transform 0.3s ease;
}

button:hover {
    transform: translateY(-2px);
}

.content {
    background: white;
    padding: 2rem;
    margin-bottom: 2rem;
    border-radius: 10px;
    box-shadow: 0 5px 15px rgba(0,0,0,0.1);
}

.content h3 {
    color: #667eea;
    margin-bottom: 1rem;
    font-size: 1.5rem;
}

.content ul {
    margin-left: 1.5rem;
}

.content li {
    margin-bottom: 0.5rem;
}

footer {
    background: #333;
    color: white;
    text-align: center;
    padding: 2rem;
    margin-top: 2rem;
}

/* Responsive */
@media (max-width: 768px) {
    nav {
        flex-direction: column;
        text-align: center;
    }
    
    nav ul {
        margin-top: 1rem;
    }
    
    nav ul li {
        margin: 0 1rem;
    }
    
    .hero {
        padding: 2rem 1rem;
    }
    
    .hero h2 {
        font-size: 1.8rem;
    }
    
    main {
        padding: 10px;
    }
}
`
	
	if err := os.WriteFile(cssFile, []byte(cssContent), 0644); err != nil {
		return fmt.Errorf("error al crear style.css: %v", err)
	}
	
	// Crear archivo JavaScript básico
	jsFile := filepath.Join(publicDir, "js", "script.js")
	jsContent := `// JavaScript básico para el sitio

// Función que se ejecuta cuando se hace clic en el botón
function showMessage() {
    alert('¡Bienvenido a tu sitio web estático! 🎉\\n\\nAhora puedes personalizar este sitio editando los archivos en tu servidor.');
}

// Smooth scroll para los enlaces de navegación
document.addEventListener('DOMContentLoaded', function() {
    // Agregar comportamiento smooth scroll a los enlaces
    const links = document.querySelectorAll('nav a[href^="#"]');
    
    links.forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            
            const targetId = this.getAttribute('href');
            const targetElement = document.querySelector(targetId);
            
            if (targetElement) {
                targetElement.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }
        });
    });
    
    // Agregar efecto de aparición al cargar la página
    const sections = document.querySelectorAll('section');
    sections.forEach(section => {
        section.style.opacity = '0';
        section.style.transform = 'translateY(20px)';
        section.style.transition = 'all 0.6s ease';
    });
    
    // Animar las secciones
    setTimeout(() => {
        sections.forEach((section, index) => {
            setTimeout(() => {
                section.style.opacity = '1';
                section.style.transform = 'translateY(0)';
            }, index * 200);
        });
    }, 100);
});

// Función para mostrar la fecha actual
function updateDateTime() {
    const now = new Date();
    const options = { 
        year: 'numeric', 
        month: 'long', 
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    };
    
    const dateTimeString = now.toLocaleDateString('es-ES', options);
    
    // Si hay un elemento con id 'datetime', actualizar su contenido
    const dateTimeElement = document.getElementById('datetime');
    if (dateTimeElement) {
        dateTimeElement.textContent = dateTimeString;
    }
}

// Actualizar fecha y hora cada minuto
setInterval(updateDateTime, 60000);
updateDateTime(); // Ejecutar inmediatamente
`
	
	if err := os.WriteFile(jsFile, []byte(jsContent), 0644); err != nil {
		return fmt.Errorf("error al crear script.js: %v", err)
	}
	
	// Crear archivo README.md para el desarrollador
	readmeFile := filepath.Join(opts.HomeDir, "README.md")
	readmeContent := fmt.Sprintf("# Sitio Web Estático - %s\n\n"+
		"Este sitio ha sido generado automáticamente por SiteManager.\n\n"+
		"## Estructura de archivos\n\n"+
		"```\n"+
		"%s/\n"+
		"├── public_html/          # Directorio público (raíz del sitio web)\n"+
		"│   ├── index.html        # Página principal\n"+
		"│   ├── css/\n"+
		"│   │   └── style.css     # Estilos del sitio\n"+
		"│   ├── js/\n"+
		"│   │   └── script.js     # JavaScript del sitio\n"+
		"│   └── img/              # Directorio para imágenes\n"+
		"├── logs/                 # Logs de Nginx\n"+
		"├── nginx/               # Configuraciones de Nginx\n"+
		"└── README.md            # Este archivo\n"+
		"```\n\n"+
		"## Personalización\n\n"+
		"1. **HTML**: Edita `public_html/index.html` para cambiar el contenido\n"+
		"2. **CSS**: Modifica `public_html/css/style.css` para cambiar los estilos\n"+
		"3. **JavaScript**: Actualiza `public_html/js/script.js` para agregar funcionalidad\n"+
		"4. **Imágenes**: Coloca tus imágenes en `public_html/img/`\n\n"+
		"## Comandos útiles\n\n"+
		"```bash\n"+
		"# Ver logs del sitio\n"+
		"sudo tail -f %s/logs/access.log\n"+
		"sudo tail -f %s/logs/error.log\n\n"+
		"# Configurar SSL (recomendado)\n"+
		"sudo sm secure -d %s -e tu@email.com\n\n"+
		"# Verificar configuración de Nginx\n"+
		"sudo nginx -t\n\n"+
		"# Recargar Nginx después de cambios\n"+
		"sudo systemctl reload nginx\n"+
		"```\n\n"+
		"## Tecnologías incluidas\n\n"+
		"- HTML5 semántico\n"+
		"- CSS3 con Flexbox y Grid\n"+
		"- JavaScript vanilla (ES6+)\n"+
		"- Diseño responsive\n"+
		"- Optimización para SEO básico\n\n"+
		"## Notas importantes\n\n"+
		"- Los archivos web deben estar en `public_html/`\n"+
		"- El sitio es completamente estático (no requiere PHP, Node.js, etc.)\n"+
		"- Para cambios en la configuración de Nginx, contacta al administrador del servidor\n"+
		"- Recuerda hacer respaldos regulares de tus archivos\n\n"+
		"¡Disfruta personalizando tu sitio web!\n",
		opts.Domain, opts.HomeDir, opts.HomeDir, opts.HomeDir, opts.Domain)
	
	if err := os.WriteFile(readmeFile, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("error al crear README.md: %v", err)
	}
	
	// Establecer permisos correctos
	cmd := exec.Command("chown", "-R", fmt.Sprintf("%s:%s", opts.User, opts.User), opts.HomeDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al establecer permisos: %v\n%s", err, output)
	}
	
	fmt.Printf("✅ Sitio estático creado con estructura completa en %s\n", filepath.Join(opts.HomeDir, "public_html"))
	return nil
}
