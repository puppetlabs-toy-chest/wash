---
title: Installing Wash
---
Wash is distributed as a single binary; the only prerequisite is `libfuse`. Here’s how to install it.

* Install `libfuse` if you haven’t already
    * On MacOS using homebrew: `brew cask install osxfuse`
        * You'll also need to restart your computer
    * On CentOS: `yum install fuse fuse-libs`
    * On Debian/Ubuntu: `apt-get install fuse`

* Install the Wash binary
    * On MacOS using homebrew: `brew install puppetlabs/puppet/wash`
    * or [download](https://github.com/puppetlabs/wash/releases/latest) Wash for your platform, then run the following commands in your terminal:
        * `tar -xvzf <path_to_downloaded_wash_tgz>` (unpack it)
        * `chmod +x <path_to_wash>` (ensure it's executable)
        * `mv <path_to_wash> /usr/local/bin` (or add the binary to your PATH)

* Run `wash --verify-install` to ensure that the installation was successful
    * If anything fails, then check out the [known issues page]({{ '/known_issues' | relative_url }}) to see if the failure(s) correspond to any of the known issues. Otherwise, please don't hesitate to ask us on [slack](https://puppetcommunity.slack.com/app_redirect?channel=wash) for help!

**Note:** Wash collects anonymous data about how you use it. See the [analytics docs]({{ '/docs#analytics' | relative_url }}) for more details.
