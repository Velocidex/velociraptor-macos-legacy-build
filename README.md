# Build Velociraptor for MacOS 11

Velociraptor uses the latest Go toolchain to incorporate the latest
security patches and updates. However, Golang has dropped support for
MacOS 11 at version 1.24:

https://go.dev/wiki/MinimumRequirements#macos-ne-os-x-aka-darwindarwin

This script allows building a version of Velociraptor using the last
supported Go version 1.24. However, note the following caveats:

* To build under this unsupported Go version we had to freeze
  dependencies. Therefore this build includes known buggy and
  unsupported dependencies.

* This build may be insecure! since it includes unsupported
  dependencies.

* We typically update to the latest version of Velociraptor but it may
  be that in future we disable some feature (VQL plugins) that can not
  be easily updated.

NOTE: Do not use this build in a general deployment! Only use it for
deploying on deprecated, unsupported operating systems:

* MacOS 11 - OS X 10.11 El Capitan
