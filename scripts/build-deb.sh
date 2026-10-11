#!/usr/bin/env bash
# Build a Debian (amd64) .deb package for auFTsanitiser.
#
# Produces a self-contained, statically linked binary (no runtime dependencies) suitable
# for Debian 13 (trixie) amd64 and installs it at /usr/bin/auFTsanitiser.
#
# Uses dpkg-deb when available (e.g. on a Debian build host); otherwise constructs the .deb
# by hand with ar + tar (works on macOS/BSD), which dpkg can install.
set -euo pipefail

cd "$(dirname "$0")/.."

PKG="auftsanitiser"                  # Debian package name (lowercase)
BIN="auFTsanitiser"                  # installed executable name
# Single source of truth: internal/cli/version.go (override with VERSION=...).
VERSION="${VERSION:-$(sed -n 's/^const Version = "\(.*\)"$/\1/p' internal/cli/version.go)}"
[ -n "$VERSION" ] || { echo "error: cannot read Version from internal/cli/version.go" >&2; exit 1; }
ARCH="amd64"
MAINTAINER="${MAINTAINER:-Michael Paholski <mpaholski@me.com>}"

install -d dist
OUT="dist/${PKG}_${VERSION}_${ARCH}.deb"
STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

echo "==> building static linux/amd64 binary"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w" -o "$STAGE/$BIN" .

# Staging tree: pkgroot/{DEBIAN,usr/...}
ROOT="$STAGE/pkgroot"
install -d -m 0755 "$ROOT/DEBIAN" "$ROOT/usr/bin" "$ROOT/usr/share/doc/$PKG"
install -m 0755 "$STAGE/$BIN" "$ROOT/usr/bin/$BIN"

# Installed size in KiB (control field).
INSTALLED_KB="$(du -k "$ROOT/usr" | tail -1 | cut -f1)"

cat > "$ROOT/DEBIAN/control" <<EOF
Package: $PKG
Version: $VERSION
Section: utils
Priority: optional
Architecture: $ARCH
Maintainer: $MAINTAINER
Installed-Size: $INSTALLED_KB
Description: Plain-text clinical report sanitiser for AU HL7 v2.4 messaging
 Replaces UTF-8 / extended characters in a plain-text clinical report with
 clinically equivalent ASCII substitutions conforming to the Australian
 Diagnostics and Referral Messaging localisation of HL7 v2.4. Self-contained
 static binary with no runtime dependencies; intended to be called from a
 transformation workflow such as a Mirth Connect channel.
EOF

cat > "$ROOT/usr/share/doc/$PKG/copyright" <<EOF
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: $BIN

Files: *
Copyright: $MAINTAINER
License: proprietary
EOF
chmod 0644 "$ROOT/DEBIAN/control" "$ROOT/usr/share/doc/$PKG/copyright"

if command -v dpkg-deb >/dev/null 2>&1; then
    echo "==> packaging with dpkg-deb"
    dpkg-deb --root-owner-group --build "$ROOT" "$OUT"
elif command -v python3 >/dev/null 2>&1; then
    echo "==> dpkg-deb not found; constructing .deb with tar + python3 (ar writer)"
    BUILD="$STAGE/build"
    install -d "$BUILD"
    printf '2.0\n' > "$BUILD/debian-binary"
    # Root-owned, deterministic tarballs (libarchive/bsdtar flags).
    tar --numeric-owner --uid 0 --gid 0 --uname root --gname root \
        -czf "$BUILD/control.tar.gz" -C "$ROOT/DEBIAN" .
    tar --numeric-owner --uid 0 --gid 0 --uname root --gname root \
        -czf "$BUILD/data.tar.gz" -C "$ROOT" --exclude ./DEBIAN .
    rm -f "$OUT"
    # macOS `ar` insists on a symbol table and corrupts data archives, so emit the (simple)
    # common-format ar archive directly. Member order matters: debian-binary first.
    python3 - "$OUT" "$BUILD/debian-binary" "$BUILD/control.tar.gz" "$BUILD/data.tar.gz" <<'PY'
import os, sys
out, members = sys.argv[1], sys.argv[2:]
with open(out, "wb") as f:
    f.write(b"!<arch>\n")
    for m in members:
        data = open(m, "rb").read()
        name = os.path.basename(m)
        hdr = (name.ljust(16) + "0".ljust(12) + "0".ljust(6) + "0".ljust(6)
               + "100644".ljust(8) + str(len(data)).ljust(10)).encode() + b"`\n"
        assert len(hdr) == 60, len(hdr)
        f.write(hdr)
        f.write(data)
        if len(data) % 2 == 1:
            f.write(b"\n")  # members are padded to an even byte boundary
PY
else
    echo "error: need dpkg-deb or python3 to build the .deb" >&2
    exit 1
fi

echo "==> built $OUT"
ls -lh "$OUT" | awk '{print "    size:", $5}'
