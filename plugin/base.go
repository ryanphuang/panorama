package plugin

import (
	"flag"

	dt "deephealth/types"
)

type LogTailPlugin interface {
	ProvideFlags() *flag.FlagSet
	ValidateFlags() error
	Init() error
	ProvideEventParser() dt.EventParser
	ProvideObserverModule() dt.ObserverModule
}
