# SiteManager

<div align="center">

[![GitHub release](https://img.shields.io/github/v/release/elmersh/sitemanager?style=flat-square)](https://github.com/elmersh/sitemanager/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8.svg?style=flat-square)](https://golang.org/)
[![Platform](https://img.shields.io/badge/platform-Linux-blue.svg?style=flat-square)](https://github.com/elmersh/sitemanager)

**Herramienta CLI para gestionar sitios web en servidores VPS de forma rápida y sencilla**

[Instalación](#-instalación) • [Uso](#-uso) • [Documentación](#-documentación) • [Contribuir](#-contribuir)

</div>

---

SiteManager (`sm`) es una herramienta de línea de comandos que automatiza la configuración y gestión de sitios web en servidores VPS Ubuntu/Debian. Simplifica tareas complejas como la configuración de Nginx, SSL, usuarios del sistema y despliegue de aplicaciones.

## ✨ Características

- 🚀 **Instalación rápida**: Un comando para instalar desde internet
- 🔧 **Configuración automática**: Nginx, usuarios y directorios
- 🔒 **SSL automático**: Integración completa con Let's Encrypt/Certbot
- 📦 **Multi-framework**: Laravel, Node.js, sitios estáticos
- 🔄 **Auto-actualización**: `sm self-update` para mantener la última versión
- 🌐 **Subdominios**: Detección y configuración automática
- 📊 **Gestión de dependencias**: Verificación inteligente sin instalación forzada
- 🎯 **Fácil de usar**: Sintaxis simple e intuitiva

## 🚀 Instalación

### Instalación rápida (recomendada)

```bash
curl -fsSL https://raw.githubusercontent.com/elmersh/sitemanager/main/install.sh | sudo bash
```

### Instalación manual

```bash
# Descargar la última versión
wget https://github.com/elmersh/sitemanager/releases/latest/download/sitemanager-1.0.0-linux-amd64.tar.gz

# Extraer e instalar
tar -xzf sitemanager-1.0.0-linux-amd64.tar.gz
cd sitemanager-1.0.0/
sudo ./install.sh
```

### Auto-actualización

```bash
# Verificar actualizaciones
sm version check

# Actualizar automáticamente
sudo sm self-update
```

## 📋 Dependencias

SiteManager **no instala automáticamente** las dependencias del sistema. Te informa qué falta y cómo instalarlo:

### Obligatorias para todos
- **Nginx** - servidor web principal

### Opcionales según tipo de sitio
- **PHP-FPM** - para sitios Laravel/PHP
- **Node.js + PM2** - para aplicaciones Node.js
- **Certbot** - para certificados SSL automáticos
- **Composer** - para proyectos Laravel

## ⚙️ Configuración inicial

### 1. Verificar el sistema
```bash
sudo sm status
```

### 2. Configurar email para SSL (obligatorio)
```bash
# Editar configuración
nano ~/.config/sitemanager/config.yaml

# Establecer tu email y aceptar términos
email: tu@email.com
agree_tos: true
```

La configuración se crea automáticamente con valores por defecto en `~/.config/sitemanager/config.yaml`:

```yaml
# Configuración básica
email: ""                    # Tu email (requerido para SSL)
default_php: "8.3"          # Versión PHP por defecto
default_port: 3000          # Puerto base para Node.js

# SSL/Certificados
agree_tos: false            # Debe ser true para SSL
use_staging: false          # false = certificados reales
backup_configs: true        # Backup automático de configs

# Funciones avanzadas
auto_update: false          # Auto-actualización (recomendado: false)
check_updates: true         # Verificar actualizaciones
```

## 💻 Uso básico

### 1. Crear un sitio web
```bash
# Sitio Laravel
sudo sm site -d miapp.com -t laravel

# Sitio Node.js
sudo sm site -d miapi.com -t nodejs -P 3001

# Sitio estático
sudo sm site -d miweb.com -t static

# Subdominio
sudo sm site -d admin.miapp.com -t laravel
```

### 2. Configurar SSL
```bash
# Usando email de configuración
sudo sm secure -d miapp.com

# Especificando email
sudo sm secure -d miapp.com -e admin@miapp.com
```

### 3. Desplegar aplicación
```bash
# Repositorio público
sudo sm deploy -d miapp.com -r https://github.com/usuario/mi-app.git

# Repositorio privado (SSH)
sudo sm deploy -d miapp.com -r git@github.com:usuario/mi-app.git -s
```

### 4. Gestionar variables de entorno
```bash
# Modo interactivo (recomendado)
sudo sm env -d miapp.com -i

# Variables específicas
sudo sm env -d miapp.com -e DATABASE_URL=postgresql://... -e JWT_SECRET=abc123
```

## 🔧 Comandos disponibles

| Comando | Descripción | Ejemplo |
|---------|-------------|---------|
| `sm status` | Verificar estado del sistema | `sudo sm status` |
| `sm site` | Crear/configurar sitio web | `sudo sm site -d miapp.com -t laravel` |
| `sm secure` | Configurar SSL/HTTPS | `sudo sm secure -d miapp.com` |
| `sm deploy` | Desplegar aplicación | `sudo sm deploy -d miapp.com -r repo.git` |
| `sm env` | Gestionar variables de entorno | `sudo sm env -d miapp.com -i` |
| `sm self-update` | Actualizar SiteManager | `sudo sm self-update` |
| `sm version` | Ver información de versión | `sm version` |
| `sm version check` | Verificar actualizaciones | `sm version check` |

### Desplegar una aplicación

```bash
sudo sm deploy -d ejemplo.com -r https://github.com/usuario/repo.git
```

Opciones disponibles:
- `-d, --domain`: Dominio del sitio (obligatorio)
- `-r, --repo`: URL del repositorio Git (obligatorio)
- `-b, --branch`: Rama a desplegar (por defecto: main)
- `-t, --type`: Tipo de aplicación (laravel, nodejs, static)
- `-e, --env`: Entorno de despliegue (development, production)
- `-s, --ssh`: Usar SSH para repositorios privados

**Ejemplos:**

Aplicación Laravel (repositorio público):
```bash
sudo sm deploy -d miapp.com -r https://github.com/usuario/miapp.git -t laravel
```

Aplicación Node.js (repositorio privado):
```bash
sudo sm deploy -d miapi.com -r git@github.com:usuario/miapi.git -t nodejs -s
```

Despliegue en subdominio con rama específica:
```bash
sudo sm deploy -d admin.miapp.com -r https://github.com/usuario/admin-panel.git -b develop
```

### Configurar variables de entorno

```bash
sudo sm env -d ejemplo.com [opciones]
```

Opciones disponibles:
- `-d, --domain`: Dominio del sitio (obligatorio)
- `-e, --env`: Variables de entorno (formato KEY=VALUE)
- `-i, --interactive`: Modo interactivo para configurar variables
- `-f, --file`: Importar desde archivo .env existente

**Ejemplos:**

Configuración interactiva (recomendada):
```bash
sudo sm env -d miapi.com -i
```

Establecer variables específicas:
```bash
sudo sm env -d miapi.com -e DATABASE_URL=postgresql://user:pass@localhost/db -e JWT_SECRET=secretkey
```

Importar desde archivo:
```bash
sudo sm env -d miapi.com -f /ruta/al/.env.production
```

## Sitios Estáticos

SiteManager incluye soporte completo para sitios web estáticos que solo requieren HTML, CSS y JavaScript.

### Características de los sitios estáticos

- **Estructura automática**: Crea directorios organizados (css/, js/, img/)
- **Plantilla base**: Genera un sitio de ejemplo completamente funcional
- **Optimización**: Configuración de Nginx optimizada para archivos estáticos
- **Caché**: Configuración automática de caché para mejor rendimiento
- **Responsive**: CSS responsive incluido por defecto
- **SEO básico**: Estructura HTML optimizada para motores de búsqueda

### Estructura generada

Cuando creas un sitio estático, SiteManager genera automáticamente:

```
/home/dominio.com/
├── public_html/
│   ├── index.html          # Página principal con navegación
│   ├── css/
│   │   └── style.css       # CSS responsive completo
│   ├── js/
│   │   └── script.js       # JavaScript interactivo
│   └── img/                # Directorio para imágenes
├── logs/                   # Logs de Nginx
├── nginx/                  # Configuración de Nginx
└── README.md               # Guía de personalización
```

### Características del template incluido

- **HTML5 semántico** con estructura de navegación
- **CSS moderno** con gradientes, animaciones y diseño responsive
- **JavaScript funcional** con smooth scroll y efectos de aparición
- **Configuración de caché** optimizada en Nginx
- **Compresión GZIP** habilitada automáticamente

### Personalización rápida

Los archivos generados son completamente editables:

1. **Contenido**: Edita `public_html/index.html`
2. **Estilos**: Modifica `public_html/css/style.css`
3. **Funcionalidad**: Actualiza `public_html/js/script.js`
4. **Imágenes**: Añade archivos a `public_html/img/`

### Compatibilidad con frameworks frontend

Los sitios estáticos son compatibles con:
- **React** (build estático)
- **Vue.js** (build estático)
- **Angular** (build estático)
- **Cualquier generador de sitios estáticos** (Gatsby, Next.js export, etc.)

Simplemente reemplaza el contenido de `public_html/` con tu build de producción.

## Características avanzadas

### Manejo de subdominios

SiteManager detecta automáticamente cuando se está trabajando con un subdominio y aplica la configuración adecuada:

- Utiliza el mismo usuario del dominio principal
- Crea configuraciones específicas para el subdominio
- Asigna puertos dinámicos para aplicaciones Node.js basados en el nombre del subdominio
- Organiza los directorios en el servidor de manera lógica

### Detección automática de frameworks Node.js

SiteManager detecta automáticamente el tipo de framework utilizado en proyectos Node.js:

- **NestJS**: Detecta aplicaciones NestJS con soporte para TypeScript
- **NextJS**: Configura correctamente aplicaciones NextJS
- **Express**: Maneja aplicaciones simples con Express
- **ReactJS**: Configura aplicaciones React con funcionalidades específicas
- **VueJS**: Soporte para aplicaciones Vue

### Integración con bases de datos

SiteManager puede configurar automáticamente bases de datos para aplicaciones:

- **PostgreSQL**: Creación automática de bases de datos, usuarios y esquemas
- **MySQL**: Configuración completa con charset y collation adecuados
- **MongoDB**: Soporte para conexiones a MongoDB

### Soporte para Prisma ORM

Para proyectos que utilizan Prisma ORM:

- Detección automática de Prisma en el proyecto
- Generación de cliente Prisma
- Ejecución automática de migraciones
- Configuración de la conexión a la base de datos

### Despliegue con SSH

Para repositorios privados, SiteManager puede configurar claves SSH automáticamente:

1. Genera una clave SSH única para cada combinación de dominio y repositorio
2. Muestra la clave pública para que puedas añadirla a las Deploy Keys de GitHub
3. Configura automáticamente el archivo `.ssh/config` con la configuración adecuada
4. Reutiliza las claves existentes en despliegues posteriores

### Configuración automática de logs

SiteManager configura automáticamente registros de logs:

- Para aplicaciones Node.js: configura PM2 para guardar logs de salida y error
- Para todas las configuraciones: configura logs de acceso y error de Nginx
- Todos los logs se organizan en el directorio `/home/dominio/logs/`

## Flujo de trabajo típico

1. **Crear sitio web**:
   ```bash
   sudo sm site -d miapp.com -t laravel
   ```

2. **Configurar SSL**:
   ```bash
   sudo sm secure -d miapp.com -e admin@miapp.com
   ```

3. **Desplegar aplicación**:
   ```bash
   sudo sm deploy -d miapp.com -r git@github.com:usuario/miapp.git -s
   ```

4. **Configurar variables de entorno**:
   ```bash
   sudo sm env -d miapp.com -i
   ```

## 🛠️ Desarrollo

### Compilar desde el código fuente

```bash
# Clonar el repositorio
git clone https://github.com/elmersh/sitemanager.git
cd sitemanager

# Compilar
make build

# Instalar localmente (opcional)
sudo make install
```

### Estructura del proyecto

```
sitemanager/
├── cmd/sm/              # Punto de entrada
├── internal/
│   ├── commands/        # Implementación de comandos CLI
│   ├── config/          # Gestión de configuración
│   ├── templates/       # Templates de archivos
│   └── utils/           # Utilidades compartidas
├── scripts/            # Scripts de construcción
└── docs/              # Documentación adicional
```

## 🤝 Contribuir

¡Las contribuciones son bienvenidas! Por favor:

1. Haz fork del repositorio
2. Crea una rama para tu funcionalidad (`git checkout -b feature/nueva-funcionalidad`)
3. Haz commit de tus cambios (`git commit -m 'Añadir nueva funcionalidad'`)
4. Push a la rama (`git push origin feature/nueva-funcionalidad`)
5. Abre un Pull Request

Consulta [CONTRIBUTING.md](CONTRIBUTING.md) para más detalles.

## 📜 Licencia

Este proyecto está bajo la Licencia MIT. Ver el archivo [LICENSE](LICENSE) para más detalles.

## ⭐ Soporte

Si encuentras útil este proyecto, considera darle una estrella ⭐ en GitHub.

Para reportar bugs o solicitar funcionalidades, abre un [issue](https://github.com/elmersh/sitemanager/issues).