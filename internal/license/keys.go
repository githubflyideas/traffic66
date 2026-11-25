package license

// Release is the set of issuing keys this build accepts, newest first.
//
// There are several on purpose. A licence cannot be withdrawn over the
// network — traffic66 never phones home — so if an issuing key ever leaks,
// the only way to stop licences signed with it is a release that no longer
// accepts it. The spare keys are already in the field for that day: their
// private halves are kept offline and have never signed anything, so one
// release swaps which key is in use without anyone having to be sent a new
// licence file.
//
// To withdraw a key, set Disabled and say why in Note; the licence page
// shows the note, so a customer sees "replaced on …" instead of what looks
// like a corrupt file.
var releaseKeys = []struct {
	Pub      string
	Disabled bool
	Note     string
}{
	{Pub: "MIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQAWS+sePS4omLSthsFwhUSg8kBWhvmXlBjUF77n5VmK5e2+XAOI6QcuaDYMreRB4/m9v7xco8WsFq3RURdmC25dscAeBFzJqt09OBWSSnmOYtu97sxs+9w7Raf5mJv3OARs7Z5nGldgWxeCSNvfkx7ChJ4ii8OXJ9Wj4bJGWu1ql91DQk="},
	{Pub: "MIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQA0Cp7Ah2bA3L/aWgSRKIvfg+sI7QRJt7uJMz8yM9BVoxc+V4hNNNi9+EOWTd6rxdUNngtcoz+YdJ5iENw/Ka64ZEB3jcAfuQBc9sEzeAbLy7Jqypg2KYyGF0BDr+JOD9FZm8sz+BQd0OnomOO0CoTLdY8/g2kVKDI6k6BMhFXJa2hgWE="},
	{Pub: "MIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQApsi5Pt/x6s+9FoxHu/zLE/vvFgqfeUXEkGLg/8SV9z8dSDjx0Gf3yx7r4gndWo+uMjyjuKYWnD+COqX8wOm6EDcAIGJcgUsZpdWbD5kH9x4qOObBAo7a1KthaZonwtP5gkvLB3AE2ROZlEoX0LSNhR0mgYTxs31vbaXgDCJh9wSBUbQ="},
}

// issued, when it holds anything, is the fingerprints of every licence
// actually issued, and nothing outside it is accepted. It is empty in a
// normal release, because filling it in means a new release for every
// licence issued afterwards. It is filled in by the release that answers a
// leaked key: everything the thief signs is then refused while every real
// customer's file keeps working untouched. `t66lic allowlist -go` prints
// it from the issuing ledger.
var issued []string

// Release returns the keyring this build verifies with.
func Release() Keyring {
	var r Keyring
	for _, k := range releaseKeys {
		pub, err := ParsePublicKey(k.Pub)
		if err != nil {
			continue // a malformed key in the source is not a licence problem
		}
		r.Keys = append(r.Keys, Key{ID: KeyID(pub), Pub: pub, Disabled: k.Disabled, Note: k.Note})
	}
	r.Allow = append(r.Allow, issued...)
	sortStrings(r.Allow)
	return r
}
