package config

import (
	"github.com/hashicorp/vault/helper/activedirectory"
)

type EngineConf struct {
	PasswordConf *PasswordConf
	ADConf       *activedirectory.Configuration
}

// Since *EngineConf will be nil if it's unset by the user
// or if its cached version has been invalidated,
// let's be super defensive around nil pointers here.
func (c *EngineConf) Map() map[string]interface{} {
	combined := make(map[string]interface{})
	if c == nil {
		return combined
	}
	if c.PasswordConf != nil {
		for k, v := range c.PasswordConf.Map() {
			combined[k] = v
		}
	}
	if c.ADConf != nil {
		for k, v := range c.ADConf.Map() {
			combined[k] = v
		}
	}
	return combined
}
