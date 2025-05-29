from locust import HttpUser, task, between
import time
import random
import string
import os
import requests
from client_http import perform_handshake, perform_finalize, derive_keys, perform_session_test
# locust -f locust_test.py --host=http://localhost:8080
class BaseUser(HttpUser):
    wait_time = between(1, 2)
    crypto_host = "http://localhost:8080"
    auth_host = "http://localhost:8081"

    def on_start(self):
        self.email = self.generate_random_email()
        self.password = self.generate_random_password()
        self.platform = "web"
        self.access_token = self.register_user()
        self.handshake_sequence()

    def generate_random_email(self):
        name = ''.join(random.choices(string.ascii_lowercase + string.digits, k=random.randint(5, 10)))
        return f"{name}@gmail.com"

    def generate_random_password(self):
        return ''.join(random.choices(string.ascii_letters + string.digits, k=random.randint(7, 13)))

    def register_user(self):
        payload = {
            "email": self.email,
            "password": self.password,
            "platform": self.platform
        }
        try:
            response = requests.post(f"{self.auth_host}/user/signup", json=payload)
            if response.status_code == 200:
                return response.json().get("access_token")
            else:
                print(f"Ошибка регистрации: {response.status_code} - {response.text}")
        except Exception as e:
            print(f"Ошибка при регистрации: {e}")
        return None

    def handshake_sequence(self):
        try:
            # handshake/init
            start = time.time()
            self.handshake_data = perform_handshake(f"{self.crypto_host}/handshake/init", access_token=self.access_token)
            duration = (time.time() - start) * 1000
            self.client_id = self.handshake_data["client_id"]
            self.environment.events.request.fire(
                request_type="POST",
                name="/handshake/init",
                response_time=duration,
                response_length=len(self.client_id),
                response=None,
                context={},
                exception=None,
                start_time=start,
                url=f"{self.crypto_host}/handshake/init"
            )

            # handshake/finalize
            start = time.time()
            self.ks = perform_finalize(f"{self.crypto_host}/handshake/finalize", self.handshake_data, access_token=self.access_token)
            duration = (time.time() - start) * 1000
            self.environment.events.request.fire(
                request_type="POST",
                name="/handshake/finalize",
                response_time=duration,
                response_length=len(self.ks),
                response=None,
                context={},
                exception=None,
                start_time=start,
                url=f"{self.crypto_host}/handshake/finalize"
            )

            self.k_enc, self.k_mac = derive_keys(self.ks)
        except Exception as e:
            print(f"Ошибка handshake: {e}")
            self.ks = None

    @task
    def test_session_cycle(self):
        if not self.ks:
            return
        try:
            session = {
                "client_id": self.client_id,
                "k_enc": self.k_enc,
                "k_mac": self.k_mac,
                "ecdsa_priv": self.handshake_data["ecdsa_priv"]
            }
            data = os.urandom(random.randint(1024 * 1024, 2 * 1024 * 1024))  # 0.5MB - 2MB
            start = time.time()
            perform_session_test(f"{self.crypto_host}/session/test", session, self.access_token, plaintext=data)
            duration = (time.time() - start) * 1000
            self.environment.events.request.fire(
                request_type="POST",
                name="/session/test",
                response_time=duration,
                response_length=len(data),
                response=None,
                context={},
                exception=None,
                start_time=start,
                url=f"{self.crypto_host}/session/test"
            )
            # Новый цикл
            self.handshake_sequence()
        except Exception as e:
            print(f"Ошибка session/test: {e}")
            self.environment.events.request.fire(
                request_type="POST",
                name="/session/test",
                response_time=0,
                response_length=0,
                response=None,
                context={},
                exception=e,
                start_time=time.time(),
                url=f"{self.crypto_host}/session/test"
            )


class MobileUser(BaseUser):
    def on_start(self):
        self.platform = "tg-bot"
        self.client.headers["User-Agent"] = "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X)"
        super().on_start()


class PCUser(BaseUser):
    def on_start(self):
        self.platform = "web"
        self.client.headers["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
        super().on_start()
