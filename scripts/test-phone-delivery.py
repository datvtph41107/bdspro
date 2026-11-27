#!/usr/bin/env python3
"""Local-only smoke test: real API/DB/Redis + Mailpit; no paid provider requests."""
from contextlib import contextmanager
import shutil
import http.cookiejar
import json
import os
from pathlib import Path
import re
import secrets
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
MAILPIT = "http://127.0.0.1:8025"
ORIGIN = "http://localhost:5173"


def read_json(url):
    with urllib.request.urlopen(url, timeout=5) as response:
        return json.load(response)


def inbox_code(channel, phone):
    recipient = channel + "." + phone[1:] + "@phone.invalid"
    for _ in range(15):
        messages = read_json(MAILPIT + "/api/v1/messages").get("messages", [])
        matches = [message for message in messages
                   if any(to.get("Address") == recipient for to in message.get("To", []))]
        if matches:
            message = read_json(MAILPIT + "/api/v1/message/" + matches[0]["ID"])
            match = re.search(r"Verification code: (\d{6})", message.get("Text", ""))
            if match:
                return match.group(1)
        time.sleep(0.2)
    raise AssertionError("No verification message in Mailpit for " + channel)


@contextmanager
def isolated_services():
    """Own all test DB/Redis state; never flush or migrate the developer's services."""
    containers = []
    try:
        ports = {}
        for name, image, port, options in [
            ("postgres", "postgres:18-alpine", 5432,
             ["-e", "POSTGRES_PASSWORD=test-only", "-e", "POSTGRES_DB=bdspro_test"]),
            ("redis", "redis:7-alpine", 6379, []),
        ]:
            result = subprocess.run(
                ["docker", "run", "--pull=never", "--rm", "-d",
                 "-p", "127.0.0.1::" + str(port), *options, image],
                check=True, text=True, capture_output=True)
            container = result.stdout.strip()
            containers.append(container)
            info = json.loads(subprocess.check_output(["docker", "inspect", container], text=True))[0]
            ports[name] = info["NetworkSettings"]["Ports"][str(port) + "/tcp"][0]["HostPort"]
        for _ in range(100):
            ready = subprocess.run(["docker", "exec", containers[0], "pg_isready", "-U", "postgres"],
                                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            if ready.returncode == 0:
                break
            time.sleep(0.2)
        else:
            raise AssertionError("Disposable PostgreSQL did not become ready")
        yield {
            "database": "postgres://postgres:test-only@127.0.0.1:" + ports["postgres"] + "/bdspro_test?sslmode=disable",
            "redis": "127.0.0.1:" + ports["redis"],
        }
    finally:
        for container in reversed(containers):
            subprocess.run(["docker", "stop", "-t", "2", container],
                           stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=True)

@contextmanager
def test_artifacts():
    work = tempfile.mkdtemp(prefix="bdspro-phone-test-")
    try:
        yield work
    except BaseException:
        print("Failure artifacts retained at", work, flush=True)
        raise
    else:
        shutil.rmtree(work)


def main():
    read_json(MAILPIT + "/api/v1/info")
    with isolated_services() as services:
        run(services)


def run(services):
    with socket.socket() as port_socket:
        port_socket.bind(("127.0.0.1", 0))
        port = port_socket.getsockname()[1]
    base = "http://127.0.0.1:" + str(port)
    # Use the local app.ini, and force all transports to Mailpit regardless of shell settings.
    env = {key: value for key, value in os.environ.items() if not key.startswith("BDS__")}
    env.update({
        "BDS____RUN_MODE": "development",
        "BDS__DATABASE__DB_SOURCE": services["database"],
        "BDS__DATABASE__LOG_SQL": "all",
        "BDS__REDIS__ADDR": services["redis"],
        "BDS__LOG__LEVEL": "debug",
        "BDS__SERVER__HTTP_SERVER_ADDRESS": "127.0.0.1:" + str(port),
        "BDS__SERVER__PUBLIC_ORIGIN": ORIGIN,
        "BDS__PHONE_DELIVERY__MODE": "mailpit",
        "BDS__PHONE_DELIVERY__DEFAULT_CHANNEL": "sms",
        "BDS__PHONE_DELIVERY__CHANNELS": "sms,zalo,telegram",
        "BDS__MAILER__SMTP_PROTOCOL": "smtp",
        "BDS__MAILER__SMTP_ADDRESS": "127.0.0.1:1025",
        "BDS__MAILER__FROM": "security@bdspro.test",
        "BDS__MAILER__USERNAME": "",
        "BDS__MAILER__PASSWORD": "",
        "BDS__LOG__MODE": "file",
        "BDS__LOG__FORMAT": "json",
    })
    with test_artifacts() as work:
        binary = str(Path(work) / "bdspro")
        env["BDS__LOG__ROOT_PATH"] = work
        test_env = dict(env, BDSPRO_TEST_DATABASE_URL=services["database"])
        subprocess.run(["go", "test", "./models/db", "-run", "^TestAuthStateMigration$", "-count=1", "-v"],
                       cwd=ROOT, env=test_env, check=True)
        subprocess.run(["go", "build", "-o", binary, "."], cwd=ROOT, check=True)
        subprocess.run([binary, "--config", "custom/conf/app.ini", "config", "check"],
                       cwd=ROOT, env=env, check=True)
        with open(Path(work) / "startup.log", "w+") as startup:
            process = subprocess.Popen([binary, "--config", "custom/conf/app.ini", "web"],
                                       cwd=ROOT, env=env, stdout=startup, stderr=startup)
            try:
                for _ in range(100):
                    if process.poll() is not None:
                        raise AssertionError("Test backend exited; inspect startup.log in retained failure artifacts")
                    try:
                        read_json(base + "/api/v1/health")
                        break
                    except (urllib.error.URLError, ConnectionError):
                        time.sleep(0.1)
                else:
                    raise AssertionError("Test backend did not start")
                capabilities = read_json(base + "/api/v1/auth/phone/channels")
                assert capabilities["channels"] == ["sms", "zalo", "telegram"]
                assert capabilities["development"] is True
                print("Temporary backend ready on port", port, flush=True)
                pending = None
                codes = []
                for channel in capabilities["channels"]:
                    opener = urllib.request.build_opener(
                        urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

                    def post(path, body, expected, request_opener=opener):
                        request = urllib.request.Request(base + "/api/v1" + path,
                            data=json.dumps(body).encode(), method="POST",
                            headers={"Content-Type": "application/json", "Origin": ORIGIN,
                                     "X-BDS-Request": "1", "Sec-Fetch-Site": "same-origin"})
                        try:
                            response = request_opener.open(request, timeout=15)
                        except urllib.error.HTTPError as error:
                            response = error
                        with response:
                            data = json.load(response)
                            assert response.status == expected, (
                                path, response.status, data.get("code"), "request_id="+str(response.headers.get("X-Request-ID")), "retry_after="+str(response.headers.get("Retry-After")))
                            assert response.headers.get("X-Request-ID")
                            return data

                    phone = "+8490" + "".join(secrets.choice("0123456789") for _ in range(7))
                    started = time.monotonic()
                    challenge = post("/auth/phone/start",
                                     {"phone": phone, "channel": channel}, 202)
                    assert challenge["delivery_channel"] == channel
                    assert "code" not in challenge and "dev_code" not in challenge
                    code = inbox_code(channel, phone)
                    outsider = urllib.request.build_opener()
                    post("/auth/phone/verify", {"challenge_id":challenge["challenge_id"], "code":code}, 400, outsider)
                    codes.append(code)
                    wrong = "000000" if code != "000000" else "111111"
                    post("/auth/phone/verify", {"challenge_id": challenge["challenge_id"], "code": wrong}, 400)
                    if channel == "sms":
                        post("/auth/phone/resend",
                             {"challenge_id": challenge["challenge_id"], "channel": "zalo"}, 429)
                        pending = (post, phone, challenge, started, code)
                    else:
                        result = post("/auth/phone/verify",
                                      {"challenge_id": challenge["challenge_id"], "code": code}, 200)
                        assert result["status"] == "registration_required"
                        post("/auth/phone/verify",
                             {"challenge_id": challenge["challenge_id"], "code": code}, 400)
                    print(channel + ": Mailpit received OTP; invalid-code protection passed", flush=True)

                post, phone, challenge, started, old_code = pending
                wait = max(0, 61 - (time.monotonic() - started))
                print("Checking shared resend cooldown and SMS -> Zalo switch; waiting",
                      round(wait), "seconds", flush=True)
                time.sleep(wait)
                resent = post("/auth/phone/resend",
                              {"challenge_id": challenge["challenge_id"], "channel": "zalo"}, 202)
                assert resent["delivery_channel"] == "zalo"
                code = inbox_code("zalo", phone)
                codes.append(code)
                if old_code != code:
                    post("/auth/phone/verify",
                         {"challenge_id": challenge["challenge_id"], "code": old_code}, 400)
                result = post("/auth/phone/verify",
                              {"challenge_id": challenge["challenge_id"], "code": code}, 200)
                assert result["status"] == "registration_required"
                post("/auth/phone/verify",
                     {"challenge_id": challenge["challenge_id"], "code": code}, 400)

                events = [json.loads(line) for line in (Path(work) / "bdspro.log").read_text().splitlines()]
                deliveries = [event for event in events if event.get("message") == "phone.delivery"]
                assert len(deliveries) == 4 and all(event["accepted"] for event in deliveries)
                denials = [event for event in events if event.get("message") == "admission.denied"]
                assert any(event.get("action") == "sms_destination_cooldown" and event.get("scope") == "subject" for event in denials)
                queries = [event for event in events if event.get("message") == "db.query"]
                assert any(event.get("query") == "CreatePhoneVerification" and event.get("request_id") for event in queries)
                for event in events:
                    for key in ("code", "otp", "phone", "authorization", "access_token", "sql", "args"):
                        # API error "code" is public metadata, never the six-digit OTP.
                        assert key not in event or (key == "code" and event[key] not in codes)
                print("PASS: all channels, browser binding, wrong/replayed/rotated codes, shared cooldown, correlated SQL and safe logs.", flush=True)
                print("Disposable DB/Redis tested; four Mailpit messages retained. No provider requests.", flush=True)
            finally:
                process.terminate()
                try:
                    process.wait(timeout=15)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()
                print("Temporary backend stopped; existing server ports untouched.", flush=True)


if __name__ == "__main__":
    main()
