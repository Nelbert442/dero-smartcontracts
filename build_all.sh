#!/usr/bin/env bash

CURDIR=`/bin/pwd`
BASEDIR=$(dirname $0)
ABSPATH=$(readlink -f $0)
ABSDIR=$(dirname $ABSPATH)


unset GOPATH

version="1.0.0"


cd $CURDIR
bash $ABSDIR/build_package.sh "github.com/Nelbert442/dero-smartcontracts/DERO-Dice/cmd/client-service"
bash $ABSDIR/build_package.sh "github.com/Nelbert442/dero-smartcontracts/DERO-Faucet/cmd/server-service"
#bash $ABSDIR/build_package.sh "github.com/Nelbert442/dero-smartcontracts/cmd/server-service"


for d in build/*; do cp Start.md "$d"; done
cd "${ABSDIR}/build"

#windows users require zip files
#zip -r dero_windows_amd64_$version.zip dero_windows_amd64
zip -r dero_windows_amd64.zip dero_windows_amd64
zip -r dero_windows_x86.zip dero_windows_386
zip -r dero_windows_386.zip dero_windows_386
zip -r dero_windows_amd64_$version.zip dero_windows_amd64
zip -r dero_windows_x86_$version.zip dero_windows_386
zip -r dero_windows_386_$version.zip dero_windows_386

#all other platforms are okay with tar.gz
find . -mindepth 1 -type d -not -name '*windows*'   -exec tar -cvzf {}.tar.gz {} \;
find . -mindepth 1 -type d -not -name '*windows*'   -exec tar -cvzf {}_$version.tar.gz {} \;

cd $CURDIR
