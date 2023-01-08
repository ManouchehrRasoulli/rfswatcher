package internal

import (
	"fmt"
	"strings"
)

const ExitName string = "exit"

type Op uint32

const (
	Create Op = 1 << iota
	Write
	Remove
	Rename
	Chmod
	Exit
)

type Event struct {
	Name string
	Op   Op
}

func (op Op) String() string {
	var b strings.Builder
	if op.Has(Exit) {
		b.WriteString("|EXIT_DAEMON")
	}
	if op.Has(Create) {
		b.WriteString("|CREATE")
	}
	if op.Has(Remove) {
		b.WriteString("|REMOVE")
	}
	if op.Has(Write) {
		b.WriteString("|WRITE")
	}
	if op.Has(Rename) {
		b.WriteString("|RENAME")
	}
	if op.Has(Chmod) {
		b.WriteString("|CHMOD")
	}
	if b.Len() == 0 {
		return "[no events]"
	}
	return b.String()[1:]
}

func (op Op) Has(h Op) bool { return op&h == h }

func (e Event) Has(op Op) bool { return e.Op.Has(op) }

func (e Event) String() string {
	return fmt.Sprintf("%-13s %q", e.Op.String(), e.Name)
}
