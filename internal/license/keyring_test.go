package license

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha512"
	"crypto/x509"
	"testing"
)

func pair(t *testing.T) (*ecdsa.PrivateKey, Key) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pub := &priv.PublicKey
	return priv, Key{ID: KeyID(pub), Pub: pub}
}

// signMsg signs a message the way the issuing tool signs a licence:
// SHA-512 digest, ECDSA, ASN.1 DER.
func signMsg(t *testing.T, priv *ecdsa.PrivateKey, msg []byte) string {
	t.Helper()
	digest := sha512.Sum512(msg)
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return b64.EncodeToString(sig)
}

// signWith signs as the issuing tool does: the key's own id goes into the
// licence before it is signed.
func signWith(t *testing.T, priv *ecdsa.PrivateKey, f File) File {
	f.KeyID = KeyID(&priv.PublicKey)
	f.Signature = signMsg(t, priv, Message(f))
	return f
}

func pubB64(t *testing.T, pub *ecdsa.PublicKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return b64.EncodeToString(der)
}

func TestRingPicksTheNamedKey(t *testing.T) {
	privA, keyA := pair(t)
	_, keyB := pair(t)
	ring := Keyring{Keys: []Key{keyA, keyB}}
	f := signWith(t, privA, paid("2027-10-31"))
	got, ver, err := ring.Verify(f)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != keyA.ID || ver != 4 {
		t.Fatalf("got key %s v%d, want %s v4", got.ID, ver, keyA.ID)
	}
}

// The id is part of what is signed, so holding one private key does not
// let anyone sign on another key's behalf.
func TestCannotSignAsAnotherKey(t *testing.T) {
	privA, keyA := pair(t)
	_, keyB := pair(t)
	ring := Keyring{Keys: []Key{keyA, keyB}}

	// A licence claiming key B, signed with key A's private half.
	f := paid("2027-10-31")
	f.KeyID = keyB.ID
	f.Signature = signMsg(t, privA, Message(f))
	if _, _, err := ring.Verify(f); err == nil {
		t.Fatal("one key signed as another")
	}

	// And the same licence with A's id put back does not verify either,
	// because the id it was signed over was B's.
	f.KeyID = keyA.ID
	if _, _, err := ring.Verify(f); err == nil {
		t.Fatal("changing the id after signing was accepted")
	}
}

func TestUnknownKey(t *testing.T) {
	privA, _ := pair(t)
	_, keyB := pair(t)
	ring := Keyring{Keys: []Key{keyB}}
	f := signWith(t, privA, paid("2027-10-31"))
	if _, _, err := ring.Verify(f); err != ErrUnknownKey {
		t.Fatalf("got %v, want ErrUnknownKey", err)
	}
}

// Answering a leak: a release stops accepting the key. The licence must
// say so clearly rather than look like a damaged file.
func TestDisabledKey(t *testing.T) {
	privA, keyA := pair(t)
	keyA.Disabled, keyA.Note = true, "replaced on 2026-11-01"
	ring := Keyring{Keys: []Key{keyA}}
	f := signWith(t, privA, paid("2027-10-31"))
	if _, _, err := ring.Verify(f); err != ErrKeyDisabled {
		t.Fatalf("got %v, want ErrKeyDisabled", err)
	}
	s := Check(ring, f, "", now)
	if s.Registered() {
		t.Fatal("a licence from a withdrawn key must not register")
	}
	if !s.Watermark() {
		t.Fatal("and it must watermark")
	}
	if s.Reason == "" || !contains(s.Reason, "replaced on") {
		t.Fatalf("the reason should carry the note, got %q", s.Reason)
	}
}

func contains(hay, needle string) bool {
	for i := 0; i+len(needle) <= len(hay); i++ {
		if hay[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// The allow list is the other half of the answer to a leak: with it on,
// only the licences actually in the issuing ledger are accepted, so every
// real customer keeps working and nothing the thief signs does.
func TestAllowList(t *testing.T) {
	priv, key := pair(t)
	mine := signWith(t, priv, paid("2027-10-31"))
	theirs := signWith(t, priv, File{
		Customer: "Whoever", Product: "traffic66", License: TypePerpetual,
		Tier: TierEnterprise, Features: Features,
	})

	open := Keyring{Keys: []Key{key}}
	if _, _, err := open.Verify(theirs); err != nil {
		t.Fatalf("with no allow list everything this key signed is fine: %v", err)
	}

	ring, err := NewKeyring([]string{pubB64(t, key.Pub)}, []string{Fingerprint(mine)})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ring.Verify(mine); err != nil {
		t.Fatalf("a licence in the list was refused: %v", err)
	}
	if _, _, err := ring.Verify(theirs); err != ErrNotAllowed {
		t.Fatalf("got %v, want ErrNotAllowed", err)
	}
	s := Check(ring, theirs, "", now)
	if s.Registered() || !s.Watermark() {
		t.Fatal("a licence outside the list grants nothing")
	}
}

// A licence naming no key is tried against every key still accepted: the
// message's empty key id line is what was signed, so it cannot be pointed
// at one key after the fact either.
func TestNoKeyIDTriedAgainstEveryKey(t *testing.T) {
	priv, key := pair(t)
	_, other := pair(t)
	f := File{InstallationID: "7f3a1c9e5b2d4a6f8091a2b3c4d5e6f7", Customer: "Legacy Co", Product: "traffic66",
		License: TypePerpetual, Tier: TierStandard}
	f.Signature = signMsg(t, priv, Message(f))

	ring := Keyring{Keys: []Key{other, key}}
	got, ver, err := ring.Verify(f)
	if err != nil {
		t.Fatal(err)
	}
	if ver != 4 || got.ID != key.ID {
		t.Fatalf("got key %s v%d", got.ID, ver)
	}

	// Withdraw that key and the licence stops too.
	ring.Keys = []Key{other, {ID: key.ID, Pub: key.Pub, Disabled: true}}
	if _, _, err := ring.Verify(f); err == nil {
		t.Fatal("a withdrawn key still accepted a licence")
	}
}

func TestEmptyRing(t *testing.T) {
	priv, _ := pair(t)
	f := signWith(t, priv, paid("2027-10-31"))
	if _, _, err := (Keyring{}).Verify(f); err != ErrNoKeys {
		t.Fatalf("got %v, want ErrNoKeys", err)
	}
	if s := Check(Keyring{}, f, "", now); s.Registered() {
		t.Fatal("a release with no keys registers nothing")
	}
}

func TestKeyIDIsStable(t *testing.T) {
	_, pub := testKey(t)
	if KeyID(pub) != testKeyID {
		t.Fatalf("key id = %q, want %q", KeyID(pub), testKeyID)
	}
	if len(KeyID(pub)) != 8 {
		t.Fatalf("a key id should be short enough to quote, got %q", KeyID(pub))
	}
}
