set -eu
S=/work/lanes/ngn-data-volume-20261010
cd "$S"
setsid nohup systemd-run --user --scope --unit=ngn-s41-archive-reader -p AllowedCPUs=8-11 -p CPUQuota=50% -p MemoryMax=128M -p MemorySwapMax=0 nice -n 19 python3 archive_server_read.py --root "$S" --token-file "$S/token" --bind 192.168.4.65 --port 8767 < /dev/null > reader-server.log 2>&1 &
echo $! > reader-server.pid
cat reader-server.pid
