---
title: whistory
---
{% include test_environment_reminder.md %}

Use the `whistory` command to view a history of all executed commands that have interacted with the Wash server.

```
bash-3.2$ wash
Welcome to Wash!
  Wash includes several built-in commands: wexec, find, list, meta, tail.
  See commands run with wash via 'whistory', and logs with 'whistory <id>'.
Try 'help'
wash . ❯ ls
aws/
docker/
gcp/
kubernetes/
wash . ❯ ls docker/volumes/wash_tutorial_redis
appendonly.aof
wash . ❯ cat docker/volumes/wash_tutorial_redis/appendonly.aof
wash . ❯ whistory
1  2019-10-05 14:51  ls -G .
2  2019-10-05 14:51  ls -G
3  2019-10-05 14:51  wash
4  2019-10-05 14:51  ls -G docker/volumes/wash_tutorial_redis
5  2019-10-05 14:51  cat docker/volumes/wash_tutorial_redis/appendonly.aof
6  2019-10-05 14:52  wash whistory
wash . ❯ exit
Goodbye!
```

We see that `whistory` recorded all of our activity[^1]. This activity is specific to our session and will not appear in any other sessions. We can test this by starting a new session and running the command again:

[^1]: Ignore the `ls -G .` and `wash` for now. Those will be fixed in [521](https://github.com/puppetlabs/wash/issues/521).

```
bash-3.2$ wash
Welcome to Wash!
  Wash includes several built-in commands: wexec, find, list, meta, tail.
  See commands run with wash via 'whistory', and logs with 'whistory <id>'.
Try 'help'
wash . ❯ whistory
1  2019-09-25 21:19  ls -G .
2  2019-09-25 21:19  wash whistory
wash . ❯ exit
Goodbye!
```

Notice from the output that this session’s history is different from the previous session.

You can pass in the command’s history ID to view a detailed log of its activity. 

```
{% raw %}
wash . ❯ whistory
1  2019-09-25 21:24  ls -G .
2  2019-09-25 21:24  ls -G docker/containers
3  2019-09-25 21:24  wash whistory
wash . ❯ whistory 2
Sep 25 21:24:54.902 FUSE: List /docker/containers
Sep 25 21:24:54.919 Listing 20 containers in &{{containers {{0 0 <nil>} {0 0 <nil>} {0 0 <nil>} {0 0 <nil>} 0 false 0 false map[]} 35 /docker/containers [15000000000 15000000000 15000000000] map[] false} 0xc0002cc880}
Sep 25 21:24:54.921 FUSE: Listed in /docker/containers: [{Inode:0 Type:dir Name:k8s_compose_compose-6c67d745f6-q54n8_docker_57a0f7e9-c41c-11e9-9d31-025000000001_5} {Inode:0 Type:dir Name:k8s_POD_coredns-fb8b8dccf-2mdnw_kube-system_bfbca97d-c3e6-11e9-9d31-025000000001_5} {Inode:0 Type:dir Name:k8s_POD_redis_default_4d21ee44-c5c7-11e9-9d31-025000000001_5} {Inode:0 Type:dir Name:k8s_POD_kube-proxy-v4fc5_kube-system_bfc80631-c3e6-11e9-9d31-025000000001_5} {Inode:0 Type:dir Name:k8s_POD_kube-controller-manager-docker-desktop_kube-system_9c58c6d32bd3a2d42b8b10905b8e8f54_5} {Inode:0 Type:dir Name:k8s_POD_kube-apiserver-docker-desktop_kube-system_7c4f3d43558e9fadf2d2b323b2e78235_5} {Inode:0 Type:dir Name:k8s_coredns_coredns-fb8b8dccf-2mdnw_kube-system_bfbca97d-c3e6-11e9-9d31-025000000001_9} {Inode:0 Type:dir Name:k8s_redis_redis_default_4d21ee44-c5c7-11e9-9d31-025000000001_5} {Inode:0 Type:dir Name:k8s_etcd_etcd-docker-desktop_kube-system_3773efb8e009876ddfa2c10173dba95e_5} {Inode:0 Type:dir Name:k8s_kube-apiserver_kube-apiserver-docker-desktop_kube-system_7c4f3d43558e9fadf2d2b323b2e78235_5} {Inode:0 Type:dir Name:k8s_POD_etcd-docker-desktop_kube-system_3773efb8e009876ddfa2c10173dba95e_5} {Inode:0 Type:dir Name:k8s_kube-scheduler_kube-scheduler-docker-desktop_kube-system_124f5bab49bf26c80b1c1be19641c3e8_6} {Inode:0 Type:dir Name:k8s_coredns_coredns-fb8b8dccf-nsrj4_kube-system_bfbdb38c-c3e6-11e9-9d31-025000000001_9} {Inode:0 Type:dir Name:k8s_compose_compose-api-57ff65b8c7-gpk27_docker_579e832e-c41c-11e9-9d31-025000000001_8} {Inode:0 Type:dir Name:k8s_POD_compose-6c67d745f6-q54n8_docker_57a0f7e9-c41c-11e9-9d31-025000000001_6} {Inode:0 Type:dir Name:k8s_POD_compose-api-57ff65b8c7-gpk27_docker_579e832e-c41c-11e9-9d31-025000000001_5} {Inode:0 Type:dir Name:k8s_kube-proxy_kube-proxy-v4fc5_kube-system_bfc80631-c3e6-11e9-9d31-025000000001_5} {Inode:0 Type:dir Name:k8s_POD_coredns-fb8b8dccf-nsrj4_kube-system_bfbdb38c-c3e6-11e9-9d31-025000000001_5} {Inode:0 Type:dir Name:k8s_kube-controller-manager_kube-controller-manager-docker-desktop_kube-system_9c58c6d32bd3a2d42b8b10905b8e8f54_5} {Inode:0 Type:dir Name:k8s_POD_kube-scheduler-docker-desktop_kube-system_124f5bab49bf26c80b1c1be19641c3e8_5}]
{% endraw %}
```

This session's history consists of three entries. The second entry, with an ID of 2, is the command `ls -G docker/containers`. From the `whistory` output, we can see that its activity consisted of making a `List /docker/containers` request to Wash’s underlying `FUSE` library. Notice that the activity also recorded the `List` endpoint’s raw response.

`whistory` is primarily useful for debugging Wash-related failures. A typical pattern is to execute the failed command, then view its activity log to see if you can isolate the failure.

# Exercises
1. Execute each of the following commands and then use `whistory` to report all the requests that were made to the server. You should only report requests of the form `FUSE: <Request> <Entry>` or `API: <Request> <Entry>`. Ignore any analytics requests. 

   **Note:** Do not worry if you’ve disabled analytics. Wash swallows all analytics requests and no data is actually sent to Google Analytics. For more information, see our [analytics docs]({{ '/docs/#analytics' | relative_url }}).

    1. `wexec docker/containers/wash_tutorial_redis_1 uname`

        {% include exercise_answer.html answer="<code>API: Exec docker/containers/wash_tutorial_redis_1</code>" %}

    2. `meta docker/containers/wash_tutorial_redis_1`

        {% include exercise_answer.html answer="<code>API: Metadata docker/containers/wash_tutorial_redis_1</code>" %}

    3. `cat docker/volumes/wash_tutorial_redis/appendonly.aof`
        
        {% include exercise_answer.html answer="<code>FUSE: Open docker/volumes/wash_tutorial_redis/appendonly.aof</code>" %}

    4. `find docker -kind '*container'`

        {% capture answer_1d %}
          <code>API: List docker</code><br />
          <code>API: List docker/containers</code>
        {% endcapture %}
        {% include exercise_answer.html answer=answer_1d %}

# Next steps

That's the end of the _Debugging_ series! Click [here](../) to go back to the tutorials page.
