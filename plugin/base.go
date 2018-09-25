package plugin

import (
	"flag"

	dt "panorama/types"
)

type LogTailPlugin interface {
	ProvideFlags() *flag.FlagSet
	ValidateFlags() error
	Init() error
	ProvideEventParser() dt.EventParser
	ProvideObserverModule() dt.ObserverModule
}
