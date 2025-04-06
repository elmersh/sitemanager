# SiteManager

SiteManager (sm) es una herramienta para gestionar rápidamente sitios web en un servidor VPS, incluyendo configuraciones de Nginx, usuarios y despliegue de aplicaciones como Laravel y Node.js.

## Características

- Configuración rápida de sitios web
- Creación automática de usuarios y directorios
- Generación de configuraciones de Nginx
- Configuración automática de SSL con Certbot
- Despliegue de aplicaciones Laravel y Node.js
- Detección automática de frameworks Node.js
- Gestión de variables de entorno
- Integración con bases de datos
- Estructura modular para fácil extensión

## Instalación

### Dependencias

- Go 1.16 o superior
- Nginx
- PHP-FPM (para sitios Laravel)
- Node.js y PM2 (para sitios Node.js)
- Certbot (para SSL)
- PostgreSQL/MySQL (opcional, para bases de datos)

### Compilación e instalación

```bash
# Clonar el repositorio
git clone https://github.com/yourusername/sitemanager.git
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
  laravel: nginx/laravel.conf.tmpl
  nodejs: nginx/nodejs.conf.tmpl
subdomainTemplates:
  laravel: nginx/subdomain_laravel.conf.tmpl
  nodejs: nginx/subdomain_nodejs.conf.tmpl
```

## Comandos disponibles

### Verificar estado del sistema

Comprueba que todas las dependencias necesarias están correctamente instaladas y funcionando:

```bash
sudo sm status
```

Este comando verificará la disponibilidad de:
- Nginx
- PHP
- Node.js
- PM2
- Certbot
- Composer

### Crear un nuevo sitio web

```bash
sudo sm site -d ejemplo.com -t laravel -p 8.4
```

Opciones:
- `-d, --domain`: Dominio del sitio (obligatorio)
- `-t, --type`: Tipo de sitio (laravel, nodejs)
- `-p, --php`: Versión de PHP (para sitios Laravel)
- `-P, --port`: Puerto para aplicaciones Node.js (default: 3000)

**Ejemplos de uso:**

Para un sitio Laravel:
```bash
sudo sm site -d miapp.com -t laravel -p 8.2
```

Para un sitio Node.js:
```bash
sudo sm site -d miapi.com -t nodejs -P 3001
```

Para un subdominio:
```bash
sudo sm site -d admin.miapp.com -t laravel
```

**Notas:**
- El comando crea un usuario en el sistema con el nombre del dominio
- Configura Nginx con las plantillas adecuadas
- Para subdominios, utiliza el usuario del dominio principal

### Configurar SSL con Certbot

```bash
sudo sm secure -d ejemplo.com -e tu@email.com
```

Opciones:
- `-d, --domain`: Dominio del sitio (obligatorio)
- `-e, --email`: Email para Let's Encrypt (obligatorio)

**Ejemplos de uso:**

Configurar SSL para un dominio principal:
```bash
sudo sm secure -d miapp.com -e admin@miapp.com
```

Configurar SSL para un subdominio:
```bash
sudo sm secure -d api.miapp.com -e admin@miapp.com
```

**Notas:**
- El comando utiliza Certbot para obtener certificados SSL
- Actualiza la configuración de Nginx para usar HTTPS
- Configura la redirección de HTTP a HTTPS

### Desplegar una aplicación

```bash
sudo sm deploy -d ejemplo.com -r https://github.com/usuario/repo.git -t laravel
```

Opciones:
- `-d, --domain`: Dominio del sitio (obligatorio)
- `-r, --repo`: Repositorio Git (obligatorio)
- `-b, --branch`: Rama del repositorio (por defecto: main)
- `-t, --type`: Tipo de aplicación (laravel, nodejs)
- `-e, --env`: Entorno (development, production)
- `-s, --ssh`: Usar SSH para clonar el repositorio

**Ejemplos de uso:**

Desplegar una aplicación Laravel usando HTTPS:
```bash
sudo sm deploy -d miapp.com -r https://github.com/usuario/miapp.git -t laravel
```

Desplegar una aplicación Node.js usando SSH (recomendado para repos privados):
```bash
sudo sm deploy -d miapi.com -r git@github.com:usuario/miapi.git -t nodejs -s
```

Desplegar en un subdominio con una rama específica:
```bash
sudo sm deploy -d admin.miapp.com -r https://github.com/usuario/admin-panel.git -t laravel -b develop
```

**Notas:**
- Para repositorios privados, use la opción `-s` para configurar claves SSH
- El comando detecta automáticamente el tipo de framework (NestJS, NextJS, Express, etc.)
- Para Laravel, ejecuta automáticamente composer install, migraciones, etc.
- Para Node.js, instala dependencias, ejecuta build y configura PM2

### Configurar variables de entorno

```bash
sudo sm env -d ejemplo.com [opciones]
```

Opciones:
- `-d, --domain`: Dominio del sitio (obligatorio)
- `-e, --env`: Variables de entorno en formato KEY=VALUE
- `-i, --interactive`: Modo interactivo para configurar variables
- `-f, --file`: Archivo .env a importar

**Ejemplos de uso:**

Configuración interactiva (recomendada):
```bash
sudo sm env -d miapi.com -i
```

Establecer variables específicas:
```bash
sudo sm env -d miapi.com -e DATABASE_URL=postgresql://user:pass@localhost:5432/midb -e JWT_SECRET=secretkey
```

Importar desde un archivo existente:
```bash
sudo sm env -d miapi.com -f /path/to/.env.production
```

**Notas:**
- El modo interactivo detecta variables de `.env.example` si existe
- Las contraseñas y secretos se ingresan con entrada oculta
- Genera automáticamente valores seguros para tokens y secretos
- Configura correctamente el propietario del archivo .env

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
│   │   ├── deploy.go
│   │   └── env.go
│   ├── templates/
│   │   ├── nginx/
│   │   │   ├── laravel.conf.tmpl
│   │   │   ├── nodejs.conf.tmpl
│   │   │   ├── subdomain_laravel.conf.tmpl
│   │   │   └── subdomain_nodejs.conf.tmpl
│   │   └── ssl/
│   │       └── ssl.conf.tmpl
│   └── utils/
│       ├── utils.go
│       ├── database.go
│       └── nodejs.go
├── go.mod
└── go.sum
```

## Solución de problemas

### Permisos incorrectos

Si encuentras problemas de permisos al desplegar aplicaciones:

```bash
# Verificar y corregir propietario de directorios
sudo chown -R usuario:usuario /home/dominio.com

# Verificar permisos de directorio .ssh
sudo chmod 700 /home/dominio.com/.ssh
sudo chmod 600 /home/dominio.com/.ssh/*
```

### Problemas con PM2

Si una aplicación Node.js no inicia correctamente:

```bash
# Ver logs de la aplicación
pm2 logs dominio.com

# Reiniciar la aplicación
pm2 restart dominio.com

# Configurar manualmente el archivo de entorno
sudo sm env -d dominio.com -i
```

## Contribuir

1. Haz un fork del proyecto
2. Crea una rama para tu característica (`git checkout -b feature/amazing-feature`)
3. Haz commit de tus cambios (`git commit -m 'Add some amazing feature'`)
4. Haz push a la rama (`git push origin feature/amazing-feature`)
5. Abre un Pull Request

## Licencia

Este proyecto está licenciado bajo la Licencia MIT - ver el archivo [LICENSE](LICENSE) para más detalles.