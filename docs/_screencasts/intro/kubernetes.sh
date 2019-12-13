#doitlive speed: 2
#doitlive prompt: {TTY.CYAN}wash {r_angle}{TTY.RESET}

cd kubernetes/docker-desktop
ls
cd docker
ls

# Pods
cd pods
ls
find . -k '*pod' -m '.status.phase' Running -m '.metadata.labels.pod-template-hash' -exists
wexec compose-6c67d745f6-ljtwr/compose uname
cat compose-6c67d745f6-ljtwr/compose

# TODO: Add PVCs
