module github.com/puppetlabs/wash

// Ensures we get the correct client version, tied to v19.03.8.
// docker/docker stopped tagging in 2017. The engine code we care about appears to be maintained at
// docker/engine, but all of that code still refers to other dependencies within itself via
// docker/docker. So we rewrite docker/docker => docker/engine so that the correct version is used.
replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20200309214505-aa6a9891b09c

// Version considerations:
// - k8s.io/client-go reports "latest" as v11.0.0. Ignore that, it follows Kubernetes versioning.
// - googleapis/gnostic 0.4.0 changed case of OpenAPIv2, making it incompatible with client-go. Stick to 0.3.x.

require (
	bazil.org/fuse v0.0.0-20200117225306-7b5117fecadc
	cloud.google.com/go v0.55.0
	cloud.google.com/go/firestore v1.2.0
	cloud.google.com/go/pubsub v1.3.1
	cloud.google.com/go/storage v1.6.0
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Benchkram/errz v0.0.0-20180520163740-571a80a661f2
	github.com/InVisionApp/tabular v0.3.0
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/anmitsu/go-shlex v0.0.0-20161002113705-648efa622239 // indirect
	github.com/araddon/dateparse v0.0.0-20190622164848-0fb0a474d195
	github.com/avast/retry-go v2.6.0+incompatible
	github.com/aws/aws-sdk-go v1.30.1
	github.com/cloudfoundry-attic/jibber_jabber v0.0.0-20151120183258-bcc4c8345a21
	github.com/cloudfoundry/jibber_jabber v0.0.0-20151120183258-bcc4c8345a21 // indirect
	github.com/containerd/containerd v1.3.3 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/ekinanp/go-cache v2.1.0+incompatible
	github.com/ekinanp/jsonschema v0.0.0-20190624212413-cd4dbe12fbae
	github.com/elazarl/goproxy v0.0.0-20200315184450-1f3cb6622dad // indirect
	github.com/emirpasic/gods v1.12.0
	github.com/fatih/color v1.9.0
	github.com/flynn/go-shlex v0.0.0-20150515145356-3f9db97f8568 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/gammazero/workerpool v0.0.0-20200311205957-7b00833861c6
	github.com/getlantern/deepcopy v0.0.0-20160317154340-7f45deb8130a
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/gliderlabs/ssh v0.3.0
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/go-openapi/errors v0.19.4 // indirect
	github.com/go-openapi/strfmt v0.19.5 // indirect
	github.com/gobwas/glob v0.2.3
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.3.5
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/gorilla/mux v1.7.4
	github.com/hashicorp/vault/sdk v0.1.14-0.20200305172021-03a3749f220d
	github.com/hpcloud/tail v1.0.0
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/jedib0t/go-pretty v4.3.0+incompatible
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/kevinburke/ssh_config v0.0.0-20190725054713-01f96b0aa0cd
	github.com/kr/logfmt v0.0.0-20140226030751-b84e30acd515
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/mattn/go-isatty v0.0.12
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/shirou/gopsutil v2.20.2+incompatible
	github.com/shopspring/decimal v0.0.0-20200227202807-02e2044944cc
	github.com/sirupsen/logrus v1.5.0
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v0.0.7
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.5.1
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/xlab/treeprint v1.0.0
	go.mongodb.org/mongo-driver v1.3.1 // indirect
	golang.org/x/crypto v0.0.0-20200323165209-0ec3e9974c59
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20200331124033-c3d80250170d
	google.golang.org/api v0.20.0
	google.golang.org/genproto v0.0.0-20200331122359-1ee6d9798940
	gopkg.in/go-ini/ini.v1 v1.55.0
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible // indirect
	k8s.io/api v0.18.0
	k8s.io/apimachinery v0.18.0
	k8s.io/client-go v0.18.0
)

go 1.13
