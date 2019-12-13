---
title: Getting Started
---

Wash is distributed as a single binary; the only prerequisite is `libfuse`. Here’s how to install it.

* Install `libfuse`
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
    * If anything fails, then check out the [known issues page]({{ '/known_issues' | relative_url }}) to see if the failure(s) correspond to any of the known issues. Otherwise, please don't hesitate to ask us on [slack](https://puppetcommunity.slack.com/app_redirect?channel=wash) for help (or file an [issue](https://github.com/puppetlabs/wash/issues))! Note that you can use `wash --verify-install` to test any fixes.

* You're good to go! Try running `wash` to start up the shell[^1]
    * If you plan on extending Wash, then check out the [external plugin docs]({{ '/docs/external-plugins' | relative_url }}).

**Note:** Wash collects anonymous data about how you use it. See the [analytics docs]({{ '/docs#analytics' | relative_url }}) for more details.

[^1]: If you came here from the [introduction]({{ '/' | relative_url }}), then you might find that the Wash prompt displayed in your shell is something like `wash . ❯` which is different from the screencasts. What you're seeing is correct. What's shown in the screencasts is an approximation that was the result of a trade-off between "good-enough Wash prompt behavior" and an "efficient and easy way to record a screencast".
