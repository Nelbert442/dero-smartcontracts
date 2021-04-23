#!/usr/bin/env bash

CURDIR=`/bin/pwd`
BASEDIR=$(dirname $0)
ABSPATH=$(readlink -f $0)
ABSDIR=$(dirname $ABSPATH)


unset GOPATH

version="1.0.0"


cd $CURDIR
bash $ABSDIR/build_package.sh "github.com/Nelbert442/dero-smartcontracts/DERO-Faucet/cmd/server-service"

cd "cmd/server-service"
for d in $ABSDIR/build/*; do cp -R site/. "$d/site"; done
cd "${ABSDIR}/build"

#copy site contents into folders
#find . -mindepth 1 -type d  -exec cp -R github.com/Nelbert442/dero-smartcontracts/DERO-Faucet/cmd/server-service/site/. {}/site/ \;

#windows users require zip files
#zip -r derofaucet_windows_amd64.zip derofaucet_windows_amd64
#zip -r derofaucet_windows_x86.zip derofaucet_windows_386
#zip -r derofaucet_windows_386.zip derofaucet_windows_386
zip -r derofaucet_windows_amd64_$version.zip derofaucet_windows_amd64
zip -r derofaucet_windows_x86_$version.zip derofaucet_windows_386
zip -r derofaucet_windows_386_$version.zip derofaucet_windows_386

#all other platforms are okay with tar.gz
#find . -mindepth 1 -maxdepth 1 -type d -not -name '*windows*'   -exec tar -cvzf {}.tar.gz {} \;
find . -mindepth 1 -maxdepth 1 -type d -not -name '*windows*'   -exec tar -cvzf {}_$version.tar.gz {} \;

cd $CURDIR
