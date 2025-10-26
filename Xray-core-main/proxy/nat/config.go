package nat

import (
	"google.golang.org/protobuf/proto"

	"github.com/xtls/xray-core/common/protocol"
)

func (c *Config) Equals(another protocol.Account) bool {
	if another == nil {
		return c == nil
	}

	thatNat, ok := another.(*Config)
	if !ok {
		return false
	}

	// Compare basic configuration
	if c.SiteId != thatNat.SiteId || c.UserLevel != thatNat.UserLevel {
		return false
	}

	// Compare virtual ranges
	if len(c.VirtualRanges) != len(thatNat.VirtualRanges) {
		return false
	}

	// Compare rules
	if len(c.Rules) != len(thatNat.Rules) {
		return false
	}

	// TODO: Implement detailed comparison of virtual ranges and rules
	// For now, just check counts

	return true
}

func (c *Config) ToProto() proto.Message {
	return c // Return the config itself as proto message
}