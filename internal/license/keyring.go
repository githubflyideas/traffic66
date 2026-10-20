package license

import (
	"crypto/ed25519"
	"sort"
)

// Key is one issuing key as a release knows it. Disabled keys stay in the
// release so a licence signed by one gives a clear answer instead of
// looking like a corrupt file.
type Key struct {
	ID       string
	Pub      ed25519.PublicKey
	Disabled bool
	Note     string // why it was disabled, for the licence page
}

// Keyring is what a release accepts. Allow, when it holds anything, is the
// fingerprints of the licences actually issued — taken from the issuing
// ledger — and nothing outside it is accepted. That is the answer to a
// leaked key: a release with the allow list switched on refuses everything
// the thief signs while every real customer's file keeps working, so the
// cost of the leak is an upgrade rather than reissuing every licence.
//
// Allow is empty in normal releases, because switching it on means a new
// release for every licence issued afterwards.
type Keyring struct {
	Keys  []Key
	Allow []string
}

// NewKeyring builds a ring from base64 public keys. Each key's id is
// derived from the key itself, so there is nothing to keep in step.
func NewKeyring(pubs []string, allow []string) (Keyring, error) {
	var r Keyring
	for _, s := range pubs {
		pub, err := ParsePublicKey(s)
		if err != nil {
			return Keyring{}, err
		}
		r.Keys = append(r.Keys, Key{ID: KeyID(pub), Pub: pub})
	}
	r.Allow = append(r.Allow, allow...)
	sort.Strings(r.Allow)
	return r, nil
}

// Allows reports whether f is in the allow list, or there is no list.
func (r Keyring) Allows(f File) bool {
	if len(r.Allow) == 0 {
		return true
	}
	fp := Fingerprint(f)
	i := sort.SearchStrings(r.Allow, fp)
	return i < len(r.Allow) && r.Allow[i] == fp
}

// Verify checks f against the ring and returns the key that signed it and
// the message version that matched.
//
// A licence naming a key is only ever checked against that key: a thief
// holding one private key cannot sign on another key's behalf, because the
// id is part of what is signed. A licence naming no key is from before key
// ids existed and is tried against every key still accepted.
func (r Keyring) Verify(f File) (Key, int, error) {
	if len(r.Keys) == 0 {
		return Key{}, 0, ErrNoKeys
	}
	if f.Signature == "" {
		return Key{}, 0, ErrNoSignature
	}
	if err := printable(f); err != nil {
		return Key{}, 0, err
	}
	sig, err := b64.DecodeString(f.Signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return Key{}, 0, ErrSignature
	}
	if !r.Allows(f) {
		return Key{}, 0, ErrNotAllowed
	}

	keys := r.Keys
	if f.KeyID != "" {
		k, ok := r.find(f.KeyID)
		if !ok {
			return Key{}, 0, ErrUnknownKey
		}
		if k.Disabled {
			return k, 0, ErrKeyDisabled
		}
		keys = []Key{k}
	}
	for _, k := range keys {
		if k.Disabled {
			continue
		}
		if len(k.Pub) != ed25519.PublicKeySize {
			continue
		}
		if ed25519.Verify(k.Pub, Message(f), sig) {
			return k, 3, nil
		}
		// A v2 licence does not say who it is made out to.
		if !namesLicensee(f) && ed25519.Verify(k.Pub, MessageV2(f), sig) {
			return k, 2, nil
		}
		// A v1 licence carries no key id, no quotas and no features; it
		// registers an installation and nothing more.
		if !namesLicensee(f) && !grants(f) && ed25519.Verify(k.Pub, MessageV1(f), sig) {
			return k, 1, nil
		}
	}
	return Key{}, 0, ErrSignature
}

func (r Keyring) find(id string) (Key, bool) {
	for _, k := range r.Keys {
		if k.ID == id {
			return k, true
		}
	}
	return Key{}, false
}

// SingleKey is the ring of one key, for issuing and for tests.
func SingleKey(pub ed25519.PublicKey) Keyring {
	return Keyring{Keys: []Key{{ID: KeyID(pub), Pub: pub}}}
}
