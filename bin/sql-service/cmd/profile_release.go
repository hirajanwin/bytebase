//go:build release
// +build release

package cmd

import (
	"github.com/bytebase/bytebase/common"
	server "github.com/bytebase/bytebase/sql-server"
)

func activeProfile() server.Profile {
	return server.Profile{
		Mode:                common.ReleaseModeProd,
		BackendHost:         flags.host,
		BackendPort:         flags.port,
		Debug:               flags.debug,
		Version:             version,
		GitCommit:           gitcommit,
		MetricConnectionKey: "",
	}
}
