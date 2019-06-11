package aws

import (
	"context"

	"github.com/puppetlabs/wash/plugin"

	"github.com/aws/aws-sdk-go/aws/session"
	ec2Client "github.com/aws/aws-sdk-go/service/ec2"
)

// ec2Dir represents the resources/ec2 directory
type ec2Dir struct {
	plugin.EntryBase
	session *session.Session
	client  *ec2Client.EC2
	entries []plugin.Entry
}

func ec2DirBase() *ec2Dir {
	ec2Dir := &ec2Dir{
		EntryBase: plugin.NewEntryBase(),
	}
	ec2Dir.
		SetName("ec2").
		IsSingleton().
		DisableDefaultCaching()
	return ec2Dir
}

func newEC2Dir(session *session.Session) *ec2Dir {
	ec2Dir := ec2DirBase()
	ec2Dir.session = session
	ec2Dir.client = ec2Client.New(session)

	ec2Dir.entries = []plugin.Entry{
		newEC2InstancesDir(ec2Dir.session, ec2Dir.client),
	}

	return ec2Dir
}

func (e *ec2Dir) ChildSchemas() []plugin.EntrySchema {
	return plugin.ChildSchemas(ec2InstancesDirBase())
}

func (e *ec2Dir) List(ctx context.Context) ([]plugin.Entry, error) {
	return e.entries, nil
}
