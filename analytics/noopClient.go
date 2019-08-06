package analytics

import log "github.com/sirupsen/logrus"

type noopClient struct {
	silenced bool
}

func (c *noopClient) Screenview(name string, params Params) error {
	s, err := newScreenview(name, params)
	if err != nil {
		return err
	}
	if !c.silenced {
		log.Debugf("Skipping submission of %v because analytics is disabled", s)
	}
	return nil
}

func (c *noopClient) Event(category string, action string, params Params) error {
	e, err := newEvent(category, action, params)
	if err != nil {
		return err
	}
	if !c.silenced {
		log.Debugf("Skipping submission of %v because analytics is disabled", e)
	}
	return nil
}

func (c *noopClient) Flush() {
	if !c.silenced {
		log.Debugf("Skipping flush because analytics is disabled")
	}
}
