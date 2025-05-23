from locust import HttpUser, task, between
from client_http import perform_handshake, perform_finalize, derive_keys, perform_session_test
import time
import os
import random
import uuid

class HandshakeUser(HttpUser):
    wait_time = between(3, 5)
    host = "http://localhost:8080"
    # host = "http://localhost:8081"  # Прокси

    def on_start(self):
        self.client.headers.clear()

    @task
    def test_handshake(self):
        try:
            time.sleep(random.uniform(3.0, 5.0))
            start = time.time()
            nonce = os.urandom(8)
            request_id = str(uuid.uuid4())
            handshake_data = perform_handshake(f"{self.host}/handshake/init", nonce1=nonce)
            self.client_id = handshake_data["client_id"]
            self.client.headers["X-Client-ID"] = self.client_id

            response_time = (time.time() - start) * 1000
            self.environment.events.request.fire(
                request_type="POST",
                name="/handshake/init",
                response_time=response_time,
                response_length=len(handshake_data["client_id"]),
                response=None,
                context={},
                exception=None,
                start_time=start,
                url=f"{self.host}/handshake/init"
            )
            self.handshake_data = handshake_data
        except Exception as e:
            print(f"Ошибка test_handshake (HandshakeUser): {str(e)}")
            self.environment.events.request.fire(
                request_type="POST",
                name="/handshake/init",
                response_time=0,
                response_length=0,
                response=None,
                context={},
                exception=e,
                start_time=time.time(),
                url=f"{self.host}/handshake/init"
            )

    @task
    def test_finalize(self):
        if not hasattr(self, 'handshake_data') or not hasattr(self, 'client_id'):
            return
        try:
            start = time.time()
            ks = perform_finalize(f"{self.host}/handshake/finalize", self.handshake_data)
            response_time = (time.time() - start) * 1000
            print(f"Время финального рукопожатия (HandshakeUser): {response_time:.2f} мс")
            self.environment.events.request.fire(
                request_type="POST",
                name="/handshake/finalize",
                response_time=response_time,
                response_length=len(ks),
                response=None,
                context={},
                exception=None,
                start_time=start,
                url=f"{self.host}/handshake/finalize"
            )
            nonce = os.urandom(8)
            self.handshake_data = perform_handshake(f"{self.host}/handshake/init", nonce1=nonce)
            self.client_id = self.handshake_data["client_id"]
            self.client.headers["X-Client-ID"] = self.client_id
        except Exception as e:
            print(f"Ошибка test_finalize (HandshakeUser): {str(e)}")
            self.environment.events.request.fire(
                request_type="POST",
                name="/handshake/finalize",
                response_time=0,
                response_length=0,
                response=None,
                context={},
                exception=e,
                start_time=time.time(),
                url=f"{self.host}/handshake/finalize"
            )

    @task
    def test_session(self):
        if not hasattr(self, 'handshake_data') or not hasattr(self, 'client_id'):
            return
        try:
            start = time.time()
            ks = perform_finalize(f"{self.host}/handshake/finalize", self.handshake_data)
            K_enc, K_mac = derive_keys(ks)
            session = {
                "client_id": self.client_id,
                "k_enc": K_enc,
                "k_mac": K_mac,
                "ecdsa_priv": self.handshake_data["ecdsa_priv"]
            }
            large_data_10mb = os.urandom(10 * 1024 * 1024)
            perform_session_test(f"{self.host}/session/test", session, large_data_10mb)
            response_time = (time.time() - start) * 1000
            self.environment.events.request.fire(
                request_type="POST",
                name="/session/test",
                response_time=response_time,
                response_length=len(large_data_10mb),
                response=None,
                context={},
                exception=None,
                start_time=start,
                url=f"{self.host}/session/test"
            )
            # Перезапускаем рукопожатие для следующей итерации
            nonce = os.urandom(8)
            self.handshake_data = perform_handshake(f"{self.host}/handshake/init", nonce1=nonce)
            self.client_id = self.handshake_data["client_id"]
            self.client.headers["X-Client-ID"] = self.client_id
        except Exception as e:
            print(f"Ошибка test_session (HandshakeUser): {str(e)}")
            self.environment.events.request.fire(
                request_type="POST",
                name="/session/test",
                response_time=0,
                response_length=0,
                response=None,
                context={},
                exception=e,
                start_time=time.time(),
                url=f"{self.host}/session/test"
            )

class MobileUser(HttpUser):
    wait_time = between(3, 5)
    host = "http://localhost:8080"
    # host = "http://localhost:8081"  # Прокси

    def on_start(self):
        self.client.headers["User-Agent"] = "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.0 Mobile/15E148 Safari/604.1"
        nonce = os.urandom(8)
        self.handshake_data = perform_handshake(f"{self.host}/handshake/init", nonce1=nonce)
        self.client_id = self.handshake_data["client_id"]
        self.client.headers["X-Client-ID"] = self.client_id

    @task
    def test_finalize(self):
        if not hasattr(self, 'handshake_data') or not hasattr(self, 'client_id'):
            return
        try:
            start = time.time()
            ks = perform_finalize(f"{self.host}/handshake/finalize", self.handshake_data)
            response_time = (time.time() - start) * 1000
            print(f"Время финального рукопожатия (MobileUser): {response_time:.2f} мс")
            self.environment.events.request.fire(
                request_type="POST",
                name="/handshake/finalize_mobile",
                response_time=response_time,
                response_length=len(ks),
                response=None,
                context={},
                exception=None,
                start_time=start,
                url=f"{self.host}/handshake/finalize"
            )
            nonce = os.urandom(8)
            self.handshake_data = perform_handshake(f"{self.host}/handshake/init", nonce1=nonce)
            self.client_id = self.handshake_data["client_id"]
            self.client.headers["X-Client-ID"] = self.client_id
        except Exception as e:
            print(f"Ошибка test_finalize (MobileUser): {str(e)}")
            self.environment.events.request.fire(
                request_type="POST",
                name="/handshake/finalize_mobile",
                response_time=0,
                response_length=0,
                response=None,
                context={},
                exception=e,
                start_time=time.time(),
                url=f"{self.host}/handshake/finalize"
            )

    @task
    def test_session(self):
        if not hasattr(self, 'handshake_data') or not hasattr(self, 'client_id'):
            return
        try:
            start = time.time()
            ks = perform_finalize(f"{self.host}/handshake/finalize", self.handshake_data)
            K_enc, K_mac = derive_keys(ks)
            session = {
                "client_id": self.client_id,
                "k_enc": K_enc,
                "k_mac": K_mac,
                "ecdsa_priv": self.handshake_data["ecdsa_priv"]
            }
            large_data_10mb = os.urandom(10 * 1024 * 1024)
            perform_session_test(f"{self.host}/session/test", session, large_data_10mb)
            response_time = (time.time() - start) * 1000
            self.environment.events.request.fire(
                request_type="POST",
                name="/session/test_mobile",
                response_time=response_time,
                response_length=len(large_data_10mb),
                response=None,
                context={},
                exception=None,
                start_time=start,
                url=f"{self.host}/session/test"
            )
            # Перезапускаем рукопожатие для следующей итерации
            nonce = os.urandom(8)
            self.handshake_data = perform_handshake(f"{self.host}/handshake/init", nonce1=nonce)
            self.client_id = self.handshake_data["client_id"]
            self.client.headers["X-Client-ID"] = self.client_id
        except Exception as e:
            print(f"Ошибка test_session (MobileUser): {str(e)}")
            self.environment.events.request.fire(
                request_type="POST",
                name="/session/test_mobile",
                response_time=0,
                response_length=0,
                response=None,
                context={},
                exception=e,
                start_time=time.time(),
                url=f"{self.host}/session/test"
            )

class WebUser(HttpUser):
    wait_time = between(3, 5)
    host = "http://localhost:8080"
    # host = "http://localhost:8081"  # Прокси

    def on_start(self):
        self.client.headers["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
        nonce = os.urandom(8)
        self.handshake_data = perform_handshake(f"{self.host}/handshake/init", nonce1=nonce)
        self.client_id = self.handshake_data["client_id"]
        self.client.headers["X-Client-ID"] = self.client_id

    @task
    def test_finalize(self):
        if not hasattr(self, 'handshake_data') or not hasattr(self, 'client_id'):
            return
        try:
            start = time.time()
            ks = perform_finalize(f"{self.host}/handshake/finalize", self.handshake_data)
            response_time = (time.time() - start) * 1000
            print(f"Время финального рукопожатия (WebUser): {response_time:.2f} мс")
            self.environment.events.request.fire(
                request_type="POST",
                name="/handshake/finalize_web",
                response_time=response_time,
                response_length=len(ks),
                response=None,
                context={},
                exception=None,
                start_time=start,
                url=f"{self.host}/handshake/finalize"
            )
            nonce = os.urandom(8)
            self.handshake_data = perform_handshake(f"{self.host}/handshake/init", nonce1=nonce)
            self.client_id = self.handshake_data["client_id"]
            self.client.headers["X-Client-ID"] = self.client_id
        except Exception as e:
            print(f"Ошибка test_finalize (WebUser): {str(e)}")
            self.environment.events.request.fire(
                request_type="POST",
                name="/handshake/finalize_web",
                response_time=0,
                response_length=0,
                response=None,
                context={},
                exception=e,
                start_time=time.time(),
                url=f"{self.host}/handshake/finalize"
            )

    @task
    def test_session(self):
        if not hasattr(self, 'handshake_data') or not hasattr(self, 'client_id'):
            return
        try:
            start = time.time()
            ks = perform_finalize(f"{self.host}/handshake/finalize", self.handshake_data)
            K_enc, K_mac = derive_keys(ks)
            session = {
                "client_id": self.client_id,
                "k_enc": K_enc,
                "k_mac": K_mac,
                "ecdsa_priv": self.handshake_data["ecdsa_priv"]
            }
            large_data_10mb = os.urandom(10 * 1024 * 1024)
            perform_session_test(f"{self.host}/session/test", session, large_data_10mb)
            response_time = (time.time() - start) * 1000
            self.environment.events.request.fire(
                request_type="POST",
                name="/session/test_web",
                response_time=response_time,
                response_length=len(large_data_10mb),
                response=None,
                context={},
                exception=None,
                start_time=start,
                url=f"{self.host}/session/test"
            )
            # Перезапускаем рукопожатие для следующей итерации
            nonce = os.urandom(8)
            self.handshake_data = perform_handshake(f"{self.host}/handshake/init", nonce1=nonce)
            self.client_id = self.handshake_data["client_id"]
            self.client.headers["X-Client-ID"] = self.client_id
        except Exception as e:
            print(f"Ошибка test_session (WebUser): {str(e)}")
            self.environment.events.request.fire(
                request_type="POST",
                name="/session/test_web",
                response_time=0,
                response_length=0,
                response=None,
                context={},
                exception=e,
                start_time=time.time(),
                url=f"{self.host}/session/test"
            )

class PCUser(HttpUser):
    wait_time = between(3, 5)
    host = "http://localhost:8080"
    # host = "http://localhost:8081"  # Прокси

    def on_start(self):
        self.client.headers["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) DesktopApp/1.0"
        nonce = os.urandom(8)
        self.handshake_data = perform_handshake(f"{self.host}/handshake/init", nonce1=nonce)
        self.client_id = self.handshake_data["client_id"]
        self.client.headers["X-Client-ID"] = self.client_id

    @task
    def test_finalize(self):
        if not hasattr(self, 'handshake_data') or not hasattr(self, 'client_id'):
            return
        try:
            start = time.time()
            ks = perform_finalize(f"{self.host}/handshake/finalize", self.handshake_data)
            response_time = (time.time() - start) * 1000
            print(f"Время финального рукопожатия (PCUser): {response_time:.2f} мс")
            self.environment.events.request.fire(
                request_type="POST",
                name="/handshake/finalize_pc",
                response_time=response_time,
                response_length=len(ks),
                response=None,
                context={},
                exception=None,
                start_time=start,
                url=f"{self.host}/handshake/finalize"
            )
            nonce = os.urandom(8)
            self.handshake_data = perform_handshake(f"{self.host}/handshake/init", nonce1=nonce)
            self.client_id = self.handshake_data["client_id"]
            self.client.headers["X-Client-ID"] = self.client_id
        except Exception as e:
            print(f"Ошибка test_finalize (PCUser): {str(e)}")
            self.environment.events.request.fire(
                request_type="POST",
                name="/handshake/finalize_pc",
                response_time=0,
                response_length=0,
                response=None,
                context={},
                exception=e,
                start_time=time.time(),
                url=f"{self.host}/handshake/finalize"
            )

    @task
    def test_session(self):
        if not hasattr(self, 'handshake_data') or not hasattr(self, 'client_id'):
            return
        try:
            start = time.time()
            ks = perform_finalize(f"{self.host}/handshake/finalize", self.handshake_data)
            K_enc, K_mac = derive_keys(ks)
            session = {
                "client_id": self.client_id,
                "k_enc": K_enc,
                "k_mac": K_mac,
                "ecdsa_priv": self.handshake_data["ecdsa_priv"]
            }
            large_data_10mb = os.urandom(10 * 1024 * 1024)
            perform_session_test(f"{self.host}/session/test", session, large_data_10mb)
            response_time = (time.time() - start) * 1000
            self.environment.events.request.fire(
                request_type="POST",
                name="/session/test_pc",
                response_time=response_time,
                response_length=len(large_data_10mb),
                response=None,
                context={},
                exception=None,
                start_time=start,
                url=f"{self.host}/session/test"
            )
            # Перезапускаем рукопожатие для следующей итерации
            nonce = os.urandom(8)
            self.handshake_data = perform_handshake(f"{self.host}/handshake/init", nonce1=nonce)
            self.client_id = self.handshake_data["client_id"]
            self.client.headers["X-Client-ID"] = self.client_id
        except Exception as e:
            print(f"Ошибка test_session (PCUser): {str(e)}")
            self.environment.events.request.fire(
                request_type="POST",
                name="/session/test_pc",
                response_time=0,
                response_length=0,
                response=None,
                context={},
                exception=e,
                start_time=time.time(),
                url=f"{self.host}/session/test"
            )
