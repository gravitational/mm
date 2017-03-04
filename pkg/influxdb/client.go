package influxdb

import (
	"github.com/gravitational/trace"
	"github.com/influxdata/influxdb/client/v2"
)

type Client struct {
	client          client.Client
	database        string
	retentionPolicy string
}

func NewClient(cfg client.HTTPConfig, db string, rp string) (*Client, error) {
	c, err := client.NewHTTPClient(cfg)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return &Client{
		client:          c,
		database:        db,
		retentionPolicy: rp,
	}, nil
}

func (c *Client) Send(points []*client.Point) error {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:        c.database,
		RetentionPolicy: c.retentionPolicy,
	})
	bp.AddPoints(points)
	if err != nil {
		return trace.Wrap(err)
	}
	if err := c.client.Write(bp); err != nil {
		return trace.Wrap(err)
	}
	return nil
}
