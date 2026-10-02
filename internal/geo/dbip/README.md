# Built-in country and ASN databases

`country.mmdb.gz` and `asn.mmdb.gz` are DB-IP's free "IP to Country Lite"
and "IP to ASN Lite" databases (https://db-ip.com), as packaged in MaxMind
DB format by the ip-location-db project
(npm `@ip-location-db/dbip-country-mmdb` 2.3.2026060120 and
`@ip-location-db/dbip-asn-mmdb` 2.3.2026060407), compressed with gzip.

They are licensed under the Creative Commons Attribution 4.0 International
License (https://creativecommons.org/licenses/by/4.0/). Attribution: IP
Geolocation by DB-IP (https://db-ip.com). traffic66 shows this attribution
on the pages that use the data.

To update them, replace both files with newer versions of the same packages
(`npm pack @ip-location-db/dbip-country-mmdb @ip-location-db/dbip-asn-mmdb`,
then `gzip -9c dbip-country.mmdb > country.mmdb.gz` and the same for ASN).
Users can also update them at run time on the Sources page.
