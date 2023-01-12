package data

import "time"

type Type int64

const (
	RequestListDirMeta Type = iota + 1
	ResponseListDirMeta
)

// Request
// General communication frame in given protocol
type Request struct {
	Sec     uint64
	RTime   time.Time
	Type    Type
	Heading map[string]interface{}
	Payload interface{}
}
