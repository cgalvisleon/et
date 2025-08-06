#!/bin/bash

set -e                                                        # Detener la ejecución en caso de error

HELP=false
MAYOR=false
MINOR=false
VERSION=false
INDEX=2
CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0") # Obtener la versión actual de Git
NEW_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0") # Obtener la versión actual de Git

# Parsear opciones
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --h | --help) HELP=true ;;                             # Activar la bandera si se proporciona --help
        --m | --major) MAYOR=true ;;                          # Activar la bandera si se proporciona --major
        --n | --minor) MINOR=true ;;                          # Activar la bandera si se proporciona --minor
        --v | --version) VERSION=true ;;                            # Activar la bandera si se proporciona --push
        *) echo "Opción desconocida: $1"; exit 1 ;;
    esac
    shift
done

# Mostrar las opciones elegidas
echo "Opciones elegidas:"
[[ "$MAYOR" == true ]] && echo " - Major: Activado"
[[ "$MINOR" == true ]] && echo " - Minor: Activado"
[[ "$VERSION" == true ]] && echo " - Version: Activado"

build_version() {
  # Obtiene la última etiqueta
  latest_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

  # Divide la etiqueta en componentes usando el punto como delimitador
  IFS='.' read -r -a version_parts <<< "${latest_tag#v}"

  if [ "$MAYOR" == true ]; then
    # Si se proporciona la opción --major, incrementa el valor de la posición 0
    version_parts[0]=$((version_parts[0] + 1))
    version_parts[1]=0
    version_parts[2]=0

    NEW_VERSION="v${version_parts[0]}.${version_parts[1]}.${version_parts[2]}"    
  elif [ "$MINOR" == true ]; then
    # Si se proporciona la opción --minor, incrementa el valor de la posición 1        
    version_parts[1]=$((version_parts[1] + 1))
    version_parts[2]=0

    NEW_VERSION="v${version_parts[0]}.${version_parts[1]}.${version_parts[2]}"    
  else
    # Incrementa el valor de la posición 2
    version_parts[2]=$((version_parts[2] + 1))

    # Reconstruye la nueva versión (X.Y.Z) y prepende la 'v' al principio
    NEW_VERSION="v${version_parts[0]}.${version_parts[1]}.${version_parts[2]}"
  fi
}

update_version() {
  echo "Versión actual: $CURRENT_VERSION"
  echo "Nueva versión: $NEW_VERSION"
  echo "Etiquetando con: $NEW_VERSION"


  sed -i "" "s/$CURRENT_VERSION/$NEW_VERSION/g" README.md
  
  git tag "$NEW_VERSION"
  git push origin --tags  

  echo "Etiqueta creada y enviada a Git"
}

version() {
  echo "Etiquetando con: $NEW_VERSION"

  sed -i "" "s/$CURRENT_VERSION/$NEW_VERSION/g" README.md
  
  git tag "$NEW_VERSION"
  git push -u origin --tags
  
  echo "Etiqueta creada y enviada a Git"
}

if [ "$HELP" == true ]; then
  echo "Uso: ./version.sh [opciones]"
  echo "Incrementa la versión de la etiqueta de Git"
  echo ""
  echo "Opciones:"
  echo "  --h, --help     Muestra este mensaje de ayuda"
  echo "  --m, --major    Incrementa la versión mayor"
  echo "  --n, --minor    Incrementa la versión menor"
  echo "  --v, --version  Incrementa la versión de la revisión"
  exit 0
elif [ "$VERSION" == true ]; then
  build_version
  version
else
  build_version
  update_version
fi

# Línea en blanco al final