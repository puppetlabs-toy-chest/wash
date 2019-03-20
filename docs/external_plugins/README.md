# Introduction
External plugins let Wash talk to other things outside of the built-in plugins. They can be written in any language. To write an external plugin, you need to do the following:

1. Think about what the plugin would look like if it were modeled as a filesystem. Some useful questions to ask here are:
    * What are the things (entries) that I want this plugin to represent?
    * What things are my directories? What things are my files?
    * What Wash actions should I support on these things?

2. Write the [plugin script](plugin_script.md). This is the script that Wash will shell out to whenever it needs to invoke an action on a specific entry within your plugin.

3. Add the plugin to the (configurable) `plugins.yaml` file by specifying a path to the plugin script. The name will be determined by invoking the script with the `init` action. An example `plugins.yaml` file is shown below:

    ```
    - script: '/Users/enis.inan/wash/external-plugins/external-aws.rb'
    - script: '/Users/enis.inan/wash/external-plugins/network.sh'
    ```
4. Start the Wash server to see your plugin in action.

**NOTE:** You can override the default `plugins.yaml` path via the `external-plugins` flag. See `wash help server` for more information.