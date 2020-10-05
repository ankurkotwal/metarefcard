package sws

import (
	"regexp"

	"github.com/ankurkotwal/MetaRefCard/metarefcard/common"
)

var initiliased bool = false
var regexes map[string]*regexp.Regexp

// HandleRequest services the request to load files
func HandleRequest(files [][]byte, deviceMap common.DeviceMap,
	config *common.Config) (common.OverlaysByImage, map[string]string) {
	return nil, nil
}
