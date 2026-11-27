#!/usr/bin/env python3
"""Runtime-only probe against a disposable local BDSPro and Mailpit."""
import base64
import datetime
import hashlib
import http.cookiejar
from http.cookies import SimpleCookie
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


def post(opener, path, body, expected, *, access=None, group="auth", response_check=None):
    headers = {"Content-Type": "application/json", "Origin": BASE,
               "X-BDS-Request": "1", "Sec-Fetch-Site": "same-origin"}
    if access:
        headers["Authorization"] = "Bearer " + access
    request = urllib.request.Request(
        BASE + "/api/v1/" + group + "/" + path,
        data=json.dumps(body).encode(), method="POST",
        headers=headers,
    )
    try:
        response = opener.open(request, timeout=15)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        if response.status != expected:
            raise RuntimeError(f"{path}: HTTP {response.status}, expected {expected}")
        if response_check is not None:
            response_check(response)
        data = None if response.status == 204 else json.load(response)
        print(f"PASS POST /api/v1/{group}/{path} {response.status}", flush=True)
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


def check_failed_attempts(opener, kind, path, challenge, code):
    challenge_id = str(uuid.UUID(challenge["challenge_id"]))
    # Always distinct from the real code, including when the code begins with zero.
    wrong = ("1" if code[0] == "0" else "0") + code[1:]
    for expected in range(1, 6):
        result = post(opener, path, {"challenge_id": challenge_id, "code": wrong}, 400)
        if result.get("code") != "invalid_verification_code":
            raise RuntimeError("Unexpected failed verification error")
        row = db_query(f"SELECT failed_attempts FROM verifications WHERE kind='{kind}' AND id='{challenge_id}'::uuid")
        if row != str(expected):
            raise RuntimeError("Incorrect persisted attempt count")
    result = post(opener, path, {"challenge_id": challenge_id, "code": code}, 400)
    if result.get("code") != "invalid_verification_code":
        raise RuntimeError("Attempt cap did not reject correct code")
    row = db_query(f"SELECT failed_attempts, proof_digest IS NULL FROM verifications WHERE kind='{kind}' AND id='{challenge_id}'::uuid")
    if row != "5|t":
        raise RuntimeError("Exhausted challenge changed state")
    return challenge_id


def check_browser_origin():
    # No refresh cookie: accepted browser requests must reach the 401 handler,
    # while rejected origins must stop at the 403 guard. No real account needed.
    common = {"Content-Type": "application/json", "Origin": BASE,
              "X-BDS-Request": "1", "Sec-Fetch-Site": "same-origin"}
    cases = [
        ("same origin", {}, 401),
        ("no fetch metadata", {"Sec-Fetch-Site": None}, 401),
        ("navigation metadata", {"Sec-Fetch-Site": "none"}, 401),
        ("missing origin", {"Origin": None}, 403),
        ("null origin", {"Origin": "null"}, 403),
        ("foreign origin", {"Origin": "https://foreign.example.invalid"}, 403),
        ("origin with path", {"Origin": BASE + "/path"}, 403),
        ("origin with query", {"Origin": BASE + "?x=1"}, 403),
        ("missing custom header", {"X-BDS-Request": None}, 403),
        ("incorrect custom header", {"X-BDS-Request": "0"}, 403),
        ("cross-site metadata", {"Sec-Fetch-Site": "cross-site"}, 403),
        ("same-site metadata", {"Sec-Fetch-Site": "same-site"}, 403),
    ]
    for name, overrides, expected in cases:
        headers = {**common, **overrides}
        request = urllib.request.Request(
            BASE + "/api/v1/auth/refresh", data=b"{}", method="POST",
            headers={key: value for key, value in headers.items() if value is not None},
        )
        try:
            response = browser().open(request, timeout=5)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            if response.status != expected:
                raise RuntimeError(f"Browser guard {name}: HTTP {response.status}, expected {expected}")
            if response.headers.get("Set-Cookie") is not None:
                raise RuntimeError(f"Browser guard {name}: unexpectedly wrote a cookie")
    print("PASS browser Origin, custom header and fetch metadata admission", flush=True)


def check_guard_boundaries():
    email = "guard-" + secrets.token_hex(6) + "@example.invalid"
    opener = browser()
    challenge = post(opener, "register/start", {"email": email}, 202)
    code, _ = inbox_code(email, r"Enter this verification code in BDSPro: (\d{8})", challenge["expires_in"])
    locked_id = check_failed_attempts(opener, "email", "register/verify", challenge, code)
    db_query(f"UPDATE verifications SET resend_after=clock_timestamp()-interval '1 second' WHERE kind='email' AND id='{locked_id}'::uuid")
    post(opener, "register/resend", {"challenge_id": locked_id}, 429)
    if db_query(f"SELECT failed_attempts FROM verifications WHERE kind='email' AND id='{locked_id}'::uuid") != "5":
        raise RuntimeError("Resend reset exhausted challenge")

    # The schema is a hard safety ceiling, not an independently adjustable runtime policy.
    try:
        db_query(f"UPDATE verifications SET failed_attempts=6 WHERE kind='email' AND id='{locked_id}'::uuid")
    except subprocess.CalledProcessError as error:
        if "check constraint" not in error.stderr:
            raise RuntimeError("Unexpected database rejection") from None
    else:
        raise RuntimeError("Database allowed an attempt count above its ceiling")

    email = "horizon-" + secrets.token_hex(6) + "@example.invalid"
    opener = browser()
    challenge = post(opener, "register/start", {"email": email}, 202)
    horizon_id = str(uuid.UUID(challenge["challenge_id"]))
    db_query(f"UPDATE verifications SET created_at=clock_timestamp()-interval '31 minutes', "
             f"resend_after=clock_timestamp()-interval '1 second' WHERE kind='email' AND id='{horizon_id}'::uuid")
    post(opener, "register/resend", {"challenge_id": horizon_id}, 429)
    code, _ = inbox_code(email, r"Enter this verification code in BDSPro: (\d{8})", challenge["expires_in"])
    # Resend horizon is distinct from validity: verify after aging only the creation time.
    result = post(opener, "register/verify", {"challenge_id": horizon_id, "code": code}, 200)
    check_proof(challenge, result, "email")

    db_query(f"UPDATE verifications SET expires_at=clock_timestamp()-interval '31 minutes' WHERE kind='email' AND id='{locked_id}'::uuid")
    db_query(f"UPDATE verifications SET expires_at=clock_timestamp()-interval '29 minutes' WHERE kind='email' AND id='{horizon_id}'::uuid")
    post(browser(), "register/start", {"email": "cleanup-" + secrets.token_hex(6) + "@example.invalid"}, 202)
    row = db_query(f"SELECT count(*) FILTER (WHERE id='{locked_id}'::uuid), count(*) FILTER (WHERE id='{horizon_id}'::uuid) "
                   f"FROM verifications WHERE kind='email' AND id IN ('{locked_id}'::uuid,'{horizon_id}'::uuid)")
    if row != "0|1":
        raise RuntimeError("Retention cleanup boundary changed")

    phone = "+8490" + "".join(secrets.choice("0123456789") for _ in range(7))
    opener = browser()
    challenge = post(opener, "phone/start", {"phone": phone, "channel": "sms"}, 202)
    # A new challenge for the same destination must still pass Redis admission.
    post(browser(), "phone/start", {"phone": phone, "channel": "sms"}, 429)
    post(opener, "phone/resend", {"challenge_id": challenge["challenge_id"], "channel": "sms"}, 429)
    code, _ = inbox_code("sms." + phone[1:] + "@phone.invalid", r"Verification code: (\d{6})", challenge["expires_in"])
    check_failed_attempts(opener, "phone", "phone/verify", challenge, code)
    print("PASS attempt caps, schema ceiling, resend horizon, retention and destination admission", flush=True)


def check_session_deadline(opener, account):
    # Decode only tokens issued by this disposable test server; never log credentials.
    payload = account["access_token"].split(".")[1]
    claims = json.loads(base64.urlsafe_b64decode(payload + "=" * (-len(payload) % 4)))
    session_id = str(uuid.UUID(claims["session_id"]))
    persisted = db_query(f"SELECT expires_at FROM sessions WHERE id='{session_id}'::uuid")
    deadline = datetime.datetime.fromisoformat(persisted)
    issued = datetime.datetime.fromisoformat(claims["issued_at"])
    expires = datetime.datetime.fromisoformat(claims["expired_at"])
    if expires > deadline or abs(account["expires_in"] - int((expires - issued).total_seconds())) > 1:
        raise RuntimeError("Issued JWT or response exceeds canonical session lifetime")

    # Shorten only this disposable session so its deadline, not access TTL, is limiting.
    db_query(f"UPDATE sessions SET expires_at=clock_timestamp()+interval '30 seconds' WHERE id='{session_id}'::uuid")
    deadline = datetime.datetime.fromisoformat(db_query(f"SELECT expires_at FROM sessions WHERE id='{session_id}'::uuid"))
    refreshed = post(opener, "refresh", {}, 200)
    payload = refreshed["access_token"].split(".")[1]
    claims = json.loads(base64.urlsafe_b64decode(payload + "=" * (-len(payload) % 4)))
    if claims["session_id"] != session_id or datetime.datetime.fromisoformat(claims["expired_at"]) != deadline:
        raise RuntimeError("Refresh JWT did not use persisted session deadline")
    if not 0 < refreshed["expires_in"] <= 30:
        raise RuntimeError("Refresh lifetime exceeded shortened session")
    if datetime.datetime.fromisoformat(db_query(f"SELECT expires_at FROM sessions WHERE id='{session_id}'::uuid")) != deadline:
        raise RuntimeError("Refresh extended the session")
    db_query(f"UPDATE sessions SET expires_at=clock_timestamp()-interval '1 second' WHERE id='{session_id}'::uuid")
    post(opener, "refresh", {}, 401)
    print("PASS persisted session deadline, non-sliding refresh and expired session denial", flush=True)


def check_recovery_and_reauth(opener, email, registration):
    password = "disposable-probe-password-123"
    account = post(opener, "register/complete", {
        "registration_token": registration["registration_token"],
        "username": "probe_" + secrets.token_hex(6), "password": password,
        "terms_version": "v1", "marketing_opt_in": False, "stay_signed_in": False,
    }, 201)
    access = account["access_token"]
    reauth = post(opener, "reauth/password", {"password": password, "action": "sessions.revoke_all"},
                  200, access=access, group="me")
    if reauth.get("expires_in") != 600:
        raise RuntimeError("Unexpected recent authentication lifetime")
    digest = hashlib.sha256(reauth["reauthentication_proof"].encode()).hexdigest()
    row = db_query(f"SELECT extract(epoch FROM expires_at-clock_timestamp()) FROM auth_proofs "
                   f"WHERE digest=decode('{digest}','hex') AND method='password' AND action='sessions.revoke_all'")
    if not 585 <= float(row) <= 600:
        raise RuntimeError("Persisted reauthentication deadline disagrees with API")
    db_query(f"UPDATE auth_proofs SET expires_at=clock_timestamp()-interval '1 second' WHERE digest=decode('{digest}','hex')")
    post(opener, "sessions/revoke-all", {"reauthentication_proof": reauth["reauthentication_proof"]},
         403, access=access, group="me")

    # Unknown and known email both retain the accepted response and minimum timing.
    for destination in (email, "missing-" + secrets.token_hex(6) + "@example.invalid"):
        started = time.monotonic()
        result = post(opener, "password/reset/request", {"email": destination}, 202)
        if result != {"status": "accepted"} or time.monotonic() - started < 3.8:
            raise RuntimeError("Recovery response or minimum timing changed")
    proof, _ = inbox_code(email, r"Enter this recovery token in BDSPro: ([A-Za-z0-9_-]{43})", 900)
    digest = hashlib.sha256(proof.encode()).hexdigest()
    row = db_query(f"SELECT extract(epoch FROM expires_at-created_at) FROM password_reset_tokens "
                   f"WHERE token_hash=decode('{digest}','hex')")
    if float(row) != 900:
        raise RuntimeError("Password recovery TTL disagrees with delivered proof")
    body = {"reset_token": proof, "new_password": "disposable-new-password-456"}
    post(opener, "password/reset/confirm", body, 204)
    post(opener, "password/reset/confirm", body, 400)
    if db_query(f"SELECT count(*) FROM password_reset_tokens WHERE token_hash=decode('{digest}','hex')") != "0":
        raise RuntimeError("Consumed password recovery proof still exists")
    post(opener, "login", {"identifier": email, "password": password}, 401)
    account = post(opener, "login", {"identifier": email, "password": body["new_password"]}, 200)
    check_session_deadline(opener, account)
    print("PASS recovery and recent-auth TTL, expired proof rejection, reset/replay and new password", flush=True)


def check_google_registration_binding():
    # Disposable fixture represents a verified provider result. No Google request is sent.
    raw = secrets.token_urlsafe(32)
    digest = hashlib.sha256(raw.encode()).hexdigest()
    subject = "fixture-" + secrets.token_hex(16)
    db_query(f"INSERT INTO google_flows(browser_digest, phase, subject, remember, expires_at) "
             f"VALUES (decode('{digest}','hex'), 'ready', '{subject}', false, clock_timestamp()+interval '1 minute')")
    opener = browser()
    post(opener, "google/result", {}, 400)
    request = urllib.request.Request(
        BASE + "/api/v1/auth/google/result", data=b"{}", method="POST",
        headers={"Content-Type": "application/json", "Origin": BASE,
                 "X-BDS-Request": "1", "Sec-Fetch-Site": "same-origin",
                 "Cookie": "bdspro_google_dev=" + raw},
    )
    with opener.open(request, timeout=15) as response:
        result = json.load(response)
        cookies = SimpleCookie()
        for header in response.headers.get_all("Set-Cookie", []):
            cookies.load(header)
    renewed = cookies.get("bdspro_google_dev")
    if result.get("status") != "registration_required" or result.get("expires_in") != 900:
        raise RuntimeError("Google registration proof lifetime changed")
    if renewed is None or renewed.value != raw or renewed["max-age"] != str(result["expires_in"]):
        raise RuntimeError("Google browser binding was not renewed for the proof lifetime")
    if not renewed["httponly"] or renewed["samesite"].lower() != "lax" or renewed["path"] != "/":
        raise RuntimeError("Google browser cookie attributes changed")
    row = db_query(f"SELECT phase, extract(epoch FROM expires_at-clock_timestamp()) "
                   f"FROM google_flows WHERE browser_digest=decode('{digest}','hex')")
    phase, remaining = row.split("|")
    if phase != "registration" or not 885 <= float(remaining) <= 900:
        raise RuntimeError("Google persisted registration deadline disagrees with API")
    # No second proof can be issued from the same ready result.
    post(opener, "google/result", {}, 400)
    db_query(f"DELETE FROM google_flows WHERE browser_digest=decode('{digest}','hex')")
    print("PASS Google registration cookie renewal, browser binding and result replay denial (fixture)", flush=True)


def check_registration_cookie(response):
    cookies = SimpleCookie()
    for header in response.headers.get_all("Set-Cookie", []):
        cookies.load(header)
    cookie = cookies.get("bdspro_register_dev")
    if cookie is None or cookie["max-age"] != "3000":
        raise RuntimeError("Registration browser lifetime does not cover resend/OTP/proof")
    if not cookie["httponly"] or cookie["samesite"].lower() != "lax" or cookie["path"] != "/":
        raise RuntimeError("Registration browser cookie attributes changed")


def main():
    capabilities = get(BASE + "/api/v1/auth/phone/channels")
    if capabilities.get("development") is not True or capabilities.get("channels") != ["sms", "zalo", "telegram"]:
        raise RuntimeError("Refusing delivery probe outside expected Mailpit development mode")
    get(MAILPIT + "/api/v1/info")
    check_browser_origin()

    email = "probe-" + secrets.token_hex(6) + "@example.invalid"
    opener = browser()
    challenge = post(opener, "register/start", {"email": email}, 202, response_check=check_registration_cookie)
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
    if os.environ.get("BDS_PROBE_PG_CONTAINER"):
        check_recovery_and_reauth(opener, email, result)

    for channel in capabilities["channels"]:
        phone = "+8490" + "".join(secrets.choice("0123456789") for _ in range(7))
        opener = browser()
        challenge = post(opener, "phone/start", {"phone": phone, "channel": channel}, 202, response_check=check_registration_cookie)
        if challenge.get("delivery_channel") != channel:
            raise RuntimeError("Unexpected delivery channel")
        check_challenge(challenge, "phone")
        code, _ = inbox_code(channel + "." + phone[1:] + "@phone.invalid", r"Verification code: (\d{6})", challenge["expires_in"])
        result = post(opener, "phone/verify", {"challenge_id": challenge["challenge_id"], "code": code}, 200)
        check_proof(challenge, result, "phone")
        print("PASS " + channel + " development delivery and verification", flush=True)

    if os.environ.get("BDS_PROBE_PG_CONTAINER"):
        print("PASS stored OTP/cooldown/proof deadlines match API; email resend reissued", flush=True)
        check_guard_boundaries()
        check_google_registration_binding()
    print("PASS delivery runtime: email and three phone channels; no live provider calls", flush=True)


if __name__ == "__main__":
    main()
