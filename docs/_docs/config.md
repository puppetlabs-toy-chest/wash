---
title: Config
---

## wash.yaml

The Wash config file is located at `~/.puppetlabs/wash/wash.yaml`, and can be used to configure the [`wash-server`](#wash-server). You can override this location via the `config-file` flag.

Below are all the configurable options.

* `logfile` - The location of the server's log file (default `stdout`)
* `loglevel` - The server's loglevel (default `info`)
* `cpuprofile` - The location that the server's CPU profile will be written to (optional)
* `external-plugins` - The external plugins that will be loaded. See [➠External Plugins]
* `plugins` - A list of shipped plugins to enable. If omitted or empty, it will load all of the shipped plugins. Note that Wash ships with the `docker`, `kubernetes`, `aws`, and `gcp` plugins.
* `socket` - The location of the server's socket file (default `<user_cache_dir>/wash/wash-api.sock`)

All options except for `external-plugins` can be overridden by setting the `WASH_<option>` environment variable with option converted to ALL CAPS.

NOTE: Do not override `socket` in a config file. Instead, override it via the `WASH_SOCKET` environment variable. Otherwise, Wash's commands will not be able to interact with the server because they cannot access the socket.

## wash shell

Wash uses your system shell to provide the shell environment. It determines this using the `SHELL` environment variable or falls back to `/bin/sh`, so if you'd like to specify a particular shell set the `SHELL` environment variable before starting Wash.

Wash uses the following environment variables

- `WASH_SOCKET` determines how to communicate with the Wash daemon
- `W` describes the path to Wash's starting directory on the host filesystem; use `cd $W` to return to the start or `ls $W/...` to list things relative to Wash's root
- `PATH` is prefixed with the location of the Wash binary and any other executables it creates

For some shells, Wash provides a customized environment. Please [file an issue](https://github.com/puppetlabs/wash/issues/new?assignees=&labels=Feature&template=feature-request.md) if you'd like to add support for new shells.

Wash currently provides a customized environment for

- `bash`
- `zsh`

Customized environments alias Wash subcommands to save typing out `wash <subcommand>` so they feel like shell builtins. If you want to use an executable or builtin Wash has overridden, please use its full path or the `builtin` command.

Customized environments also supports reading `~/.washenv` and `~/.washrc` files. These files are loaded as follows:

1. If running Wash non-interactively (by piping `stdin` or passing the `-c` option)
   1. If `~/.washenv` does not exist, load the shell's default non-interactive config (such as `.zshenv` or from `BASH_ENV`)
   2. Configure subcommand aliases
   3. If `~/.washenv` exists, load it
2. If running Wash interactively
   1. Do all non-interactive config above
   2. If `~/.washrc` does not exist, load the shell's default interactive config (such as `.bash_profile` or `.zshrc`)
   3. Re-configure subcommand aliases, and configure the command prompt
   4. If `~/.washrc` exists, load it

That order ensures that the out-of-box experience of Wash is not adversely impacted by your existing environment while still inheriting most of your config. If you customize your Wash environment with `.washenv` and `.washrc`, be aware that it's possible to override Wash's default prompt and aliases.

For other shells, Wash creates executables for subcommands and does no other customization.
