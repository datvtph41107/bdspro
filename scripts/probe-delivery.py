#!/usr/bin/env python3
"""Runtime-only probe against a disposable local BDSPro and Mailpit."""
import http.cookiejar
import json
import re
import secrets
import time
import urllib.error
import urllib.request

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


def inbox_code(recipient, pattern):
    for _ in range(30):
        messages = get(MAILPIT + "/api/v1/messages").get("messages", [])
        for message in messages:
            if any(to.get("Address") == recipient for to in message.get("To", [])):
                content = get(MAILPIT + "/api/v1/message/" + message["ID"])
                match = re.search(pattern, content.get("Text", ""))
                if match:
                    return match.group(1)
        time.sleep(0.2)
    raise RuntimeError("Expected verification message missing from development inbox")


def main():
    capabilities = get(BASE + "/api/v1/auth/phone/channels")
    if capabilities.get("development") is not True or capabilities.get("channels") != ["sms", "zalo", "telegram"]:
        raise RuntimeError("Refusing delivery probe outside expected Mailpit development mode")
    get(MAILPIT + "/api/v1/info")

    email = "probe-" + secrets.token_hex(6) + "@example.invalid"
    opener = browser()
    challenge = post(opener, "register/start", {"email": email}, 202)
    code = inbox_code(email, r"Enter this verification code in BDSPro: (\d{8})")
    result = post(opener, "register/verify", {"challenge_id": challenge["challenge_id"], "code": code}, 200)
    if result.get("status") != "registration_required":
        raise RuntimeError("Expected email registration proof")
    print("PASS email registration SMTP delivery and verification", flush=True)

    for channel in capabilities["channels"]:
        phone = "+8490" + "".join(secrets.choice("0123456789") for _ in range(7))
        opener = browser()
        challenge = post(opener, "phone/start", {"phone": phone, "channel": channel}, 202)
        if challenge.get("delivery_channel") != channel:
            raise RuntimeError("Unexpected delivery channel")
        code = inbox_code(channel + "." + phone[1:] + "@phone.invalid", r"Verification code: (\d{6})")
        result = post(opener, "phone/verify", {"challenge_id": challenge["challenge_id"], "code": code}, 200)
        if result.get("status") != "registration_required":
            raise RuntimeError("Expected phone registration proof")
        print("PASS " + channel + " development delivery and verification", flush=True)

    print("PASS delivery runtime: email and three phone channels; no live provider calls", flush=True)


if __name__ == "__main__":
    main()
