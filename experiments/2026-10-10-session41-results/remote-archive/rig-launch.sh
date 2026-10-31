set -eu
R=/home/ehrli/ngn-data/night-20261007
S=$R/session41-remote-archive
mkdir -p "$S/source"
tar -xf /mnt/c/Users/ehrli/ngn-session41-remote-source.tar -C "$S/source"
cp /mnt/c/Users/ehrli/ngn-session41-token "$S/token"
chmod 600 "$S/token"
cp /mnt/c/Users/ehrli/ngn-session41-checks.sh "$S/checks.sh"
cp /mnt/c/Users/ehrli/ngn-session41-transfer.sh "$S/transfer.sh"
cd "$S"
setsid nohup systemd-run --user --scope --unit=ngn-s41-remote-checks -p AllowedCPUs=12,14 -p CPUQuota=200% -p MemoryMax=2500M -p MemorySwapMax=0 bash checks.sh < /dev/null > checks.log 2>&1 &
echo $! > checks.pid
setsid nohup systemd-run --user --scope --unit=ngn-s41-t74-transfer -p AllowedCPUs=12,14 -p CPUQuota=25% -p MemoryMax=128M -p MemorySwapMax=0 bash transfer.sh < /dev/null > transfer.log 2>&1 &
echo $! > transfer.pid
cat checks.pid transfer.pid
