#!/bin/bash
set -euo pipefail

# Variables
topdir=$(pwd)/rpmbuild
go_binary=enigma-agent
version=${1:-0.1.0}
release=1
arch=$(uname -m)
spec_file=$(pwd)/enigma-agent.spec

# Clean build dir
rm -rf "$topdir"
mkdir -p "$topdir"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

# Build Go binary
cd ../../..
go build -o "$topdir/SOURCES/$go_binary" ./cmd/enigma-agent
cd - > /dev/null

# Copy config example
cp ../../../config.example.json "$topdir/SOURCES/config.json"

# Build RPM
rpmbuild --define "_topdir $topdir" \
         --define "_version $version" \
         --define "_release $release" \
         --define "_arch $arch" \
         -bb "$spec_file"

# Output location
echo "RPM built at: $topdir/RPMS/$arch/enigma-agent-$version-$release.$arch.rpm"