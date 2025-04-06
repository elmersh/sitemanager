# SiteManager

SiteManager (sm) es una herramienta para gestionar rápidamente sitios web en un servidor VPS, incluyendo configuraciones de Nginx, usuarios y despliegue de aplicaciones como Laravel y Node.js.

## Características

- Configuración rápida de sitios web
- Creación automática de usuarios y directorios
- Generación de configuraciones de Nginx
- Configuración automática de SSL con Certbot
- Despliegue de aplicaciones Laravel y Node.js
- Estructura modular para fácil extensión

## Instalación

### Dependencias

- Go 1.16 o superior
- Nginx
- PHP-FPM (para sitios Laravel)
- Node.js y PM2 (para sitios Node.js)
- Certbot (para SSL)

### Compilación e instalación

```bash
# Clonar el repositorio
git clone https://github.com/elmersh/sitemanager.git
cd sitemanager

# Compilar e instalar
make install
```

## Uso

### Configuración

SiteManager busca un archivo de configuración en `~/.config/sitemanager.yaml`. Si no existe, creará uno con valores predeterminados.

Ejemplo de configuración:

```yaml
nginxPath: /etc/nginx
sitesAvailable: /etc/nginx/sites-available
sitesEnabled: /etc/nginx/sites-enabled
defaultUser: www-data
defaultGroup: www-data
phpVersions:
  - 7.4
  - 8.0
  - 8.1
  - 8.2
  - 8.3
  - 8.4
defaultTemplate: laravel
templates:
  laravel: templates/nginx/laravel.conf.tmpl
  nodejs: templates/nginx/nodejs.conf.tmpl
```

### Comandos disponibles

#### Crear un nuevo sitio

```bash
sudo sm site -d ejemplo.com -t laravel -p 8.4
```

Opciones:
- `-d, --domain`: Dominio del sitio (obligatorio)
- `-t, --type`: Tipo de sitio (laravel, nodejs)
- `-p, --php`: Versión de PHP (para sitios Laravel)
- `-P, --port`: Puerto (para sitios Node.js)

#### Configurar SSL

```bash
sudo sm secure -d ejemplo.com -e tu@email.com
```

Opciones:
- `-d, --domain`: Dominio del sitio (obligatorio)
- `-e, --email`: Email para Let's Encrypt (obligatorio)

#### Desplegar una aplicación

```bash
sudo sm deploy -d ejemplo.com -r https://github.com/usuario/repo.git -t laravel
```

Opciones:
- `-d, --domain`: Dominio del sitio (obligatorio)
- `-r, --repo`: Repositorio Git (obligatorio)
- `-b, --branch`: Rama del repositorio (por defecto: main)
- `-t, --type`: Tipo de aplicación (laravel, nodejs)
- `-e, --env`: Entorno (development, production)

## Estructura del proyecto

```
sitemanager/
├── cmd/
│   └── sm/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── commands/
│   │   ├── commands.go
│   │   ├── site.go
│   │   ├── secure.go
│   │   └── deploy.go
│   └── utils/
│       └── utils.go
├── templates/
│   ├── nginx/
│   │   ├── laravel.conf.tmpl
│   │   └── nodejs.conf.tmpl
│   └── ssl/
│       └── ssl.conf.tmpl
├── go.mod
└── go.sum
```

## Contribuir

1. Haz un fork del proyecto
2. Crea una rama para tu característica (`git checkout -b feature/amazing-feature`)
3. Haz commit de tus cambios (`git commit -m 'Add some amazing feature'`)
4. Haz push a la rama (`git push origin feature/amazing-feature`)
5. Abre un Pull Request

## Licencia

Este proyecto está licenciado bajo la Licencia MIT - ver el archivo [LICENSE](LICENSE) para más detalles.