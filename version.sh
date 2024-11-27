#!/bin/bash

set -e                                                        # Detener la ejecución en caso de error

HELP=false
MAYOR=false
MINOR=false
INDEX=2
REBUILD=false
CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0") # Obtener la versión actual de Git

# Parsear opciones
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --h | --help) HELP=true ;;                             # Activar la bandera si se proporciona --help
        --m | --major) MAYOR=true ;;                          # Activar la bandera si se proporciona --major
        --n | --minor) MINOR=true ;;                          # Activar la bandera si se proporciona --minor
        *) echo "Opción desconocida: $1"; exit 1 ;;
    esac
    shift
done

# Mostrar las opciones elegidas
echo "Opciones elegidas:"
[[ "$MAYOR" == true ]] && echo " - Major: Activado"
[[ "$MINOR" == true ]] && echo " - Minor: Activado"

version() {
  echo "Versión actual: $CURRENT_VERSION"
  echo "Nueva versión: $new_version"
  echo "Etiquetando con: $new_version"
  git add .
  git commit -m 'Update'
  git push -u origin
  git tag "$new_version"
  git push origin "$new_version"
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
  exit 0
elif [ "$CURRENT_VERSION" == "v0.0.0" ]; then
  new_version="v1.0.0"
  version
else
  # Obtiene la última etiqueta
  latest_tag=$(git describe --tags --abbrev=0 2>/dev/null)

  # Divide la etiqueta en componentes usando el punto como delimitador
  IFS='.' read -r -a version_parts <<< "${latest_tag#v}"

  if [ "$MAYOR" == true ]; then
    # Si se proporciona la opción --major, incrementa el valor de la posición 0
    version_parts[0]=$((version_parts[0] + 1))
    version_parts[1]=0
    version_parts[2]=0

    new_version="v${version_parts[0]}.${version_parts[1]}.${version_parts[2]}"    
  elif [ "$MINOR" == true ]; then
    # Si se proporciona la opción --minor, incrementa el valor de la posición 1        
    version_parts[1]=$((version_parts[1] + 1))
    version_parts[2]=0

    new_version="v${version_parts[0]}.${version_parts[1]}.${version_parts[2]}"    
  else
    # Incrementa el valor de la posición 2
    version_parts[2]=$((version_parts[2] + 1))

    # Reconstruye la nueva versión (X.Y.Z) y prepende la 'v' al principio
    new_version="v${version_parts[0]}.${version_parts[1]}.${version_parts[2]}"    
  fi

  version
fi

# Línea en blanco al final