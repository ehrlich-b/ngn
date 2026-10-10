set -eu
cd /work/lanes/ngn-data-volume-20261010
test -f transfer.done
pid=$(cat server.pid)
test "$(ps -p "$pid" -o args=)" = 'python3 archive_server.py --root /work/lanes/ngn-data-volume-20261010 --token-file /work/lanes/ngn-data-volume-20261010/token --bind 192.168.4.65 --port 8766 --uploads'
kill -TERM "$pid"
date -Is > upload-server.stopped
cat controller.pid reader-server.pid
