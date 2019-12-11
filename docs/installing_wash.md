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
    * or download Wash for your platform.
        * On Linux:
            ```
            curl -sfLO https://github.com/puppetlabs/wash/releases/download/{WASH_VERSION}/wash-{WASH_VERSION}-x86_64-unknown-linux.tgz
            ```
        * On MacOS:
            ```
            curl -sfLO https://github.com/puppetlabs/wash/releases/download/{WASH_VERSION}/wash-{WASH_VERSION}-x86_64-apple-darwin.tgz
            ```

      where `{WASH_VERSION}` is the [latest Wash version](https://github.com/puppetlabs/wash/releases/latest) (e.g. `0.16.0`). After downloading the `.tgz`, run the following commands:
        * `tar -xvzf <path_to_downloaded_wash_tgz>` (unpack it)
        * `chmod +x wash` (ensure it's executable)
        * `mv wash /usr/local/bin` (or add the binary to your PATH)

* Run `wash --verify-install` to ensure that the installation was successful
    * If anything fails, then check out the [known issues page]({{ '/known_issues' | relative_url }}) to see if the failure(s) correspond to any of the known issues. Otherwise, please don't hesitate to ask us on [slack](https://puppetcommunity.slack.com/app_redirect?channel=wash) for help! Note that you can use `wash --verify-install` to test any fixes.

* You're good to go! Try running `wash` to start up the shell. 

**Note:** Wash collects anonymous data about how you use it. See the [analytics docs]({{ '/docs#analytics' | relative_url }}) for more details.
