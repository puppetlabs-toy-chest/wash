#doitlive speed: 2
#doitlive prompt: {TTY.CYAN}wash {r_angle}{TTY.RESET}

cd docker
ls

# Containers
cd containers
ls
find . -k '*container' -m '.state' running -m '.labels.com\.docker\.compose\.version' -exists
wexec wash_tutorial_redis_1 uname
cd wash_tutorial_redis_1
ls
cat log
cd fs
ls
find var/log -mtime -6w
cat var/log/dpkg.log
tail -f var/log/dpkg.log
# Hit Ctrl+C

cd $W/docker
ls

# Volumes
cd volumes
ls
find wash_tutorial_redis -name '*.aof'
cd wash_tutorial_redis
ls
cat appendonly.aof
