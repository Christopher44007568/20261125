package stereofault

import "fmt"

// command is one recorded shared command.
type command struct {
	frame int
	eye   Eye
	layer LayerID
	kind  string
}

func (c command) String() string {
	return fmt.Sprintf("f%d/%s/%s/%s", c.frame, c.eye, c.layer, c.kind)
}

// CommandBuffer is the single shared recording stream for both eyes. The
// renderer guarantees that a layer can only append after it has passed
// resource validation, so failed-layer commands never reach this buffer.
type CommandBuffer struct {
	commands []command
}

func NewCommandBuffer() *CommandBuffer {
	return &CommandBuffer{}
}

func (cb *CommandBuffer) append(c command) {
	cb.commands = append(cb.commands, c)
}

// Len returns the number of recorded commands.
func (cb *CommandBuffer) Len() int { return len(cb.commands) }

// CommandsFor returns the recorded commands for one frame and eye.
func (cb *CommandBuffer) CommandsFor(frame int, eye Eye) []command {
	var out []command
	for _, c := range cb.commands {
		if c.frame == frame && c.eye == eye {
			out = append(out, c)
		}
	}
	return out
}

// ContainsLayer reports whether a layer ever recorded a command, optionally
// restricted to one frame. frame < 0 scans every frame.
func (cb *CommandBuffer) ContainsLayer(id LayerID, frame int) bool {
	for _, c := range cb.commands {
		if c.layer == id && (frame < 0 || c.frame == frame) {
			return true
		}
	}
	return false
}
