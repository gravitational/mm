package influxdb

import (
	"github.com/gravitational/trace"
	"github.com/influxdata/influxdb/client"
)

type Client struct {
	client          *client.Client
	database        string
	retentionPolicy string
}

func NewClient(cfg client.Config, db string, rp string) (*Client, error) {
	c, err := client.NewClient(cfg)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return &Client{
		client:          c,
		database:        db,
		retentionPolicy: rp,
	}, nil
}

func (c *Client) Send(points []client.Point) (*client.Response, error) {
	bps := client.BatchPoints{
		Points:          points,
		Database:        c.database,
		RetentionPolicy: c.retentionPolicy,
	}
	resp, err := c.client.Write(bps)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return resp, nil
}
