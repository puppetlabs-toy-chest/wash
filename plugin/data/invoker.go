package data

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os/exec"
	"strings"

	"github.com/kballard/go-shellquote"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

func invoke(input string, entry plugin.Entry) (io.Reader, error) {
	t, err := template.New("invoke").Parse(input)
	if err != nil {
		return nil, err
	}

	var cmdstring strings.Builder
	if err = t.Execute(&cmdstring, entry); err != nil {
		return nil, err
	}

	log.Debugf("Invoking: %v", cmdstring.String())
	segments, err := shellquote.Split(cmdstring.String())
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(segments[0], segments[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("%v:\n%v", err.Error(), string(stderr.Bytes()))
	}
	if stderr.String() != "" {
		log.Debugf("Stderr: %v", stderr.String())
	}
	return &stdout, nil
}
