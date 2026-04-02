package publisher

import "fmt"

type Event struct {
	Action string `json:"action"`
	Key    string `json:"key"`
}

type Publisher interface {
	Publish(event Event)
}

// StdoutPublisher is the dummy implementation for this challenge.
type StdoutPublisher struct{}

func NewStdoutPublisher() *StdoutPublisher {
	return &StdoutPublisher{}
}

func (p *StdoutPublisher) Publish(event Event) {
	fmt.Printf("Publishing {\"action\": %q, \"key\": %q}\n", event.Action, event.Key)
}
