#!/usr/bin/env python3
"""Runtime-only probe against a disposable local BDSPro and Mailpit."""
import http.cookiejar
import json
import os
import re
import secrets
import subprocess
import time
import urllib.error
import urllib.request
import uuid

BASE = "http://127.0.0.1:8080"
MAILPIT = "http://127.0.0.1:8025"


def get(url):
    with urllib.request.urlopen(url, timeout=5) as response:
        return json.load(response)


def browser():
    return urllib.request.build_opener(
        urllib.request.ProxyHandler({}),
        urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()),
    )


def post(opener, path, body, expected):
    request = urllib.request.Request(
        BASE + "/api/v1/auth/" + path,
        data=json.dumps(body).encode(), method="POST",
        headers={"Content-Type": "application/json", "Origin": BASE,
                 "X-BDS-Request": "1", "Sec-Fetch-Site": "same-origin"},
    )
    try:
        response = opener.open(request, timeout=15)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        data = json.load(response)
        if response.status != expected:
            raise RuntimeError(f"{path}: HTTP {response.status}, expected {expected}")
        print(f"PASS POST /api/v1/auth/{path} {response.status}", flush=True)
        return data


def inbox_code(recipient, pattern, ttl_seconds, previous_id=None):
    for _ in range(30):
        messages = get(MAILPIT + "/api/v1/messages").get("messages", [])
        for message in messages:
            if message["ID"] != previous_id and any(to.get("Address") == recipient for to in message.get("To", [])):
                content = get(MAILPIT + "/api/v1/message/" + message["ID"])
                body = content.get("Text", "")
                match = re.search(pattern, body)
                if match:
                    if f"expires in {ttl_seconds // 60} minutes" not in body.lower():
                        raise RuntimeError("Delivery text disagrees with API TTL")
                    return match.group(1), message["ID"]
        time.sleep(0.2)
    raise RuntimeError("Expected verification message missing from development inbox")


def db_query(sql):
    container = os.environ.get("BDS_PROBE_PG_CONTAINER")
    if not container:
        return None
    result = subprocess.run(
        ["docker", "exec", container, "psql", "-X", "-qAt", "-U", "root", "-d", "bdspro",
         "-v", "ON_ERROR_STOP=1", "-c", sql],
        check=True, capture_output=True, text=True,
    )
    return result.stdout.strip()


def check_challenge(challenge, kind):
    if challenge.get("expires_in") != 300 or challenge.get("resend_after") != 60:
        raise RuntimeError("Unexpected challenge policy")
    # UUID parsing restricts the interpolated fixture ID to a canonical UUID.
    challenge_id = str(uuid.UUID(challenge["challenge_id"]))
    row = db_query(
        f"SELECT extract(epoch FROM expires_at-clock_timestamp()), "
        f"extract(epoch FROM resend_after-clock_timestamp()) FROM verifications "
        f"WHERE kind='{kind}' AND id='{challenge_id}'::uuid AND proof_digest IS NULL"
    )
    if row is not None:
        remaining, cooldown = map(float, row.split("|"))
        if not 285 <= remaining <= 300 or not 45 <= cooldown <= 60:
            raise RuntimeError("Stored challenge deadline disagrees with API policy")
        if abs((remaining - cooldown) - 240) > 0.1:
            raise RuntimeError("Stored OTP TTL/cooldown disagreement")


def check_proof(challenge, result, kind):
    if result.get("status") != "registration_required" or result.get("expires_in") != 900:
        raise RuntimeError("Expected registration proof with canonical lifetime")
    challenge_id = str(uuid.UUID(challenge["challenge_id"]))
    row = db_query(
        f"SELECT extract(epoch FROM expires_at-clock_timestamp()) FROM verifications "
        f"WHERE kind='{kind}' AND id='{challenge_id}'::uuid AND proof_digest IS NOT NULL"
    )
    if row is not None and not 885 <= float(row) <= 900:
        raise RuntimeError("Stored registration proof deadline disagrees with API policy")


def main():
    capabilities = get(BASE + "/api/v1/auth/phone/channels")
    if capabilities.get("development") is not True or capabilities.get("channels") != ["sms", "zalo", "telegram"]:
        raise RuntimeError("Refusing delivery probe outside expected Mailpit development mode")
    get(MAILPIT + "/api/v1/info")

    email = "probe-" + secrets.token_hex(6) + "@example.invalid"
    opener = browser()
    challenge = post(opener, "register/start", {"email": email}, 202)
    check_challenge(challenge, "email")
    code, message_id = inbox_code(email, r"Enter this verification code in BDSPro: (\d{8})", challenge["expires_in"])
    post(opener, "register/resend", {"challenge_id": challenge["challenge_id"]}, 429)
    if os.environ.get("BDS_PROBE_PG_CONTAINER"):
        # Advance only the disposable fixture's resend deadline, not the production policy.
        challenge_id = str(uuid.UUID(challenge["challenge_id"]))
        db_query(f"UPDATE verifications SET resend_after=clock_timestamp()-interval '1 second' "
                 f"WHERE kind='email' AND id='{challenge_id}'::uuid")
        rotated = post(opener, "register/resend", {"challenge_id": challenge["challenge_id"]}, 202)
        if rotated["challenge_id"] != challenge["challenge_id"]:
            raise RuntimeError("Resend changed challenge identity")
        check_challenge(rotated, "email")
        code, _ = inbox_code(email, r"Enter this verification code in BDSPro: (\d{8})", rotated["expires_in"], message_id)
    result = post(opener, "register/verify", {"challenge_id": challenge["challenge_id"], "code": code}, 200)
    check_proof(challenge, result, "email")
    print("PASS email registration SMTP delivery and verification", flush=True)

    for channel in capabilities["channels"]:
        phone = "+8490" + "".join(secrets.choice("0123456789") for _ in range(7))
        opener = browser()
        challenge = post(opener, "phone/start", {"phone": phone, "channel": channel}, 202)
        if challenge.get("delivery_channel") != channel:
            raise RuntimeError("Unexpected delivery channel")
        check_challenge(challenge, "phone")
        code, _ = inbox_code(channel + "." + phone[1:] + "@phone.invalid", r"Verification code: (\d{6})", challenge["expires_in"])
        result = post(opener, "phone/verify", {"challenge_id": challenge["challenge_id"], "code": code}, 200)
        check_proof(challenge, result, "phone")
        print("PASS " + channel + " development delivery and verification", flush=True)

    if os.environ.get("BDS_PROBE_PG_CONTAINER"):
        print("PASS stored OTP/cooldown/proof deadlines match API; email resend reissued", flush=True)
    print("PASS delivery runtime: email and three phone channels; no live provider calls", flush=True)


if __name__ == "__main__":
    main()
