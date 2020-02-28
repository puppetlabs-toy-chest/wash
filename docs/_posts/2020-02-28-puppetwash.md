---
title: "Puppet + Wash"
description: "Navigate your Puppet infrastructure in Wash"
author: michaelsmith
---

A full Puppet installation - masters, agents, database, etc - contains a lot of knowledge about your infrastructure. Sometimes this can be challenging to get at. [Puppetwash](https://github.com/puppetlabs/puppetwash) exposes that knowledge, starting with making your PuppetDB data more accessible. I'll demonstrate it by setting up an example with [Pupperware](https://github.com/puppetlabs/pupperware).

## Start Pupperware

Following the [instructions to provision the stack](https://github.com/puppetlabs/pupperware#provisioning), I've installed Docker Compose (using macOS 10.15) and - within the Pupperware project - run
```
$ docker-compose up -d
```

This starts PostgreSQL, Puppet Server, and PuppetDB in separate containers. I'm using defaults for DNS_ALT_NAMES (that determine the addresses I can use to securely connect to Puppet Server or PuppetDB), which are `puppet` and `puppet.test`. I've added the following line to my `/etc/hosts` file
```
127.0.0.1	puppet.test puppetdb.test
```

Pupperware's README notes that the `pupperware_puppetserver-config` includes SSL certificates for your Pupperware infrastructure. Now that Pupperware's running, lets take a look at that with Wash. We can first check that PuppetDB is running by looking at its logs
```
wash . > cd docker/containers
wash docker/containers > tail -f pupperware_puppetdb_1/log
===> pupperware_puppetdb_1/log <===
127.0.0.1 - - [27/Feb/2020:23:25:39 +0000] "GET /status/v1/services/puppetdb-status HTTP/1.1" 200 247 "-" "curl/7.64.0"
127.0.0.1 - - [27/Feb/2020:23:25:49 +0000] "GET /status/v1/services/puppetdb-status HTTP/1.1" 200 247 "-" "curl/7.64.0"
```

Let's take a look at what certificates were created
```
wash docker/containers > cd ../volumes/pupperware_puppetserver-config
wash docker/volumes/pupperware_puppetserver-config > tree
.
├── auth.conf
├── hiera.yaml
├── puppet.conf
├── puppetdb.conf
└── ssl
    ├── ca
    │   ├── ca_crl.pem
    │   ├── ca_crt.pem
    │   ├── ca_key.pem
    │   ├── ca_pub.pem
    │   ├── infra_crl.pem
    │   ├── infra_inventory.txt
    │   ├── infra_serials
    │   ├── inventory.txt
    │   ├── requests
    │   ├── serial
    │   └── signed
    │       ├── puppet.test.pem
    │       └── puppetdb.test.pem
    ├── certificate_requests
    ├── certs
    │   ├── ca.pem
    │   └── puppet.test.pem
    ├── crl.pem
    ├── private
    ├── private_keys
    │   └── puppet.test.pem
    └── public_keys
        └── puppet.test.pem

9 directories, 20 files
```

> Note that on Linux it'd be possible to find the volume with `docker inspect`, but on macOS Docker containers run in an isolated Linux VM.

We can see that the CA has signed two certificates, one for `puppet.test` and another for `puppetdb.test`. We can see how PuppetDB expects to be addressed by looking at its Subject name
```
wash docker/volumes/pupperware_puppetserver-config ❯ openssl x509 -text -in ssl/ca/signed/puppetdb.test.pem | grep -m1 Subject
        Subject: CN=puppetdb.test
```
So we should be able to connect to PuppetDB using it's default port 8081 at `puppetdb.test`. We can confirm that with `cat puppetdb.conf`. To get the local port we'll use to connect to it - within the Pupperware project - run
```
$ docker-compose port puppetdb 8081
0.0.0.0:32770
```

Let's copy the certificates somewhere we can use them later. The Puppet Server will be in PuppetDB's whitelist, so let's use that one since it's easily accessible.
```
wash docker/volumes/pupperware_puppetserver-config ❯ mkdir -p ~/.puppetlabs/wash/pupperware
wash docker/volumes/pupperware_puppetserver-config ❯ cp certs/ca.pem certs/puppet.test.pem ~/.puppetlabs/wash/pupperware/
wash docker/volumes/pupperware_puppetserver-config ❯ cp private_keys/puppet.test.pem ~/.puppetlabs/wash/pupperware/puppet.test.key.pem
```

Last, let's make sure there's something in PuppetDB to look at. In the Pupperware project we'll do a noop agent run to add facts and a report
```
$ docker-compose exec puppet puppet agent -t --noop
Info: Using configured environment 'production'
Info: Retrieving pluginfacts
Info: Retrieving plugin
Info: Retrieving locales
Info: Applying configuration version '1582848507'
Info: Creating state file /opt/puppetlabs/puppet/cache/state/state.yaml
Notice: Applied catalog in 0.02 seconds
```

## Configure Puppetwash

To configure Puppetwash we need to install and configure it. It's distributed as a Ruby Gem, so we can install it with
```
gem install puppetwash
```

Get the path to the Puppetwash script with
```
$ gem contents puppetwash
/Users/me/.gem/ruby/2.6.0/gems/puppetwash-0.2.0/puppet.rb
```

We then add that to Wash's config at `~/.puppetlabs/wash/wash.yaml`
```
external-plugins:
- script: '/Users/me/.gem/ruby/2.6.0/gems/puppetwash-0.2.0/puppet.rb'
```

If we were using Puppet Enterprise we would use a user authentication (RBAC) token. Wash config would look like
```
external-plugins:
- script: '/Users/me/.gem/ruby/2.6.0/gems/puppetwash-0.2.0/puppet.rb'
my_pe_instance:
  puppetdb_url: https://puppet:8081
  rbac_token: <my_rbac_token>
  cacert: /path/to/cacert.pem
```

With open-source Puppet, we'll use cert-based authentication. Using the examples from above, `~/.puppetlabs/wash/wash.yaml` should look like
```
external-plugins:
- script: '/Users/me/.gem/ruby/2.6.0/gems/puppetwash-0.2.0/puppet.rb'
my_pe_instance:
  puppetdb_url: https://puppetdb.test:32770
  cacert: /Users/michaelsmith/.puppetlabs/wash/pupperware/ca.pem
  cert: /Users/michaelsmith/.puppetlabs/wash/pupperware/puppet.test.pem
  key: /Users/michaelsmith/.puppetlabs/wash/pupperware/puppet.test.key.pem
```

Start (or restart) Wash to load the new Puppetwash config and you should now be able to explore PuppetDB data in Wash
```
wash . > cd puppet/pupperware/nodes
wash puppet/pupperware/nodes > tree
.
└── puppet.test
    ├── catalog.json
    ├── facts
    │   ├── aio_agent_version
    │   ├── architecture
    │   ├── augeas
    │   ├── augeasversion
    │   ├── bios_release_date
    │   ├── bios_vendor
    │   ├── bios_version
    │   ├── blockdevice_sda_model
    │   ├── blockdevice_sda_size
    │   ├── blockdevice_sda_vendor
    │   ├── blockdevice_sr0_model
    │   ├── blockdevice_sr0_size
    │   ├── blockdevice_sr0_vendor
    │   ├── blockdevice_sr1_model
    │   ├── blockdevice_sr1_size
    │   ├── blockdevice_sr1_vendor
    │   ├── blockdevice_sr2_model
    │   ├── blockdevice_sr2_size
    │   ├── blockdevice_sr2_vendor
    │   ├── blockdevices
    │   ├── chassisassettag
    │   ├── chassistype
    │   ├── clientcert
    │   ├── clientnoop
    │   ├── clientversion
    │   ├── disks
    │   ├── dmi
    │   ├── domain
    │   ├── facterversion
    │   ├── filesystems
    │   ├── fips_enabled
    │   ├── fqdn
    │   ├── gid
    │   ├── hardwareisa
    │   ├── hardwaremodel
    │   ├── hostname
    │   ├── hypervisors
    │   ├── id
    │   ├── identity
    │   ├── interfaces
    │   ├── ipaddress
    │   ├── ipaddress_eth0
    │   ├── ipaddress_lo
    │   ├── is_virtual
    │   ├── kernel
    │   ├── kernelmajversion
    │   ├── kernelrelease
    │   ├── kernelversion
    │   ├── load_averages
    │   ├── macaddress
    │   ├── macaddress_eth0
    │   ├── memory
    │   ├── memoryfree
    │   ├── memoryfree_mb
    │   ├── memorysize
    │   ├── memorysize_mb
    │   ├── mountpoints
    │   ├── mtu_eth0
    │   ├── mtu_ip6tnl0
    │   ├── mtu_lo
    │   ├── mtu_tunl0
    │   ├── netmask
    │   ├── netmask_eth0
    │   ├── netmask_lo
    │   ├── network
    │   ├── network_eth0
    │   ├── network_lo
    │   ├── networking
    │   ├── operatingsystem
    │   ├── operatingsystemmajrelease
    │   ├── operatingsystemrelease
    │   ├── os
    │   ├── osfamily
    │   ├── partitions
    │   ├── path
    │   ├── physicalprocessorcount
    │   ├── processor0
    │   ├── processor1
    │   ├── processor2
    │   ├── processor3
    │   ├── processor4
    │   ├── processor5
    │   ├── processorcount
    │   ├── processors
    │   ├── productname
    │   ├── puppetversion
    │   ├── ruby
    │   ├── rubyplatform
    │   ├── rubysitedir
    │   ├── rubyversion
    │   ├── selinux
    │   ├── serialnumber
    │   ├── swapfree
    │   ├── swapfree_mb
    │   ├── swapsize
    │   ├── swapsize_mb
    │   ├── system_uptime
    │   ├── timezone
    │   ├── trusted
    │   ├── uptime
    │   ├── uptime_days
    │   ├── uptime_hours
    │   ├── uptime_seconds
    │   ├── uuid
    │   └── virtual
    └── reports
        └── 2020-02-28T00:08:28.495Z

3 directories, 107 files
```

Pupperware presents the last catalog (`catalog.json`), reports (as JSON), and individual fact values. We can look at fact values with
```
wash puppet/pupperware/nodes > cd puppet.test
wash puppet/pupperware/nodes/puppet.test > cat facts/osfamily
Debian
```

Reports also have metadata attached to them that we can filter on
```
wash puppet/pupperware/nodes/puppet.test > meta reports/2020-02-28T00:08:28.495Z
end_time: "2020-02-28T00:08:28.495Z"
environment: production
hash: 8043a61b020be5ed2f161eac6e1d7f1df2ee8fcf
noop: true
producer: puppet.test
puppet_version: 6.7.2
status: unchanged
wash puppet/pupperware/nodes/puppet.test > find . -meta .noop -true
./reports/2020-02-28T00:08:28.495Z
wash puppet/pupperware/nodes/puppet.test > find . -meta .noop -false
wash puppet/pupperware/nodes/puppet.test > echo $?
0
```

These are just a few things we thought might be useful to be able to explore. Let us know what other ideas you might have at https://github.com/puppetlabs/puppetwash/issues (or check out https://puppetlabs.github.io/wash/contributing for other ways to get involved).
