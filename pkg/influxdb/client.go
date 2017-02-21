package influxdb

import influx "github.com/influxdata/influxdb/client"

type Client struct {
	client          *influx.Client
	database        string
	retentionPolicy string
}

func NewClient(cfg influx.Config, db string, rp string) (*Client, error) {
	c, err := influx.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{
		client:          c,
		database:        db,
		retentionPolicy: rp,
	}, nil
}

func (c *Client) Send(points []influx.Point) error {
	bps := influx.BatchPoints{
		Points:          points,
		Database:        c.database,
		RetentionPolicy: c.retentionPolicy,
	}
	_, err := c.client.Write(bps)
	return err
}
