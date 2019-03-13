package aws

import (
	"context"

	awsSDK "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	ec2Client "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
)

// ec2InstancesDir represents the ec2/instances
// directory
//
// NOTE: Could re-structure this as ec2/instances/reservations/<instance>.
// No need to do this now since there's no clear use-case for it yet.
type ec2InstancesDir struct {
	plugin.EntryBase
	session *session.Session
	client  *ec2Client.EC2
}

func newEC2InstancesDir(session *session.Session, client *ec2Client.EC2) *ec2InstancesDir {
	ec2InstancesDir := &ec2InstancesDir{
		EntryBase: plugin.NewEntry("instances"),
		session:   session,
		client:    client,
	}

	return ec2InstancesDir
}

func (is *ec2InstancesDir) List(ctx context.Context) ([]plugin.Entry, error) {
	resp, err := is.client.DescribeInstancesWithContext(ctx, nil)
	if err != nil {
		return nil, err
	}

	journal.Record(ctx, "Listing %v EC2 reservations", len(resp.Reservations))

	var entries []plugin.Entry
	for _, reservation := range resp.Reservations {
		journal.Record(
			ctx,
			"Listing %v instances in reservation %v",
			len(reservation.Instances),
			awsSDK.StringValue(reservation.ReservationId),
		)

		instances := make([]plugin.Entry, len(reservation.Instances))
		for i, instance := range reservation.Instances {
			// AWS does not include the EC2 instance's ctime in its
			// metadata. It also does not include the EC2 instance's
			// last state transition time (mtime). Thus, we try to "guess"
			// reasonable values for ctime and mtime by looping over each
			// block device's attachment time and the instance's launch time.
			// The oldest of these times is the ctime; the newest is the mtime.
			var attr plugin.Attributes
			attr.Ctime = awsSDK.TimeValue(instance.LaunchTime)
			attr.Mtime = attr.Ctime
			for _, mapping := range instance.BlockDeviceMappings {
				attachTime := awsSDK.TimeValue(mapping.Ebs.AttachTime)

				if attachTime.Before(attr.Ctime) {
					attr.Ctime = attachTime
				}

				if attachTime.After(attr.Mtime) {
					attr.Mtime = attachTime
				}
			}

			instances[i] = newEC2Instance(
				ctx,
				awsSDK.StringValue(instance.InstanceId),
				is.session,
				is.client,
				attr,
			)
		}

		entries = append(entries, instances...)
	}

	return entries, nil
}
