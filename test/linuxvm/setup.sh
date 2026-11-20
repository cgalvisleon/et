#!/bin/bash
# Crea (si falta) y arranca la VM Linux `etstore` en el equipo IA, e instala
# sus herramientas. Idempotente: se puede correr las veces que se quiera.
# Uso (desde esta Mac):  ./test/linuxvm/setup.sh [--smoke] [--lazyfs]
#   --smoke   prueba de humo de dm-flakey dentro de la VM
#   --lazyfs  intenta compilar LazyFS dentro de la VM (en ~/lazyfs)
# La VM queda encendida; deténla con: ssh ia limactl stop etstore
set -euo pipefail

HOST=${ETSTORE_HOST:-ia}
VM=etstore
DIR=$(cd "$(dirname "$0")" && pwd)
SSH=(ssh -o BatchMode=yes "$HOST")
REMOTE_PATH='export PATH=/opt/homebrew/bin:$PATH'

smoke=0; lazyfs=0
for a in "$@"; do
  case "$a" in
    --smoke) smoke=1 ;;
    --lazyfs) lazyfs=1 ;;
    *) echo "argumento desconocido: $a" >&2; exit 2 ;;
  esac
done

echo "== Lima en $HOST"
"${SSH[@]}" "$REMOTE_PATH; command -v limactl >/dev/null || brew install lima; limactl --version"

echo "== definición"
"${SSH[@]}" 'mkdir -p ~/etstore'
scp -q -o BatchMode=yes "$DIR/etstore.yaml" "$HOST:etstore/etstore.yaml"

echo "== crear / arrancar $VM"
"${SSH[@]}" "$REMOTE_PATH
if ! limactl list -q 2>/dev/null | grep -qx $VM; then
  limactl create --tty=false --name=$VM ~/etstore/etstore.yaml
fi
st=\$(limactl list --format '{{.Status}}' $VM)
[ \"\$st\" = Running ] || limactl start --tty=false $VM
# Reaplica el aprovisionamiento por si cambió el yaml (es idempotente).
limactl shell $VM -- bash -lc 'go version; systemctl is-active vigilante; dpkg -s dmsetup fuse3 libfuse3-dev build-essential >/dev/null && echo paquetes-ok'"

if [ $smoke = 1 ]; then
  echo "== prueba de humo de dm-flakey (dentro de la VM)"
  "${SSH[@]}" "$REMOTE_PATH; limactl shell $VM -- sudo bash -s" <<'SMOKE'
set -eux
cd /tmp
modprobe dm-flakey || true
truncate -s 64M flakey.img
LOOP=$(losetup -f --show flakey.img)
SECT=$(blockdev --getsz "$LOOP")
# Tabla flakey: up 60 s, down 0 s (siempre sano).
dmsetup create flakeytest --table "0 $SECT flakey $LOOP 0 60 0"
mkfs.ext4 -q /dev/mapper/flakeytest
mkdir -p /mnt/flakey
mount /dev/mapper/flakeytest /mnt/flakey
echo hola > /mnt/flakey/a.txt; sync
# Cambia a drop_writes: escrituras descartadas en silencio.
dmsetup suspend flakeytest
dmsetup load flakeytest --table "0 $SECT flakey $LOOP 0 0 60 1 drop_writes"
dmsetup resume flakeytest
echo perdido > /mnt/flakey/b.txt; sync || true
dmsetup status flakeytest
umount /mnt/flakey
dmsetup remove flakeytest
# Montado directo sobre el loop: a.txt sobrevive, b.txt no debió llegar.
mount "$LOOP" /mnt/flakey
test "$(cat /mnt/flakey/a.txt)" = hola
if [ -e /mnt/flakey/b.txt ]; then echo "b.txt llegó al disco: drop_writes no funcionó" >&2; exit 1; fi
umount /mnt/flakey
losetup -d "$LOOP"
rm -f flakey.img
echo DM-FLAKEY-OK
SMOKE
fi

if [ $lazyfs = 1 ]; then
  echo "== LazyFS (dentro de la VM, máx. 20 min)"
  "${SSH[@]}" "$REMOTE_PATH; limactl shell $VM -- bash -s" <<'LAZY'
set -eux
cd ~
[ -d lazyfs ] || git clone --depth 1 https://github.com/dsrhaslab/lazyfs.git
cd lazyfs
timeout 1200 bash -c 'cd libs/libpcache && ./build.sh && cd ../../lazyfs && ./build.sh'
ls -l lazyfs/build/lazyfs
# Prueba de humo: lo escrito sin fsync se pierde con clear-cache.
cd lazyfs
mountpoint -q /tmp/lfs.mnt && fusermount3 -u /tmp/lfs.mnt
rm -rf /tmp/lfs.mnt /tmp/lfs.root /tmp/lfs.fifo
mkdir -p /tmp/lfs.mnt /tmp/lfs.root
cat > /tmp/lfs.toml <<'T'
[faults]
fifo_path="/tmp/lfs.fifo"
[cache]
apply_eviction=false
[cache.simple]
custom_size="64mb"
blocks_per_page=1
[filesystem]
log_all_operations=false
logfile=""
T
./scripts/mount-lazyfs.sh -c /tmp/lfs.toml -m /tmp/lfs.mnt -r /tmp/lfs.root >/tmp/lfs.log 2>&1
for i in $(seq 20); do mountpoint -q /tmp/lfs.mnt && break; sleep 0.5; done
mountpoint -q /tmp/lfs.mnt || { cat /tmp/lfs.log; exit 1; }
echo durable > /tmp/lfs.mnt/a.txt
python3 -c 'import os;f=os.open("/tmp/lfs.mnt/a.txt",os.O_RDWR);os.fsync(f)'
echo volatil > /tmp/lfs.mnt/b.txt
timeout 10 bash -c 'echo lazyfs::clear-cache > /tmp/lfs.fifo'
sleep 1
a=$(cat /tmp/lfs.mnt/a.txt); b=$(stat -c %s /tmp/lfs.mnt/b.txt 2>/dev/null || echo ausente)
fusermount3 -u /tmp/lfs.mnt
echo "a.txt=$a b.txt(tamaño)=$b"
[ "$a" = durable ] && [ "$b" != 8 ] && echo LAZYFS-OK
LAZY
fi

echo "== listo. Detener: ssh $HOST limactl stop $VM"
