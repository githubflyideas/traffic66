package geo

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"
)

// The built-in databases: DB-IP Lite country and ASN (CC BY 4.0), see
// dbip/README.md.
var (
	//go:embed dbip/country.mmdb.gz
	builtinCountry []byte
	//go:embed dbip/asn.mmdb.gz
	builtinASN []byte
)

// Builtin opens the built-in DB-IP Lite country and ASN databases.
func Builtin() (country, asn *Reader, err error) {
	open := func(gz []byte) (*Reader, error) {
		zr, err := gzip.NewReader(bytes.NewReader(gz))
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(zr)
		if err != nil {
			return nil, err
		}
		return FromBytes(b)
	}
	if country, err = open(builtinCountry); err != nil {
		return nil, nil, err
	}
	if asn, err = open(builtinASN); err != nil {
		return nil, nil, err
	}
	return country, asn, nil
}
