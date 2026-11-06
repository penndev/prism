#!/usr/bin/env python3
"""Print the next iOS CFBundleVersion for com.penndev.prism."""

import json
import subprocess
import sys
import time
import urllib.error
import urllib.request

BUNDLE_ID = "com.penndev.prism"
API = "https://api.appstoreconnect.apple.com"


def b64(data: bytes) -> str:
    import base64

    return base64.urlsafe_b64encode(data).rstrip(b"=").decode()


def der_to_raw(sig: bytes) -> bytes:
    if len(sig) < 8 or sig[0] != 0x30:
        raise SystemExit("openssl did not return an ECDSA signature")
    i = 2
    if sig[1] & 0x80:
        i = 2 + (sig[1] & 0x7F)
    if sig[i] != 0x02:
        raise SystemExit("bad ECDSA signature")
    rlen = sig[i + 1]
    r = sig[i + 2 : i + 2 + rlen]
    i = i + 2 + rlen
    if sig[i] != 0x02:
        raise SystemExit("bad ECDSA signature")
    slen = sig[i + 1]
    s = sig[i + 2 : i + 2 + slen]
    return r[-32:].rjust(32, b"\x00") + s[-32:].rjust(32, b"\x00")


def token(key_id: str, issuer_id: str, key_path: str) -> str:
    now = int(time.time())
    header = b64(json.dumps({"alg": "ES256", "kid": key_id, "typ": "JWT"}, separators=(",", ":")).encode())
    claims = b64(
        json.dumps(
            {"iss": issuer_id, "iat": now, "exp": now + 600, "aud": "appstoreconnect-v1"},
            separators=(",", ":"),
        ).encode()
    )
    unsigned = f"{header}.{claims}".encode()
    sig = subprocess.check_output(["openssl", "dgst", "-sha256", "-sign", key_path], input=unsigned)
    return unsigned.decode() + "." + b64(der_to_raw(sig))


def get(url: str, bearer: str) -> dict:
    req = urllib.request.Request(url, headers={"Authorization": "Bearer " + bearer})
    try:
        with urllib.request.urlopen(req) as resp:
            return json.loads(resp.read().decode())
    except urllib.error.HTTPError as err:
        detail = err.read().decode("utf-8", "replace")
        raise SystemExit(f"GET {url} failed: {err.code} {detail}") from err


def next_build(key_id: str, issuer_id: str, key_path: str) -> None:
    bearer = token(key_id, issuer_id, key_path)
    apps = get(f"{API}/v1/apps?filter[bundleId]={BUNDLE_ID}&limit=1", bearer)
    data = apps.get("data") or []
    if not data:
        print(1)
        return
    app_id = data[0]["id"]
    highest = 0
    url = f"{API}/v1/builds?filter[app]={app_id}&fields[builds]=version&limit=200"
    while url:
        page = get(url, bearer)
        for item in page.get("data") or []:
            version = str((item.get("attributes") or {}).get("version") or "")
            if version.isdigit():
                highest = max(highest, int(version))
        url = ((page.get("links") or {}).get("next")) or ""
    print(highest + 1)


def main() -> None:
    if len(sys.argv) != 4:
        raise SystemExit("usage: appstore_build.py KEY_ID ISSUER_ID AuthKey.p8")
    next_build(sys.argv[1], sys.argv[2], sys.argv[3])


if __name__ == "__main__":
    main()
