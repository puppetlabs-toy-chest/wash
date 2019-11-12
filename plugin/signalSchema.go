package plugin

import (
	"encoding/json"
	"regexp"
	"strings"
)

// SignalSchema represents a given signal/signal group's schema
type SignalSchema struct {
	signalSchema
	regex *regexp.Regexp
}

// Name returns the signal/signal group's name
func (s *SignalSchema) Name() string {
	return s.signalSchema.Name
}

// SetName sets the signal/signal group's name. This should only
// be called by the tests.
func (s *SignalSchema) SetName(name string) *SignalSchema {
	s.signalSchema.Name = name
	return s
}

// Description returns the signal/signal group's name
func (s *SignalSchema) Description() string {
	return s.signalSchema.Description
}

// SetDescription sets the signal/signal group's description. This should
// only be called by the tests.
func (s *SignalSchema) SetDescription(description string) *SignalSchema {
	s.signalSchema.Description = description
	return s
}

// Regex returns a regex describing an arbitrary signal in the signal
// group.
func (s *SignalSchema) Regex() *regexp.Regexp {
	return s.regex
}

// SetRegex sets the signal group's regex. This should only be called by the
// tests.
func (s *SignalSchema) SetRegex(regex *regexp.Regexp) *SignalSchema {
	s.regex = regex
	s.signalSchema.Regex = regex.String()
	return s
}

// IsGroup returns true if s is a signal group's schema. This is
// true if s.Regex() != nil
func (s *SignalSchema) IsGroup() bool {
	return s.Regex() != nil
}

// MarshalJSON marshals the signal schema to JSON. It takes
// a value receiver so that the entry schema's still marshalled
// when it's referenced as an interface{} object. See
// https://stackoverflow.com/a/21394657 for more details.
func (s SignalSchema) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.signalSchema)
}

// UnmarshalJSON unmarshals the signal schema JSON.
func (s *SignalSchema) UnmarshalJSON(bytes []byte) error {
	err := json.Unmarshal(bytes, &s.signalSchema)
	if err != nil {
		return err
	}
	s.signalSchema.Name = strings.ToLower(s.signalSchema.Name)
	if len(s.signalSchema.Regex) > 0 {
		// (?i) tells Go that the regex is case-insensitive
		s.signalSchema.Regex = "(?i)" + s.signalSchema.Regex
		s.regex, err = regexp.Compile(s.signalSchema.Regex)
	}
	return err
}

// This is to implement JSON Marshal/Unmarshal. The main reason
// for implementing the Marshaler/Unmarshaler interfaces is to
// validate the marshalled Regex.
type signalSchema struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Regex       string `json:"regex,omitempty"`
}
