#!/bin/bash

# Verificar si se proporcionó el archivo tar.gz como argumento
if [ $# -ne 1 ]; then
    echo "Uso: $0 <archivo_tar.gz>"
    exit 1
fi

TAR_FILE=$1

# Verificar si el archivo existe
if [ ! -f "$TAR_FILE" ]; then
    echo "Error: El archivo $TAR_FILE no existe"
    exit 1
fi

# Crear directorio temporal para la instalación
TEMP_DIR=$(mktemp -d)
echo "Creando directorio temporal: $TEMP_DIR"

# Descomprimir el archivo tar.gz
echo "Descomprimiendo $TAR_FILE..."
tar -xzf "$TAR_FILE" -C "$TEMP_DIR"

# Verificar si la descompresión fue exitosa
if [ $? -ne 0 ]; then
    echo "Error al descomprimir el archivo"
    rm -rf "$TEMP_DIR"
    exit 1
fi

# Eliminar la instalación anterior si existe
if [ -f /usr/local/bin/sm ]; then
    echo "Eliminando instalación anterior..."
    rm /usr/local/bin/sm
fi

# Copiar el nuevo binario
echo "Instalando nueva versión..."
cp "$TEMP_DIR/sitemanager/sm" /usr/local/bin/
chmod +x /usr/local/bin/sm

# Limpiar
echo "Limpiando archivos temporales..."
rm -rf "$TEMP_DIR"

echo "SiteManager actualizado exitosamente en /usr/local/bin/sm"
