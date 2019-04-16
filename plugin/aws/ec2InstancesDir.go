package aws

import (
	"context"

	awsSDK "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	ec2Client "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/puppetlabs/wash/activity"
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

	activity.Record(ctx, "Listing %v EC2 reservations", len(resp.Reservations))

	var entries []plugin.Entry
	for _, reservation := range resp.Reservations {
		activity.Record(
			ctx,
			"Listing %v instances in reservation %v",
			len(reservation.Instances),
			awsSDK.StringValue(reservation.ReservationId),
		)

		instances := make([]plugin.Entry, len(reservation.Instances))
		for i, instance := range reservation.Instances {
			instances[i] = newEC2Instance(
				ctx,
				instance,
				is.session,
				is.client,
			)
		}

		entries = append(entries, instances...)
	}

	return entries, nil
}
