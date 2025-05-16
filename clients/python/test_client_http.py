import unittest
import base64
import requests
import time
from cryptography.hazmat.primitives import serialization, hashes
from cryptography.hazmat.primitives.asymmetric import padding, ec
from client_http import perform_handshake, perform_finalize, derive_keys, generate_keys, generate_nonce, sign_ecdsa, verify_ecdsa


class TestHandshakeSecurity(unittest.TestCase):
    def setUp(self):
        self.init_url = "http://localhost:8080/handshake/init"
        self.fin_url = "http://localhost:8080/handshake/finalize"
        # Задержка перед каждым тестом, чтобы избежать 429
        time.sleep(1)

    def test_replay_attack_init(self):
        # Тест на replay-атаку: повторная отправка того же nonce1
        handshake_data = perform_handshake(self.init_url)
        nonce1 = handshake_data["nonce1"]
        with self.assertRaises(requests.RequestException):
            perform_handshake(self.init_url, nonce1=nonce1)  # Повторяем тот же nonce1

    def test_replay_attack_finalize(self):
        # Тест на replay-атаку: повторная отправка того же nonce3
        handshake_data = perform_handshake(self.init_url)
        _, nonce3 = generate_nonce(8)
        ks = perform_finalize(self.fin_url, handshake_data, nonce3=nonce3)
        try:
            perform_finalize(self.fin_url, handshake_data, nonce3=nonce3)  # Повторяем тот же nonce3
            self.fail("Ожидалась ошибка при повторной отправке nonce3")
        except requests.HTTPError as e:
            # Ожидаем HTTP 409 Conflict как подтверждение защиты от replay-атаки
            self.assertEqual(e.response.status_code, 409, "Ожидался код 409 для повторного nonce3")

    def test_timing_attack_signature(self):
        # Тест на timing-атаку: сравнение времени проверки валидной и невалидной подписи
        handshake_data = perform_handshake(self.init_url)
        valid_signature = handshake_data["ecdsa_priv"].sign(
            handshake_data["nonce1"], ec.ECDSA(hashes.SHA256())
        )
        invalid_signature = valid_signature[:-1] + (b'\x00' if valid_signature[-1] != b'\x00' else b'\x01')

        # Усредняем время для нескольких запусков, чтобы уменьшить шум
        num_runs = 5
        valid_times = []
        invalid_times = []
        for _ in range(num_runs):
            start = time.time_ns()
            verify_ecdsa(handshake_data["ecdsa_pub_server"], handshake_data["nonce1"],
                         base64.b64encode(valid_signature).decode('utf-8'))
            valid_times.append(time.time_ns() - start)

            start = time.time_ns()
            verify_ecdsa(handshake_data["ecdsa_pub_server"], handshake_data["nonce1"],
                         base64.b64encode(invalid_signature).decode('utf-8'))
            invalid_times.append(time.time_ns() - start)

        # Среднее время
        avg_valid_time = sum(valid_times) / num_runs
        avg_invalid_time = sum(invalid_times) / num_runs

        # Проверяем, что разница во времени минимальна (порог 1 мс)
        self.assertLess(abs(avg_valid_time - avg_invalid_time), 1_000_000,
                        "Время проверки подписи не константное")

    def test_brute_force_client_id(self):
        # Тест на brute-force: попытка отправки запросов с неверным client_id
        handshake_data = perform_handshake(self.init_url)
        invalid_client_id = "invalid_" + handshake_data["client_id"]
        _, ks = generate_nonce(32)
        nonce3_b64, nonce3 = generate_nonce(8)
        payload = ks + nonce3 + handshake_data["nonce2"]
        sig3 = sign_ecdsa(handshake_data["ecdsa_priv"], payload)
        sig3_der = base64.b64decode(sig3)
        to_encrypt = payload + sig3_der
        encrypted = handshake_data["rsa_pub_server"].encrypt(
            to_encrypt,
            padding.OAEP(
                mgf=padding.MGF1(algorithm=hashes.SHA256()),
                algorithm=hashes.SHA256(),
                label=None
            )
        )
        encrypted_b64 = base64.b64encode(encrypted).decode('utf-8')
        payload = {"encrypted": encrypted_b64}
        headers = {"X-Client-ID": invalid_client_id}

        with self.assertRaises(requests.RequestException):
            response = requests.post(self.fin_url, json=payload, headers=headers, timeout=5)
            response.raise_for_status()

    def test_mitm_key_substitution(self):
        # Тест на MitM: подмена публичного ключа сервера
        handshake_data = perform_handshake(self.init_url)
        # Создаём фальшивый ключ
        fake_priv, fake_pub, _, _ = generate_keys()
        fake_pub_der = fake_pub.public_bytes(
            encoding=serialization.Encoding.DER,
            format=serialization.PublicFormat.SubjectPublicKeyInfo
        )
        data_to_verify = fake_pub_der + handshake_data["ecdsa_pub_server"].public_bytes(
            encoding=serialization.Encoding.DER,
            format=serialization.PublicFormat.SubjectPublicKeyInfo
        ) + handshake_data["nonce2"] + handshake_data["nonce1"] + handshake_data["client_id"].encode('utf-8')
        signature2_b64 = sign_ecdsa(handshake_data["ecdsa_priv"], data_to_verify)  # Фальшивая подпись
        self.assertFalse(
            verify_ecdsa(handshake_data["ecdsa_pub_server"], data_to_verify, signature2_b64),
            "Подмена ключа сервера не обнаружена"
        )

    def test_functional_success(self):
        # Функциональный тест: успешное выполнение рукопожатия
        handshake_data = perform_handshake(self.init_url)
        self.assertIsNotNone(handshake_data["client_id"])
        ks = perform_finalize(self.fin_url, handshake_data)
        self.assertIsNotNone(ks)
        K_enc, K_mac = derive_keys(ks)
        self.assertEqual(len(K_enc), 32)
        self.assertEqual(len(K_mac), 32)


if __name__ == "__main__":
    unittest.main()
# написать прокси который подменяет данные