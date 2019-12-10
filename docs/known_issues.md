---
title: Known Issues
---
# On macOS

If using iTerm2, we recommend installing [iTerm2's shell integration](https://www.iterm2.com/documentation-shell-integration.html) to avoid [issue#84](https://github.com/puppetlabs/wash/issues/84).

If Wash exits with an exit status of 1, and the error message is related to `load_osxfuse`, then that typically means that Mac OS blocked loading the FUSE extension. See [this github issue](https://github.com/osxfuse/osxfuse/issues/437#issuecomment-340347943) for more details.

If Wash exits with an exit status of 255, that typically means that it couldn't load the FUSE extensions. MacOS only allows for a certain (small) number of virtual devices on the system, and if all available slots are taken up by other programs then we won't be able to run. You can view loaded extensions with `kextstat`. More information in [this github issue for *FUSE for macOS*](https://github.com/osxfuse/osxfuse/issues/358).

