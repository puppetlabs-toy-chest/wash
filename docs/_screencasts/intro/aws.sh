#doitlive speed: 2
#doitlive prompt: {TTY.CYAN}wash {r_angle}{TTY.RESET}

cd aws/wash/resources
ls

# EC2 instances
cd ec2/instances
ls
find . -k '*instance' -m '.state.name' running -m '.tags[?].key' owner
wexec i-04621c13583930e6c uname
cd i-04621c13583930e6c
ls
cat console.out
cd fs
ls
find var/log -mtime -1h
cat var/log/messages
tail -f var/log/messages
# Hit Ctrl+C

cd $W/aws/wash/resources
ls

# S3
cd s3
ls
find some-wash-bucket -name '*.sh'
cd some-wash-bucket
ls
cat godoc.sh
cd folder
ls
