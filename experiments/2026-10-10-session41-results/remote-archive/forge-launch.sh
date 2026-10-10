set -eu
S=/work/lanes/ngn-data-volume-20261010
cd "$S"
setsid nohup systemd-run --user --scope --unit=ngn-s41-archive-server -p AllowedCPUs=8-11 -p CPUQuota=25% -p MemoryMax=128M -p MemorySwapMax=0 nice -n 19 python3 archive_server.py --root "$S" --token-file "$S/token" --bind 192.168.4.65 --port 8766 --uploads < /dev/null > server.log 2>&1 &
echo $! > server.pid
setsid nohup systemd-run --user --scope --unit=ngn-s41-t74-convert -p AllowedCPUs=8-11 -p CPUQuota=200% -p MemoryMax=2G -p MemorySwapMax=0 nice -n 19 bash "$S/controller.sh" < /dev/null > controller.log 2>&1 &
echo $! > controller.pid
cat server.pid controller.pid
