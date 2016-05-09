package mq

import (
	"github.com/go-ini/ini"
)

// NewServerOpts parses a file (or byteslice data) and returns ServerOpts
// Note: This takes the same argument types as ini.Load (see below for details):
// - Byteslice when it's direct data
// - String when it's referencing a file
func NewServerOpts(v interface{}) (opts ServerOpts, err error) {
	var f *ini.File
	if f, err = ini.Load(v); err != nil {
		return
	}

	if err = f.MapTo(&opts); err != nil {
		return
	}

	for _, sec := range f.Sections() {
		if sec.Name() == ini.DEFAULT_SECTION {
			continue
		}

		var key, tkn string
		if key = sec.Key("name").String(); len(key) == 0 {
			err = ErrEmptyName
			return
		}

		if tkn = sec.Key("token").String(); len(tkn) == 0 {
			err = ErrEmptyToken
			return
		}

		opts.Clients = append(opts.Clients, KeyToken{
			Key:   key,
			Token: tkn,
		})
	}

	return
}

// ServerOpts are used to call a new Server
type ServerOpts struct {
	Name string `ini:"name"`
	Loc  string `ini:"location"`

	Clients []KeyToken

	Op Operator
}

// NewClientOpts parses a file (or byteslice data) and returns ClientOpts
// Note: This takes the same argument types as ini.Load (see below for details):
// - Byteslice when it's direct data
// - String when it's referencing a file
func NewClientOpts(v interface{}) (opts ClientOpts, err error) {
	var f *ini.File
	if f, err = ini.Load(v); err != nil {
		return
	}

	if err = f.MapTo(&opts); err != nil {
		return
	}

	return
}

// ClientOpts are used to call a new Server
type ClientOpts struct {
	Name  string `ini:"name"`
	Token string `ini:"token"`
	Loc   string `ini:"location"`

	Op Operator
}
