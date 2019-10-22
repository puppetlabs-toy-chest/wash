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
}

func newEC2Dir(session *session.Session) *ec2Dir {
	ec2Dir := &ec2Dir{
		EntryBase: plugin.NewEntry("ec2"),
	}
	ec2Dir.DisableDefaultCaching()
	ec2Dir.session = session
	ec2Dir.client = ec2Client.New(session)
	return ec2Dir
}

func (e *ec2Dir) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(e, "ec2").IsSingleton()
}

func (e *ec2Dir) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&ec2InstancesDir{}).Schema(),
	}
}

func (e *ec2Dir) List(ctx context.Context) ([]plugin.Entry, error) {
	return []plugin.Entry{newEC2InstancesDir(ctx, e.session, e.client)}, nil
}
