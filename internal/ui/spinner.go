package ui

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
)

type Spinner struct {
	s *spinner.Spinner
}

func NewSpinner(message string) *Spinner {
	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond)
	s.Suffix = " " + message
	_ = s.Color("cyan")
	return &Spinner{s: s}
}

func (sp *Spinner) Start() {
	sp.s.Start()
}

func (sp *Spinner) Stop() {
	sp.s.Stop()
}

func (sp *Spinner) UpdateMessage(message string) {
	sp.s.Suffix = " " + message
}

func (sp *Spinner) Success(message string) {
	sp.s.Stop()
	fmt.Print(Success(message))
}

func (sp *Spinner) Error(message string) {
	sp.s.Stop()
	fmt.Print(Error(message))
}
