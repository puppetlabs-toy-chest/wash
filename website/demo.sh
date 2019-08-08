# with 'docker-compose -f examples/swarm/docker-compose.yml up -d' running
# $ asciinema rec -c "doitlive play demo.sh -q" wash.cast
# then type like mad
#doitlive env: SHELL=bash
#doitlive shell: wash
#doitlive speed: 2
#doitlive prompt: damoekri

echo Wash presents a hierarchical view of your cloud resources.
tree -sh -L 4 docker

echo Wash resources provide support for different actions, such as streaming updates or executing commands.
list docker/containers
list docker/containers/swarm_web_1

tail -f docker/containers/*/log
# curl http://localhost:5000
# Ctrl-C

wexec docker/containers/swarm_web_1 uname -a

echo The same things patterns can be used with other platforms as well.
ls gcp/Wash/*
tree -sh gcp/Wash/storage
tail gcp/Wash/compute/instance-1/fs/var/log/messages
wexec gcp/Wash/compute/instance-1 uname -a

echo Wash builds more powerful capabilities on the ability to execute commands.
wash ps docker/containers/* gcp/Wash/compute/instance-1

echo Wash find supports powerful queries on resource metadata.
jq .status gcp/Wash/compute/instance-1/metadata.json
echo "Let's find all running instances."
find gcp/Wash -meta .status RUNNING

exit
# upload as wash-integrations@puppet.com
