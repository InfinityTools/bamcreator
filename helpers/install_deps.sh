#!/bin/sh

# *** This script makes sure specified packages are installed locally. ***
# *** Syntax: install_deps [-u] package1 [package2 [...]]              ***
# *** Option -u: Update installed packages                             ***

# Syntax: install_package(update_flag: string,  package_name: string)
install_package() {
  if test $# -gt 1; then
    go list $2 >/dev/null 2>&1
    if test $? -eq 0; then
      # Don't update existing packages?
      if test $1 = 0; then
        return 0
      fi
    fi

    echo "Installing package $2"
    go get -f -u $2 || exit 1
  fi
}

# Checking Go compiler
if test ! $(which go); then
  echo "Error: Go compiler not found."
  exit 1
fi

# Processing packages
update=0
while test $# != 0
do
  case $1 in
  -u)
    update=1
    ;;
  *)
    install_package $update $1
    ;;
  esac
  shift
done
