#!/bin/bash

K0SCTL_TEMPLATE=${K0SCTL_TEMPLATE:-"k0sctl.yaml.tpl"}

set -e

. ./smoke.common.sh
trap cleanup EXIT

envsubst < k0sctl-files.yaml.tpl > k0sctl.yaml

deleteCluster
createCluster

remoteFileExist() {
  local userhost="$1"
  local path="$2"
  footloose ssh "${userhost}" -- test -e "${path}"
}

remoteFileContent() {
  local userhost="$1"
  local path="$2"
  footloose ssh "${userhost}" -- cat "${path}"
}

echo "* Creating random files"
mkdir -p upload
mkdir -p upload/nested

head -c 8192 </dev/urandom > upload/toplevel.txt
head -c 8192 </dev/urandom > upload/nested/nested.txt
head -c 8192 </dev/urandom > upload/nested/exclude-on-glob

echo "* Starting apply"
../k0sctl apply --config k0sctl.yaml --debug

echo "* Apply OK"
echo "* Verifying files"

echo -n "  - Single file using destination file path .."
remoteFileExist root@manager0 /root/singlefile/renamed.txt
echo "OK"

echo -n "  - Single file using destination dir .."
remoteFileExist root@manager0 /root/singlefile/toplevel.txt
echo "OK"

echo -n "  - Directory using destination dir .."
remoteFileExist root@manager0 /root/dir/toplevel.txt
remoteFileExist root@manager0 /root/dir/nested/nested.txt
remoteFileExist root@manager0 /root/dir/nested/exclude-on-glob
echo "OK"

echo -n "  - Glob using destination dir .."
remoteFileExist root@manager0 /root/glob/toplevel.txt
remoteFileExist root@manager0 /root/glob/nested/nested.txt
remoteFileExist root@manager0 /root/glob/nested/exclude-on-glob && false
echo "OK"

echo -n "  - URL using destination file .."
remoteFileExist root@manager0 /root/url/releases.json
remoteFileContent root@manager0 /root/url/releases.json | grep -q html_url
echo "OK"

echo -n "  - URL using destination dir .."
remoteFileExist root@manager0 /root/url/releases
remoteFileContent root@manager0 /root/url/releases | grep -q html_url
echo "OK"

echo "* Done"

