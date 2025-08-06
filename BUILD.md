# Guía de Compilación y Distribución - SiteManager

Este documento detalla cómo compilar, empaquetar y distribuir SiteManager para servidores Debian/Ubuntu.

## Requisitos de Compilación

### Entorno de Desarrollo
- Go 1.23.0 o superior
- Make
- Git
- Acceso a internet para descargar dependencias

### Dependencias del Sistema de Destino (Debian/Ubuntu)
- Nginx
- PHP-FPM (8.0 o superior recomendado)
- Node.js y npm (para sitios Node.js)
- PM2 (para gestión de procesos Node.js)
- Certbot (para SSL)
- PostgreSQL o MySQL (opcional)

## Compilación

### 1. Compilación Local (para desarrollo)

```bash
# Clonar el repositorio
git clone https://github.com/elmersh/sitemanager.git
cd sitemanager

# Instalar dependencias
make deps

# Compilar para la plataforma actual
make build

# El binario se genera como 'sm' en la raíz del proyecto
```

### 2. Compilación Cruzada para Ubuntu/Debian

```bash
# Limpiar builds anteriores
make clean

# Compilar específicamente para Ubuntu/Debian (Linux AMD64)
make ubuntu

# Esto genera:
# - sitemanager/sm (binario)
# - sitemanager.tar.gz (paquete comprimido)
```

### 3. Compilación Manual Cruzada

```bash
# Para Ubuntu/Debian 64-bit
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=1.0.0" -o sm-linux-amd64 cmd/sm/main.go

# Para Ubuntu/Debian 32-bit (si es necesario)
GOOS=linux GOARCH=386 go build -ldflags "-X main.Version=1.0.0" -o sm-linux-386 cmd/sm/main.go

# Para ARM64 (servidores ARM como Raspberry Pi)
GOOS=linux GOARCH=arm64 go build -ldflags "-X main.Version=1.0.0" -o sm-linux-arm64 cmd/sm/main.go
```

## Empaquetado para Distribución

### 1. Paquete TAR.GZ (Recomendado)

```bash
# Crear estructura de directorios
mkdir -p dist/sitemanager-1.0.0/{bin,templates/nginx,templates/ssl,scripts}

# Copiar binario compilado
cp sm-linux-amd64 dist/sitemanager-1.0.0/bin/sm

# Copiar templates
cp internal/templates/nginx/*.tmpl dist/sitemanager-1.0.0/templates/nginx/
cp internal/templates/ssl/*.tmpl dist/sitemanager-1.0.0/templates/ssl/

# Crear script de instalación
cat > dist/sitemanager-1.0.0/install.sh << 'EOF'
#!/bin/bash

set -e

# Verificar permisos de sudo
if [ "$EUID" -ne 0 ]; then
    echo "Este script debe ejecutarse con sudo"
    exit 1
fi

# Variables
PREFIX=${PREFIX:-/usr/local}
BINDIR=$PREFIX/bin
SHAREDIR=$PREFIX/share/sitemanager
CONFDIR=/etc/sitemanager

echo "Instalando SiteManager..."

# Crear directorios
mkdir -p $BINDIR
mkdir -p $SHAREDIR/templates/nginx
mkdir -p $SHAREDIR/templates/ssl
mkdir -p $CONFDIR/skel

# Instalar binario
cp bin/sm $BINDIR/
chmod +x $BINDIR/sm

# Instalar templates
cp templates/nginx/*.tmpl $SHAREDIR/templates/nginx/
cp templates/ssl/*.tmpl $SHAREDIR/templates/ssl/

# Crear enlace simbólico si no existe
if [ ! -L /usr/bin/sm ]; then
    ln -s $BINDIR/sm /usr/bin/sm
fi

echo "✅ SiteManager instalado correctamente"
echo "Ejecuta 'sudo sm status' para verificar el sistema"
EOF

chmod +x dist/sitemanager-1.0.0/install.sh

# Crear paquete
cd dist
tar -czf sitemanager-1.0.0-linux-amd64.tar.gz sitemanager-1.0.0/
cd ..
```

### 2. Script de Instalación Automática

```bash
# Crear script de instalación desde URL
cat > install-sitemanager.sh << 'EOF'
#!/bin/bash

set -e

# Configuración
REPO="elmersh/sitemanager"
VERSION="latest"
ARCH="amd64"
INSTALL_DIR="/tmp/sitemanager-install"

# Detectar arquitectura
if [ "$(uname -m)" = "aarch64" ]; then
    ARCH="arm64"
fi

# Función de limpieza
cleanup() {
    rm -rf "$INSTALL_DIR"
}
trap cleanup EXIT

# Verificar permisos
if [ "$EUID" -ne 0 ]; then
    echo "❌ Este script debe ejecutarse con sudo"
    exit 1
fi

echo "🚀 Instalando SiteManager..."

# Crear directorio temporal
mkdir -p "$INSTALL_DIR"
cd "$INSTALL_DIR"

# Descargar la última release
if [ "$VERSION" = "latest" ]; then
    DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/sitemanager-linux-$ARCH.tar.gz"
else
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/v$VERSION/sitemanager-linux-$ARCH.tar.gz"
fi

echo "📥 Descargando desde $DOWNLOAD_URL..."
curl -L -o sitemanager.tar.gz "$DOWNLOAD_URL"

# Extraer y ejecutar instalación
tar -xzf sitemanager.tar.gz
cd sitemanager-*/
chmod +x install.sh
./install.sh

echo "✅ SiteManager instalado correctamente"
echo "📖 Ejecuta 'sudo sm status' para verificar el sistema"
EOF

chmod +x install-sitemanager.sh
```

## Automatización con GitHub Actions

### 1. Crear Workflow de Build

```bash
mkdir -p .github/workflows
cat > .github/workflows/build-and-release.yml << 'EOF'
name: Build and Release

on:
  push:
    tags:
      - 'v*'
  pull_request:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        goos: [linux]
        goarch: [amd64, arm64]

    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.23'

    - name: Install dependencies
      run: make deps

    - name: Run tests
      run: make test

    - name: Build binary
      run: |
        GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} go build \
          -ldflags "-X main.Version=${GITHUB_REF#refs/tags/v}" \
          -o sm-${{ matrix.goos }}-${{ matrix.goarch }} \
          cmd/sm/main.go

    - name: Create package
      run: |
        mkdir -p dist/sitemanager-${GITHUB_REF#refs/tags/v}
        cp sm-${{ matrix.goos }}-${{ matrix.goarch }} dist/sitemanager-${GITHUB_REF#refs/tags/v}/sm
        cp -r internal/templates dist/sitemanager-${GITHUB_REF#refs/tags/v}/
        
        # Crear script de instalación
        cat > dist/sitemanager-${GITHUB_REF#refs/tags/v}/install.sh << 'INSTALL_EOF'
        #!/bin/bash
        set -e
        if [ "$EUID" -ne 0 ]; then
            echo "Este script debe ejecutarse con sudo"
            exit 1
        fi
        
        PREFIX=${PREFIX:-/usr/local}
        BINDIR=$PREFIX/bin
        SHAREDIR=$PREFIX/share/sitemanager
        
        mkdir -p $BINDIR $SHAREDIR/templates/nginx $SHAREDIR/templates/ssl /etc/sitemanager/skel
        cp sm $BINDIR/
        chmod +x $BINDIR/sm
        cp -r templates/* $SHAREDIR/templates/
        
        if [ ! -L /usr/bin/sm ]; then
            ln -s $BINDIR/sm /usr/bin/sm
        fi
        
        echo "✅ SiteManager instalado correctamente"
        INSTALL_EOF
        
        chmod +x dist/sitemanager-${GITHUB_REF#refs/tags/v}/install.sh
        
        cd dist
        tar -czf sitemanager-${{ matrix.goos }}-${{ matrix.goarch }}.tar.gz sitemanager-${GITHUB_REF#refs/tags/v}/

    - name: Upload artifacts
      uses: actions/upload-artifact@v3
      with:
        name: sitemanager-${{ matrix.goos }}-${{ matrix.goarch }}
        path: dist/sitemanager-${{ matrix.goos }}-${{ matrix.goarch }}.tar.gz

  release:
    if: startsWith(github.ref, 'refs/tags/')
    needs: build
    runs-on: ubuntu-latest
    
    steps:
    - name: Download artifacts
      uses: actions/download-artifact@v3
      
    - name: Create Release
      uses: softprops/action-gh-release@v1
      with:
        files: |
          */sitemanager-*.tar.gz
        generate_release_notes: true
        draft: false
        prerelease: false
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
EOF
```

## Distribución y Publicación

### 1. Publicación en GitHub Releases

```bash
# Crear y subir un tag
git tag v1.0.0
git push origin v1.0.0

# GitHub Actions se encargará automáticamente de:
# - Compilar para múltiples arquitecturas
# - Crear paquetes TAR.GZ
# - Publicar en GitHub Releases
```

### 2. Script de Instalación Rápida

Los usuarios podrán instalar con un solo comando:

```bash
# Instalación directa desde GitHub
curl -fsSL https://raw.githubusercontent.com/elmersh/sitemanager/main/install-sitemanager.sh | sudo bash

# O descarga manual
wget https://github.com/elmersh/sitemanager/releases/latest/download/sitemanager-linux-amd64.tar.gz
tar -xzf sitemanager-linux-amd64.tar.gz
cd sitemanager-*/
sudo ./install.sh
```

### 3. Verificación Post-instalación

```bash
# Verificar instalación
sudo sm status

# Verificar versión
sm --version

# Ver ayuda
sm --help
```

## Actualización de Versiones

### 1. Proceso de Release

```bash
# Actualizar versión en el código
sed -i 's/var version = "[^"]*"/var version = "1.1.0"/' cmd/sm/main.go

# Crear changelog
git log --oneline v1.0.0..HEAD > CHANGELOG-v1.1.0.md

# Commit y tag
git add .
git commit -m "Release v1.1.0"
git tag v1.1.0
git push origin main v1.1.0
```

### 2. Script de Actualización

```bash
cat > update-sitemanager.sh << 'EOF'
#!/bin/bash
set -e

echo "🔄 Actualizando SiteManager..."

# Verificar instalación actual
if ! command -v sm &> /dev/null; then
    echo "❌ SiteManager no está instalado"
    exit 1
fi

echo "📋 Versión actual: $(sm --version)"

# Descargar e instalar nueva versión
curl -fsSL https://raw.githubusercontent.com/elmersh/sitemanager/main/install-sitemanager.sh | sudo bash

echo "✅ Actualización completada"
echo "📋 Nueva versión: $(sm --version)"
EOF

chmod +x update-sitemanager.sh
```

## Notas de Desarrollo

- Los binarios se compilan estáticamente sin dependencias externas
- Las plantillas se embeben en el binario durante la compilación
- Se requiere sudo para la instalación y operación
- Compatible con Ubuntu 18.04+, Debian 9+ y derivados
- Soporta arquitecturas AMD64 y ARM64

Para cualquier problema durante la compilación o instalación, revisa los logs y asegúrate de que todas las dependencias estén instaladas correctamente.