package license

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha512"
	"math/big"
	"testing"
)

// The vectors below pin the signed message format. traffic66 verifies
// licences with its own copy of this package, so this file is copied into
// that repository unchanged: if either side ever changes how a message is
// built, these signatures stop matching and the test fails on both sides,
// instead of a customer finding out that an issued licence does not load.
//
// The key is a fixed test key, derived below from a fixed seed. It is
// not, and must never be, an issuing key.
//
// ECDSA signatures are not reproducible byte for byte — the algorithm
// takes a random nonce — so where the Ed25519 vectors could be re-signed
// and compared, these pin the message bytes themselves (msg, base64) and
// the signature made over them: a change to the message breaks the byte
// comparison directly, and a change to what is signed breaks the
// verification.
// testKeyID is the id derived from testPub.
const testKeyID = "1f12fab7"

const (
	testPub   = "MIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQBcEK5MkI9aB0tR3vE/7agvVgyXcs2WfOwSt12yXmTFo+oouFczdS/BxJJppym31L8qIizjHab1qqDt3lHf4eosjYANLIssEzkIsHkV7gaxyGoGx/UwHICVZgRjXdZzzw0B1UorI2hgn8b4SKKqd3v/nLqxl4Y9vOgiikSnGWWY7KooEg="
	msgOrg    = "dHJhZmZpYzY2LWxpY2Vuc2UtdjQKMWYxMmZhYjcKCkJpZyBDbwpvcmdhbml6YXRpb24KdHJhZmZpYzY2CnBlcnBldHVhbAplbnRlcnByaXNlCm51bGwKMjAyNy0xMC0zMQowCjAKMAowCmRjaW0sbGRhcCxtdWx0aS1sb2dvLHB1cmdlLWlmYWNlLHN5c2xvZyx0bHMsdXNlcnMsd2hpdGUtbGFiZWwK"
	sigOrg    = "MIGIAkIBS6TdN10gqHzy67mR/6/YzzYP8flkwpZbbf1rUMbdRlyV91NqRhz7k7eC+bj2Ve6stGzCnjLfoRoQsg8YmCLFTUICQgHFrXN9HdwkOFnYm2vxILMkx58DTZ5B8ffij/pUR28CtKax3BBZ9zkL0gJdLaHunQOuSw/taWm9TrgOKavoC54Ikw=="
	msgPerson = "dHJhZmZpYzY2LWxpY2Vuc2UtdjQKMWYxMmZhYjcKN2YzYTFjOWU1YjJkNGE2ZjgwOTFhMmIzYzRkNWU2ZjcKQWRhIExvdmVsYWNlCmluZGl2aWR1YWwKdHJhZmZpYzY2CnN1YnNjcmlwdGlvbgp0ZWFtCjIwMjctMTAtMzEKbnVsbAoxMAoxCjEwCjEwMApkY2ltLHN5c2xvZwo="
	sigPerson = "MIGHAkIBQUtpgq2Y4N42zu3pHkUrhfYdxx5ykFqC5Iqy477drqzvtIPuEkuqrqcPwxprN0+d2dfYayWleZh89hLGfMv9gMsCQWmNy0c2oyjGp4hXDNv6m8ppDeteoSgBLYrR7ouWat2ISbknfVKPvmiUiyBSrXv70+JsfXUJda6lcqVFltFtLclD"
	msgSub    = "dHJhZmZpYzY2LWxpY2Vuc2UtdjQKMWYxMmZhYjcKCuagquW8j+S8muekvuOCteODs+ODl+ODqwoKdHJhZmZpYzY2CnN1YnNjcmlwdGlvbgpzdGFuZGFyZAoyMDI3LTEwLTMxCm51bGwKMjUKMwo1MAo1MDAKZGNpbSxsZGFwLHB1cmdlLWlmYWNlLHN5c2xvZyx0bHMsdXNlcnMK"
	sigSub    = "MIGIAkIBLGjPpNsPyLJ5oN2TIfmIs6QuV35EmdA4MOZUbDVZwYGBdvnDdrVq1NU6eNowEgNfTSmozUyKdlMR6YBv9ShvrwgCQgCW18ChjHTNvHip8XlLEhj9oABKQ/Gi/bXSwXcJ7ceELCP+g8rp5FqZMaqQNZUzTG0JDK3iHSdHZmD+dVutHcbtUA=="
	msgTrial  = "dHJhZmZpYzY2LWxpY2Vuc2UtdjQKMWYxMmZhYjcKN2YzYTFjOWU1YjJkNGE2ZjgwOTFhMmIzYzRkNWU2ZjcKRXhhbXBsZSBLSwoKdHJhZmZpYzY2CnRyaWFsCnRyaWFsCjIwMjYtMTEtMTMKbnVsbAozCjEKNQo1MApkY2ltLGxkYXAsbXVsdGktbG9nbyxwdXJnZS1pZmFjZSxzeXNsb2csdGxzLHVzZXJzLHdoaXRlLWxhYmVsCg=="
	sigTrial  = "MIGIAkIBi4Vh/ht7x14aZQSqKBlTLDRLVsaWQpNGqR+PtcGpQDZC8t5ytWjx3pb0nvi5njPtS78UcR7xhRvBxXgLXHBJzJsCQgGprYd6yYSaZKyq9lyb10BuNtE8ste+mFAGiDDcpff5wryUFn8Nixq1RdVf9xJqTToA42PbUWd+9wvc8aVbVdxs3A=="
	msgPerp   = "dHJhZmZpYzY2LWxpY2Vuc2UtdjQKMWYxMmZhYjcKClBlcnBldHVhbCBHbWJICgp0cmFmZmljNjYKcGVycGV0dWFsCmVudGVycHJpc2UKbnVsbAoyMDI3LTEwLTMxCjAKMAowCjAKZGNpbSxsZGFwLG11bHRpLWxvZ28scHVyZ2UtaWZhY2Usc3lzbG9nLHRscyx1c2Vycyx3aGl0ZS1sYWJlbAo="
	sigPerp   = "MIGHAkIA+ipKh+cUMKkLp7bRuGycSMPBCwTsUocrTsSwTvV8uJ/qcFzCzdESftoPnVCcAQ/zgTSWjLpAOZw1RkpprbBpMu4CQSfhfkTlWq9UhvBfEzUJvlkNWbLBuK1dfWfkaYIFmhArCPDDx87JRSNJvq6fW2z083fmgGsFavVbW6UI0BixXfj6"
)

func str(v string) *string { return &v }

// testKey derives the fixed test key: the seed is the bytes 1 to 32, and
// the private scalar is the seed's SHA-512 taken modulo the curve order.
// Anyone can recompute it, which is the point — it must never sign a real
// licence.
func testKey(t *testing.T) (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	t.Helper()
	seed := make([]byte, 32)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	sum := sha512.Sum512(seed)
	n := elliptic.P521().Params().N
	d := new(big.Int).SetBytes(sum[:])
	d.Mod(d, new(big.Int).Sub(n, big.NewInt(1))).Add(d, big.NewInt(1))
	// The public point comes out of crypto/ecdh: the elliptic package's
	// scalar multiplication is deprecated, and this route yields the
	// same point for the same scalar.
	dBytes := make([]byte, 66)
	d.FillBytes(dBytes)
	ekey, err := ecdh.P521().NewPrivateKey(dBytes)
	if err != nil {
		t.Fatal(err)
	}
	point := ekey.PublicKey().Bytes() // 0x04, then X and Y, 66 bytes each
	x := new(big.Int).SetBytes(point[1:67])
	y := new(big.Int).SetBytes(point[67:133])
	priv := &ecdsa.PrivateKey{PublicKey: ecdsa.PublicKey{Curve: elliptic.P521(), X: x, Y: y}, D: d}
	pub, err := ParsePublicKey(testPub)
	if err != nil {
		t.Fatal(err)
	}
	if !pub.Equal(&priv.PublicKey) {
		t.Fatal("testPub does not belong to the test seed")
	}
	return priv, pub
}

// vector is one pinned licence.
type vector struct {
	name string
	file File
	msg  string // Message(file), base64
	sig  string
	ver  int
}

func vectors() []vector {
	return []vector{
		// v4: ECDSA P-521, the licence says who it is made out to
		{"organization", File{
			KeyID:    testKeyID,
			Customer: "Big Co", Licensee: LicenseeOrganization, Product: "traffic66", License: TypePerpetual,
			Tier: TierEnterprise, UpdatesUntil: str("2027-10-31"), Features: Features,
		}, msgOrg, sigOrg, 4},
		{"individual", File{
			KeyID:          testKeyID,
			InstallationID: "7f3a1c9e5b2d4a6f8091a2b3c4d5e6f7", Customer: "Ada Lovelace", Licensee: LicenseeIndividual,
			Product: "traffic66", License: TypeSubscription, Tier: TierTeam, Expires: str("2027-10-31"),
			MaxUsers: 10, MaxSites: 1, MaxRacks: 10, MaxDevices: 100,
			Features: []string{FeatSyslog, FeatDCIM},
		}, msgPerson, sigPerson, 4},
		{"subscription", File{
			KeyID:    testKeyID,
			Customer: "株式会社サンプル", Product: "traffic66", License: TypeSubscription,
			Tier: TierStandard, Expires: str("2027-10-31"),
			MaxUsers: 25, MaxSites: 3, MaxRacks: 50, MaxDevices: 500,
			Features: []string{FeatSyslog, FeatDCIM, FeatTLS, FeatUsers, FeatLDAP, FeatPurgeIface},
		}, msgSub, sigSub, 4},
		{"trial", File{
			KeyID:          testKeyID,
			InstallationID: "7f3a1c9e5b2d4a6f8091a2b3c4d5e6f7", Customer: "Example KK", Product: "traffic66",
			License: TypeTrial, Tier: TierTrial, Expires: str("2026-11-13"),
			MaxUsers: 3, MaxSites: 1, MaxRacks: 5, MaxDevices: 50,
			Features: Features,
		}, msgTrial, sigTrial, 4},
		{"perpetual", File{
			KeyID:    testKeyID,
			Customer: "Perpetual GmbH", Product: "traffic66", License: TypePerpetual,
			Tier: TierEnterprise, UpdatesUntil: str("2027-10-31"), Features: Features,
		}, msgPerp, sigPerp, 4},
	}
}

// vectorNamed finds one pinned licence. Tests ask for it by name: the
// list grows at the front as message versions are added, and an index
// would quietly become a different licence.
func vectorNamed(t *testing.T, name string) vector {
	t.Helper()
	for _, v := range vectors() {
		if v.name == name {
			return v
		}
	}
	t.Fatalf("no pinned licence called %q", name)
	return vector{}
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

// TestVectorsStable catches a change to how a message is built: the
// pinned bytes are what the issuing side signed, so the message this
// build constructs must equal them exactly.
func TestVectorsStable(t *testing.T) {
	for _, v := range vectors() {
		got := b64.EncodeToString(Message(v.file))
		if got != v.msg {
			t.Fatalf("%s: message changed\n got %s\nwant %s\nmessage:\n%s", v.name, got, v.msg, Message(v.file))
		}
	}
}
