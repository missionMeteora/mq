package mq

import "github.com/missionMeteora/lockie"

// newAuth returns a pointer to a new instance of auth
func newAuth() *auth {
	return &auth{
		mux: lockie.NewLockie(),
		str: make(map[Chunk]Chunk),
	}
}

// auth controls the authentication portion of the mq handshake
type auth struct {
	mux lockie.Lockie

	// Keyed by user's key with a value of the user's token
	str map[Chunk]Chunk
}

// Get will return a token and ok (whether or not it exists) for a provided key
func (a *auth) Get(key Chunk) (token Chunk, ok bool) {
	a.mux.Lock()
	token, ok = a.str[key]
	a.mux.Unlock()
	return
}

// Put sets the store value for a provided key key to the provided token
func (a *auth) Put(key Chunk, token Chunk) {
	a.mux.Lock()
	a.str[key] = token
	a.mux.Unlock()
}

// Delete removes the entry in store which matches the provided key
func (a *auth) Delete(key Chunk) {
	a.mux.Lock()
	delete(a.str, key)
	a.mux.Unlock()
}

// IsValid will validate the provided credentials
func (a *auth) IsValid(h handshake) (ok bool) {
	var tkn Chunk
	if tkn, ok = a.str[h.key]; !ok {
		// If key doesn't exist within our store, return
		return
	}

	// If tokens match, return true
	return tkn == h.token
}
