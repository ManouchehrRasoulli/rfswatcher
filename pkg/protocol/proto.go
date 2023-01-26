package protocol

import (
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/model"
	"time"
)

type Type int64

const (
	ChangeNotify Type = iota + 1
	RequestFile
	ResponseFile
	SubscribePath
	FilesList
)

/*
            A  con <----------------- Subscribe path  B
	        A Change Notify ------------------------> B
			A     <------------------- Request File   B
			A   File -------------------------------> B

	TODO :: Change protocol fix download entire file problem.
*/

// Data
// General communication frame in given protocol
type Data struct {
	Sec     uint64                 `json:"sc"`
	Time    time.Time              `json:"t"`
	Type    Type                   `json:"tp"`
	Heading map[string]interface{} `json:"h"`
	Payload interface{}            `json:"p"`
}

type FileMetaPayload struct {
	Path       string    `json:"p"`
	FileName   string    `json:"f"`
	Op         model.Op  `json:"op"`
	Size       int64     `json:"sz"`
	ChangeDate time.Time `json:"cd"`
}

type RequestFilePayload struct {
	Path       string    `json:"p"`
	FileName   string    `json:"f"`
	ChangeDate time.Time `json:"cd"`
}

type FileResponsePayload []byte

type SubscribePathPayload struct {
	Path string `json:"p"`
	Id   string `json:"id"`
}

type PathFiles struct {
	Path  string            `json:"p"`
	Files []FileMetaPayload `json:"fi"`
}
