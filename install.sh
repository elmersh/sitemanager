#!/bin/bash

# Script de instalación de SiteManager
# Uso: curl -fsSL https://raw.githubusercontent.com/elmersh/sitemanager/main/install.sh | sudo bash

set -e

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuración
REPO="elmersh/sitemanager"
BINARY_NAME="sm"
INSTALL_DIR="/usr/local/bin"
SHARE_DIR="/usr/local/share/sitemanager"
CONFIG_DIR="/etc/sitemanager"
GITHUB_API="https://api.github.com/repos/$REPO"

# Función para mostrar mensajes
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Función de limpieza
cleanup() {
    if [ -d "$TMP_DIR" ]; then
        rm -rf "$TMP_DIR"
    fi
}
trap cleanup EXIT

# Verificar permisos de sudo
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "Este script debe ejecutarse con permisos de sudo"
        echo "Uso: curl -fsSL https://raw.githubusercontent.com/elmersh/sitemanager/main/install.sh | sudo bash"
        exit 1
    fi
}

# Detectar arquitectura del sistema
detect_arch() {
    local arch
    arch=$(uname -m)
    case $arch in
        x86_64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            log_error "Arquitectura no soportada: $arch"
            log_info "SiteManager soporta: amd64, arm64"
            exit 1
            ;;
    esac
}

# Detectar sistema operativo
detect_os() {
    local os
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case $os in
        linux*)
            echo "linux"
            ;;
        *)
            log_error "Sistema operativo no soportado: $os"
            log_info "SiteManager solo soporta Linux (Ubuntu/Debian)"
            exit 1
            ;;
    esac
}

# Verificar dependencias del sistema
check_dependencies() {
    log_info "Verificando dependencias básicas..."
    
    local missing_critical=()
    local missing_optional=()
    
    # Verificar dependencias críticas para la instalación
    if ! command -v curl >/dev/null 2>&1; then
        missing_critical+=("curl")
    fi
    
    if ! command -v tar >/dev/null 2>&1; then
        missing_critical+=("tar")
    fi
    
    # Verificar dependencias opcionales de SiteManager
    log_info "Verificando dependencias de SiteManager..."
    
    if ! command -v nginx >/dev/null 2>&1; then
        missing_optional+=("nginx - requerido para crear sitios web")
    fi
    
    if ! command -v php-fpm >/dev/null 2>&1; then
        missing_optional+=("php-fpm - requerido para sitios Laravel/PHP")
    fi
    
    if ! command -v node >/dev/null 2>&1; then
        missing_optional+=("node - requerido para sitios Node.js")
    fi
    
    if ! command -v pm2 >/dev/null 2>&1; then
        missing_optional+=("pm2 - requerido para gestión de procesos Node.js")
    fi
    
    if ! command -v certbot >/dev/null 2>&1; then
        missing_optional+=("certbot - requerido para certificados SSL")
    fi
    
    if ! command -v composer >/dev/null 2>&1; then
        missing_optional+=("composer - requerido para proyectos Laravel")
    fi
    
    # Solo instalar dependencias críticas automáticamente
    if [ ${#missing_critical[@]} -ne 0 ]; then
        log_info "Instalando dependencias críticas: ${missing_critical[*]}"
        
        if command -v apt >/dev/null 2>&1; then
            apt update
            apt install -y "${missing_critical[@]}"
        elif command -v yum >/dev/null 2>&1; then
            yum install -y "${missing_critical[@]}"
        else
            log_error "No se pudo detectar el gestor de paquetes"
            log_info "Instala manualmente: ${missing_critical[*]}"
            exit 1
        fi
    fi
    
    # Mostrar dependencias opcionales faltantes sin instalar
    if [ ${#missing_optional[@]} -ne 0 ]; then
        log_warning "Dependencias opcionales no instaladas:"
        for dep in "${missing_optional[@]}"; do
            echo "   ⚠️  $dep"
        done
        echo ""
        
        # Guardar para mostrar al final
        MISSING_DEPS=("${missing_optional[@]}")
    fi
}

# Obtener la última versión desde GitHub
get_latest_version() {
    local api_response
    if ! api_response=$(curl -s "$GITHUB_API/releases/latest"); then
        log_error "No se pudo conectar a GitHub API"
        exit 1
    fi
    
    local version
    version=$(echo "$api_response" | grep '"tag_name":' | cut -d'"' -f4)
    
    if [ -z "$version" ]; then
        log_error "No se pudo obtener la versión desde GitHub"
        exit 1
    fi
    
    echo "$version"
}

# Descargar SiteManager
download_sitemanager() {
    local version="$1"
    local os="$2"
    local arch="$3"
    
    local version_clean="${version#v}"  # Remover 'v' del inicio
    local filename="sitemanager-$version_clean-$os-$arch.tar.gz"
    local download_url="https://github.com/$REPO/releases/download/$version/$filename"
    
    log_info "Descargando SiteManager $version para $os/$arch..." >&2
    log_info "URL: $download_url" >&2
    
    TMP_DIR=$(mktemp -d)
    local tar_file="$TMP_DIR/$filename"
    
    if ! curl -L -o "$tar_file" "$download_url" 2>&1; then
        log_error "Falló la descarga de $filename" >&2
        log_info "Verifica que la versión $version esté disponible para tu arquitectura" >&2
        exit 1
    fi
    
    log_success "Descarga completada" >&2
    echo "$tar_file"
}

# Extraer e instalar SiteManager
install_sitemanager() {
    local tar_file="$1"
    local extract_dir="$TMP_DIR/extract"
    
    log_info "Extrayendo SiteManager..."
    
    mkdir -p "$extract_dir"
    if ! tar -xzf "$tar_file" -C "$extract_dir" --strip-components=1; then
        log_error "Falló la extracción del archivo"
        exit 1
    fi
    
    # Verificar que el binario existe
    local binary_path="$extract_dir/bin/$BINARY_NAME"
    if [ ! -f "$binary_path" ]; then
        log_error "No se encontró el binario en el paquete descargado"
        exit 1
    fi
    
    log_info "Instalando SiteManager..."
    
    # Crear directorios necesarios
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$SHARE_DIR/templates/nginx"
    mkdir -p "$SHARE_DIR/templates/ssl"
    mkdir -p "$CONFIG_DIR/skel"
    
    # Instalar binario
    cp "$binary_path" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    
    # Instalar templates
    if [ -d "$extract_dir/templates" ]; then
        cp -r "$extract_dir/templates"/* "$SHARE_DIR/templates/"
    fi
    
    # Crear enlace simbólico en /usr/bin si no existe
    if [ ! -L "/usr/bin/$BINARY_NAME" ]; then
        ln -s "$INSTALL_DIR/$BINARY_NAME" "/usr/bin/$BINARY_NAME"
    fi
    
    log_success "SiteManager instalado correctamente"
}

# Verificar instalación
verify_installation() {
    log_info "Verificando instalación..."
    
    if ! command -v "$BINARY_NAME" >/dev/null 2>&1; then
        log_error "SiteManager no se encuentra en el PATH"
        exit 1
    fi
    
    local version_output
    if version_output=$("$BINARY_NAME" --version 2>/dev/null); then
        log_success "SiteManager instalado: $version_output"
    else
        log_error "SiteManager instalado pero no funciona correctamente"
        exit 1
    fi
}

# Mostrar información post-instalación
show_post_install_info() {
    echo ""
    log_success "🎉 ¡SiteManager instalado exitosamente!"
    echo ""
    echo "📋 Próximos pasos:"
    echo ""
    echo "1️⃣  Verificar el sistema:"
    echo "   sudo $BINARY_NAME status"
    echo ""
    echo "2️⃣  Configurar tu email (requerido para SSL):"
    echo "   Edita: ~/.config/sitemanager/config.yaml"
    echo "   Establece: email: tu@email.com"
    echo "   Establece: agree_tos: true"
    echo ""
    echo "3️⃣  Crear tu primer sitio:"
    echo "   sudo $BINARY_NAME site -d ejemplo.com -t laravel"
    echo ""
    echo "4️⃣  Ver ayuda completa:"
    echo "   $BINARY_NAME --help"
    echo ""
    echo "🔗 Documentación: https://github.com/$REPO"
    echo "🐛 Reportar problemas: https://github.com/$REPO/issues"
    echo ""
    
    # Mostrar dependencias faltantes detectadas durante la instalación
    if [ ${#MISSING_DEPS[@]} -ne 0 ]; then
        echo "⚠️  Dependencias opcionales no instaladas:"
        for dep in "${MISSING_DEPS[@]}"; do
            echo "   • $dep"
        done
        echo ""
        echo "💡 Comandos de instalación sugeridos:"
        echo ""
        
        if command -v apt >/dev/null 2>&1; then
            echo "   # Nginx (requerido para todos los sitios)"
            echo "   sudo apt update && sudo apt install nginx"
            echo ""
            echo "   # Para sitios Laravel/PHP"
            echo "   sudo apt install php8.3-fpm php8.3-mysql php8.3-xml php8.3-curl composer"
            echo ""
            echo "   # Para sitios Node.js"
            echo "   curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -"
            echo "   sudo apt install nodejs"
            echo "   sudo npm install -g pm2"
            echo ""
            echo "   # Para certificados SSL"
            echo "   sudo apt install certbot python3-certbot-nginx"
        elif command -v yum >/dev/null 2>&1; then
            echo "   # Paquetes básicos"
            echo "   sudo yum install nginx php-fpm php-mysql composer certbot"
            echo "   sudo npm install -g pm2  # Después de instalar Node.js"
        fi
        echo ""
        echo "📋 Nota: SiteManager detectará automáticamente las dependencias"
        echo "    cuando intentes crear sitios específicos."
        echo ""
    fi
}

# Función principal
main() {
    echo ""
    log_info "🚀 Instalador de SiteManager"
    echo ""
    
    # Verificaciones iniciales
    check_root
    check_dependencies
    
    # Detectar sistema
    local os arch version
    os=$(detect_os)
    arch=$(detect_arch)
    log_info "Obteniendo información de la última versión..."
    version=$(get_latest_version)
    
    log_info "Sistema detectado: $os/$arch"
    log_info "Versión a instalar: $version"
    
    # Descargar e instalar
    local tar_file
    tar_file=$(download_sitemanager "$version" "$os" "$arch")
    install_sitemanager "$tar_file"
    verify_installation
    show_post_install_info
}

# Ejecutar instalación
main "$@"