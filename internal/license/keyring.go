package license

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha512"
	"encoding/asn1"
	"math/big"
	"sort"
)

// Key is one issuing key as a release knows it. Disabled keys stay in the
// release so a licence signed by one gives a clear answer instead of
// looking like a corrupt file.
type Key struct {
	ID       string
	Pub      *ecdsa.PublicKey
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

// ecdsaSig is the ASN.1 form of a signature, as it sits in license.json
// under base64: a sequence of the two integers the algorithm produces.
// DER signatures vary in length — 138 or 139 bytes for P-521 — so there
// is no fixed size to check against; parsing the DER and finding both
// integers inside the curve's order is the check.
type ecdsaSig struct {
	R, S *big.Int
}

func parseSignature(b []byte) (r, s *big.Int, ok bool) {
	var sig ecdsaSig
	rest, err := asn1.Unmarshal(b, &sig)
	if err != nil || len(rest) != 0 || sig.R == nil || sig.S == nil {
		return nil, nil, false
	}
	n := elliptic.P521().Params().N
	if sig.R.Sign() <= 0 || sig.S.Sign() <= 0 || sig.R.Cmp(n) >= 0 || sig.S.Cmp(n) >= 0 {
		return nil, nil, false
	}
	return sig.R, sig.S, true
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
	if err != nil {
		return Key{}, 0, ErrSignature
	}
	sigR, sigS, ok := parseSignature(sig)
	if !ok {
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
	digest := sha512.Sum512(Message(f))
	for _, k := range keys {
		if k.Disabled {
			continue
		}
		if !isP521(k.Pub) {
			continue
		}
		if ecdsa.Verify(k.Pub, digest[:], sigR, sigS) {
			return k, 4, nil
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
func SingleKey(pub *ecdsa.PublicKey) Keyring {
	return Keyring{Keys: []Key{{ID: KeyID(pub), Pub: pub}}}
}
