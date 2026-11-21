# Security policy

## Reporting a vulnerability

Please report security problems privately, not in a public issue:

- Use GitHub's private vulnerability reporting on this repository
  (Security → Report a vulnerability), or
- if that is not available to you, open a regular issue that only says
  you have a security report and how to reach you — no details in public.

You will get an answer within a few days. A fix ships in a new release;
the release notes describe fixes in general terms, without the details
that would let someone attack installations that have not updated yet.

## Supported versions

Only the latest release is supported. traffic66 keeps no backports:
updating to the newest release is the fix path for every security
problem.

## Verifying a download

Every release carries build-provenance attestations. Check a download
with `sha256sum -c SHA256SUMS --ignore-missing`, or verify the
attestation itself with `gh attestation verify <file> -R githubflyideas/traffic66`
(SHA256SUMS and the SBOM are attested too).
