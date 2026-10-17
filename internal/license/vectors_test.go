package license

import (
	"crypto/ed25519"
	"testing"
)

// The vectors below pin the signed message formats. traffic66 verifies
// licences with its own copy of this package, so this file is copied into
// that repository unchanged: if either side ever changes how a message is
// built, these signatures stop matching and the test fails on both sides,
// instead of a customer finding out that an issued licence does not load.
//
// The key is a fixed test seed. It is not, and must never be, an issuing
// key.
// testKeyID is the id derived from testPub.
const testKeyID = "65b60673"

const (
	testPub   = "ebVWLo/mVPlAeLES6KmLp5AfhTrmlb7X4OORC60ElmQ="
	sigV2     = "DL9Y+Ajkw3OsQmC4owY0NNiGUAnQSLYwbRdSwW2RI6PurXDtdE4Nlt/2sTq+jauT1wqU8oI0ayDR1h/QcDOeCA=="
	sigTrial  = "laCeKuijrFNXzPBOOjbKQS5DV7cvBn1ATrTDsh4cc7jfT4GuYSb9c2b3x41a+1M3wYL0+WtHyMNmktFZweE1BA=="
	sigPerp   = "yniSBg+/WiNbTj3Xwx4chq8r57ywRAn5UVRvSX7nKEmyrY+QvByaXQWOSomM4ZJ9CpOlIPypNRnqaW7TILDsBA=="
	sigLegacy = "frCOnEm7NHonJ4XGqUJ7q5ku4EH2hudVKyywEi1Aepkyw5ro/YMNUybZgTRk9CB+eeAETwLoqBa2rjUBFGtEBw=="
)

func str(v string) *string { return &v }

func testKey(t *testing.T) (ed25519.PrivateKey, ed25519.PublicKey) {
	t.Helper()
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	priv := ed25519.NewKeyFromSeed(seed)
	pub, err := ParsePublicKey(testPub)
	if err != nil {
		t.Fatal(err)
	}
	if !pub.Equal(priv.Public()) {
		t.Fatal("testPub does not belong to the test seed")
	}
	return priv, pub
}

// vector is one pinned licence.
type vector struct {
	name string
	file File
	sig  string
	ver  int
}

func vectors() []vector {
	return []vector{
		{"subscription", File{
			KeyID:    testKeyID,
			Customer: "株式会社サンプル", Product: "traffic66", License: TypeSubscription,
			Tier: TierStandard, Expires: str("2027-10-31"),
			MaxUsers: 25, MaxSites: 3, MaxRacks: 50, MaxDevices: 500,
			Features: []string{FeatSyslog, FeatDCIM, FeatTLS, FeatUsers, FeatLDAP, FeatPurgeIface},
		}, sigV2, 2},
		{"trial", File{
			KeyID:          testKeyID,
			InstallationID: "7f3a1c9e", Customer: "Example KK", Product: "traffic66",
			License: TypeTrial, Tier: TierTrial, Expires: str("2026-11-13"),
			MaxUsers: 3, MaxSites: 1, MaxRacks: 5, MaxDevices: 50,
			Features: Features,
		}, sigTrial, 2},
		{"perpetual", File{
			KeyID:    testKeyID,
			Customer: "Perpetual GmbH", Product: "traffic66", License: TypePerpetual,
			Tier: TierEnterprise, UpdatesUntil: str("2027-10-31"), Features: Features,
		}, sigPerp, 2},
		{"legacy v1", File{
			InstallationID: "7f3a1c9e", Customer: "Legacy Co", Product: "traffic66",
			License: "registration", Expires: str("2027-01-31"),
		}, sigLegacy, 1},
	}
}

func TestVectorsVerify(t *testing.T) {
	_, pub := testKey(t)
	if KeyID(pub) != testKeyID {
		t.Fatalf("key id = %q, want %q", KeyID(pub), testKeyID)
	}
	for _, v := range vectors() {
		f := v.file
		f.Signature = v.sig
		ver, err := Verify(pub, f)
		if err != nil {
			t.Fatalf("%s: %v", v.name, err)
		}
		if ver != v.ver {
			t.Fatalf("%s: matched message v%d, want v%d", v.name, ver, v.ver)
		}
	}
}

// TestVectorsStable catches a change to how a message is built.
func TestVectorsStable(t *testing.T) {
	priv, _ := testKey(t)
	for _, v := range vectors() {
		msg := Message(v.file)
		if v.ver == 1 {
			msg = MessageV1(v.file)
		}
		got := b64.EncodeToString(ed25519.Sign(priv, msg))
		if got != v.sig {
			t.Fatalf("%s: message changed\n got %s\nwant %s\nmessage:\n%s", v.name, got, v.sig, msg)
		}
	}
}
