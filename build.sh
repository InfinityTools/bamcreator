#!/bin/sh

show_help() {
  echo "Usage: $0 [options] [target]"
  echo ""
  echo "Options:"
  echo "  --debug         Don't strip debugging symbols"
  echo "  --nodeps        Don't check dependencies"
  echo "  --update        Force updating dependencies"
  echo "  --compress      Compress the binary with upx if available"
  echo "  --help          This help"
  echo ""
  echo "Available targets:"
  echo "  bamcreator (default)"
  echo ""
  libos=$(go env GOOS)
  libarch=$(go env GOARCH)
  echo "The resulting binary is placed into the folder \"bin/$libos/$libarch\"."
  exit 0
}

# terminate(msg: string, level: int): Print "msg" and exit with "level". Default level is 0.
terminate() {
  if test $# != 0; then
    echo $1
    shift
  fi
  if test $# != 0; then
    exit $1
  else
    exit 0
  fi
}

# Checking Go compiler
if test ! $(which go); then
  terminate "Error: Go compiler not found." 1
fi

# Setting up variables
builddir="${0%/*}"
pkgRoot="github.com/InfinityTools"

pkgPrefix="$pkgRoot/bamcreator"
pkgBinRoot="bin"
pkgBinExt=""

# Dependencies
pkgCharmap="golang.org/x/text/encoding/charmap"
pkgBmp="golang.org/x/image/bmp"
pkgThreadpool="github.com/pbenner/threadpool"
pkgBinpack="$pkgRoot/go-binpack2d"
pkgCmdArgs="$pkgRoot/go-cmdargs"
pkgIETools="$pkgRoot/go-ietools"
pkgIEToolsBuffers="$pkgRoot/go-ietools/buffers"
pkgIEToolsPvrz="$pkgRoot/go-ietools/pvrz"
pkgImagequant="$pkgRoot/go-imagequant"
pkgLogging="$pkgRoot/go-logging"
pkgSquish="$pkgRoot/go-squish"

# Supported targets
targetBamCreator="bamcreator"
targetBamConv="bamconv"
targetBamGen="bamgen"

# Initializing with default target
target="$targetBamCreator"

# Evaluating command line arguments...
bin_flags="-ldflags -s"
skipdeps=0
compress=0
get_flags=""
build_flags=""
while test $# != 0
do
  case $1 in
  bamcreator)
    target="bamcreator"
    ;;
  bamconv)
    target="bamconv"
    ;;
  bamgen)
    target="bamgen"
    ;;
  --debug)
    bin_flags=""
    ;;
  --compress)
    compress=1
    ;;
  --nodeps)
    skipdeps=1
    ;;
  --update)
    get_flags="-u"
    build_flags="-a"
    ;;
  --help)
    show_help
    ;;
  esac
  shift
done

if test -z "$bin_flags"; then
  echo "Building $target debug version"
else
  echo "Building $target release version"
fi

if test $skipdeps = 0; then
  # Iterating over list of dependencies: simple check and install-on-demand
  chmod +x "$builddir/helpers/install_deps.sh"
  for pkg in  $pkgCharmap \
              $pkgBmp \
              $pkgThreadpool \
              $pkgBinpack \
              $pkgCmdArgs \
              $pkgIETools \
              $pkgIEToolsBuffers \
              $pkgIEToolsPvrz \
              $pkgImagequant \
              $pkgLogging \
              $pkgSquish; do
    echo Checking $pkg ...
    "$builddir/helpers/install_deps.sh" $get_flags $pkg || terminate "Cancelled." 1
  done
fi

libos=$(go env GOOS)
libarch=$(go env GOARCH)
echo "Detected: os=$libos, arch=$libarch"

if test "$libos" = "windows"; then
  pkgBinExt=".exe"
  # Use static linking on Windows if possible
  ldflags="-static -static-libstdc++ $CGO_LDFLAGS"
else
  ldflags="$CGO_LDFLAGS"
fi

# Starting build operation
pkgBinPath=$builddir/$pkgBinRoot/$libos/$libarch/$target$pkgBinExt
echo "Building \"$pkgBinPath\"..."
# echo "CGO_LDFLAGS="$ldflags" go build -o \"$pkgBinPath\" $build_flags $bin_flags $pkgPrefix/$target"
CGO_LDFLAGS="$ldflags" go build -o "$pkgBinPath" $build_flags $bin_flags $pkgPrefix/$target || terminate "Cancelled." 1

# Applying compression if needed
if test $compress = 1; then
  if test -x "$pkgBinPath"; then
    if test $(which upx); then
      echo "Compressing binary. This may take a while..."
      $(which upx) --best -q "$pkgBinPath" >/dev/null || terminate "Compression failed." 1
    else
      echo "Could not find upx. Skipping compression...."
    fi
  fi
fi

terminate "Finished." 0
