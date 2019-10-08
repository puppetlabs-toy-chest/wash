# TEST INFRA:
#   AWS: demo_one profile with at least one EC2 instance and one S3 object
#   GCP: Wash project with at least one GCP compute instance and one GCP storage object
#   Puppet: Master and some agents with reports (modified w/in the last hour)
#
# TO RECORD:
# $ asciinema rec -c "doitlive play docs/_screencast.sh -q" wash.cast
# then type like mad
#doitlive env: SHELL=bash
#doitlive shell: wash
#doitlive speed: 2
#doitlive prompt: damoekri

# FS entry
echo Tailing logs on containers/VMs is as easy as tail -f
tail -f aws/demo_one/resources/ec2/instances/i-0d27b603c65103c1c/fs/var/log/messages
# Ctrl-C
echo And it is still easy when they are from different vendors
tail -f aws/demo_one/resources/ec2/instances/i-0d27b603c65103c1c/fs/var/log/messages gcp/Wash/compute/instance-1/fs/var/log/messages
# Ctrl-C

# Find + grep
echo You can also grep and filter those log files
grep 'systemd' aws/demo_one/resources/ec2/instances/i-0d27b603c65103c1c/fs/var/log/messages
find aws/demo_one/resources/ec2/instances/i-0d27b603c65103c1c/fs/var/log -mtime -1h
echo And grep and filter other things as well, like GCP storage objects
grep 'termination_date' gcp/Wash/storage/some-wash-stuff/reaper.sh
find gcp/Wash/storage/some-wash-stuff -name '*.sh'

# Extend Wash
echo In fact, you can extend Wash to talk to more things. This is as easy as adding a script
cat ~/.puppetlabs/wash/wash.yaml
ls puppet/master/nodes
find puppet -k '*report' -mtime -1h
echo Thanks github.com/timidri for writing puppetwash

exit
# edit wash.cast to have width=100, height=20
# upload as wash-integrations@puppet.com
