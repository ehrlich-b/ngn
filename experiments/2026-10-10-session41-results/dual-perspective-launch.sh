set -eu
R=/home/ehrli/ngn-data/night-20261007
S=$R/session41-dual-perspective
mkdir "$S"
cp /mnt/c/Users/ehrli/ngn-session41-base.tar "$S/base.tar"
cp /mnt/c/Users/ehrli/ngn-session41-pair.patch "$S/candidate.patch"
cp /mnt/c/Users/ehrli/ngn-session41-pair-controller.sh "$S/controller.sh"
cp /mnt/c/Users/ehrli/ngn-session41-pair-check.py "$S/check.py"
cp "$R/session40-move-select/baseline-probe" "$S/baseline-probe"
cd "$S"
setsid nohup systemd-run --user --scope --unit=ngn-s41-dual-perspective -p AllowedCPUs=12,14 -p CPUQuota=200% -p MemoryMax=2500M -p MemorySwapMax=0 bash controller.sh < /dev/null > controller.log 2>&1 &
echo $! > controller.pid
cat controller.pid
