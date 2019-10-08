---
title: Setting up the test environment
main: true
sections: []
---

The test environment uses a multi-node Docker application. Here's how to set it up.

1. Install [Docker](https://docs.docker.com/v17.09/engine/installation/) and make sure it's running.
    * If you're using Mac OS, then make sure to include `docker` in your PATH. This is typically found in `/Applications/Docker.app/Contents/Resources/bin`.
1. Install [docker-compose](https://docs.docker.com/compose/install/). If you're using Mac OS, then it should already be included with Docker.
1. Run the following commands in your terminal

    ```
    curl -O {{ page.url | remove: "/index.html" | append: "/test_environment.tgz" | absolute_url }}
    ```

    ```
    tar -xvzf test_environment.tgz
    ```

    ```
    docker-compose -f wash_tutorial/docker-compose.yml up -d --build
    ```

If everything worked, then you should see the `wash_tutorial_redis_1` and `wash_tutorial_web_1` containers when you run `docker ps`. You can run `docker-compose -f wash_tutorial/docker-compose.yml down` to bring down the test environment.

Click [here](../) to go back to the tutorials page.
