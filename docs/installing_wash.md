---
title: Installing Wash
---
Wash is distributed as a single binary; the only prerequisite is `libfuse`. Here’s how to install it.

* Install `libfuse` if you haven’t already
    * E.g. on MacOS using homebrew: `brew cask install osxfuse`
    * E.g. on CentOS: `yum install fuse fuse-libs`
    * E.g. on Debian/Ubuntu: `apt-get install fuse`

* Install the Wash binary
    * E.g. on MacOS using homebrew: `brew install puppetlabs/puppet/wash`
    * or [download](https://github.com/puppetlabs/wash/releases) Wash for your platform, then run the following commands in your terminal:
        * `tar -xvzf <path_to_downloaded_wash_tgz>` (unpack it)
        * `chmod +x <path_to_wash>` (ensure it's executable)
        * `mv <path_to_wash> /usr/local/bin` (or add the binary to your PATH)

* Run `wash version` to ensure that the installation was successful

**Note:** Wash collects anonymous data about how you use it. See the [analytics docs]({{ '/docs#analytics' | relative_url }}) for more details.
