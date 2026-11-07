#!/usr/bin/env python3
"""Read the next versionCode from Google Play, or upload a signed AAB."""

import json
import os
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request

PACKAGE = "com.penndev.prism"
API = "https://androidpublisher.googleapis.com/androidpublisher/v3/applications/" + PACKAGE
UPLOAD = "https://androidpublisher.googleapis.com/upload/androidpublisher/v3/applications/" + PACKAGE


def b64(data: bytes) -> str:
    import base64

    return base64.urlsafe_b64encode(data).rstrip(b"=").decode()


def access_token(key_path: str) -> str:
    with open(key_path, encoding="utf-8") as f:
        key = json.load(f)
    now = int(time.time())
    header = b64(b'{"alg":"RS256","typ":"JWT"}')
    claims = json.dumps(
        {
            "iss": key["client_email"],
            "scope": "https://www.googleapis.com/auth/androidpublisher",
            "aud": "https://oauth2.googleapis.com/token",
            "iat": now,
            "exp": now + 3600,
        },
        separators=(",", ":"),
    ).encode()
    unsigned = f"{header}.{b64(claims)}".encode()
    with tempfile.NamedTemporaryFile("w", delete=False) as pem:
        pem.write(key["private_key"])
        pem_path = pem.name
    try:
        sig = subprocess.check_output(
            ["openssl", "dgst", "-sha256", "-sign", pem_path],
            input=unsigned,
        )
    finally:
        os.remove(pem_path)
    form = urllib.parse.urlencode(
        {
            "grant_type": "urn:ietf:params:oauth:grant-type:jwt-bearer",
            "assertion": unsigned.decode() + "." + b64(sig),
        }
    ).encode()
    body = request("POST", "https://oauth2.googleapis.com/token", form, "application/x-www-form-urlencoded")
    return json.loads(body)["access_token"]


def request(method: str, url: str, payload=None, content_type: str = "application/json", token: str = "") -> bytes:
    headers = {}
    if token:
        headers["Authorization"] = "Bearer " + token
    if content_type and payload is not None:
        headers["Content-Type"] = content_type
    req = urllib.request.Request(url, data=payload, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req) as resp:
            return resp.read()
    except urllib.error.HTTPError as err:
        detail = err.read().decode("utf-8", "replace")
        raise SystemExit(f"{method} {url} failed: {err.code} {detail}") from err


def open_edit(token: str) -> str:
    body = request("POST", API + "/edits", b"{}", token=token)
    return json.loads(body)["id"]


def delete_edit(token: str, edit_id: str) -> None:
    request("DELETE", f"{API}/edits/{edit_id}", token=token)


def next_code(key_path: str) -> None:
    token = access_token(key_path)
    edit_id = open_edit(token)
    try:
        body = request("GET", f"{API}/edits/{edit_id}/bundles", token=token)
    finally:
        delete_edit(token, edit_id)
    codes = [int(item["versionCode"]) for item in json.loads(body).get("bundles", [])]
    print(max(codes, default=0) + 1)


def upload(key_path: str, aab_path: str, version_name: str, track: str) -> None:
    token = access_token(key_path)
    edit_id = open_edit(token)
    try:
        with open(aab_path, "rb") as f:
            payload = f.read()
        uploaded = request(
            "POST",
            f"{UPLOAD}/edits/{edit_id}/bundles?uploadType=media",
            payload,
            "application/octet-stream",
            token,
        )
        version_code = str(json.loads(uploaded)["versionCode"])
        current = json.loads(request("GET", f"{API}/edits/{edit_id}/tracks/{track}", token=token))
        releases = list(current.get("releases") or [])
        drafts = [item.get("name") or item.get("versionCodes") for item in releases if item.get("status") == "draft"]
        if drafts:
            print(f"replacing existing draft on {track}: {drafts}")
        releases = [item for item in releases if item.get("status") != "draft"]
        releases.append(
            {
                "name": version_name,
                "versionCodes": [version_code],
                "status": "draft",
            }
        )
        request(
            "PUT",
            f"{API}/edits/{edit_id}/tracks/{track}",
            json.dumps({"track": track, "releases": releases}).encode(),
            token=token,
        )
        request("POST", f"{API}/edits/{edit_id}:commit", token=token)
    except BaseException:
        delete_edit(token, edit_id)
        raise
    print(f"uploaded {aab_path} as versionCode {version_code} to {track} (draft)")


def main() -> None:
    if len(sys.argv) < 3:
        raise SystemExit("usage: play_android.py next-code KEY | upload KEY AAB VERSION_NAME [TRACK]")
    cmd, key_path = sys.argv[1], sys.argv[2]
    if cmd == "next-code":
        next_code(key_path)
        return
    if cmd == "upload":
        if len(sys.argv) < 5:
            raise SystemExit("usage: play_android.py upload KEY AAB VERSION_NAME [TRACK]")
        track = sys.argv[5] if len(sys.argv) > 5 else "internal"
        upload(key_path, sys.argv[3], sys.argv[4], track)
        return
    raise SystemExit("unknown command")


if __name__ == "__main__":
    main()
