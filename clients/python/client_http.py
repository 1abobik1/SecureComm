import base64
import os
import requests
import time
from cryptography.hazmat.primitives.asymmetric import rsa, ec, padding
from cryptography.hazmat.primitives import serialization, hashes
from cryptography.hazmat.primitives.kdf.hkdf import HKDF


# Генерирует пару RSA и ECDSA ключей для клиента
def generate_keys():
    # RSA (3072 бит)
    rsa_private_key = rsa.generate_private_key(public_exponent=65537, key_size=3072)
    rsa_public_key = rsa_private_key.public_key()

    # ECDSA (P-256)
    ecdsa_private_key = ec.generate_private_key(curve=ec.SECP256R1())
    ecdsa_public_key = ecdsa_private_key.public_key()

    return rsa_private_key, rsa_public_key, ecdsa_private_key, ecdsa_public_key


# Сериализует ключ в формат DER (или PEM для приватных) и кодирует в base64 для публичных
def serialize_key_to_der(key, is_private=False):
    if is_private:
        pem = key.private_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PrivateFormat.PKCS8,
            encryption_algorithm=serialization.NoEncryption()
        )
        return pem
    else:
        der = key.public_bytes(
            encoding=serialization.Encoding.DER,
            format=serialization.PublicFormat.SubjectPublicKeyInfo
        )
        return base64.b64encode(der).decode('utf-8')


# Создает ECDSA подпись для данных и возвращает её в base64
def sign_ecdsa(ecdsa_private_key, data):
    signature = ecdsa_private_key.sign(data, ec.ECDSA(hashes.SHA256()))
    return base64.b64encode(signature).decode('utf-8')


# Проверяет ECDSA подпись, возвращает True если подпись верна
def verify_ecdsa(ecdsa_public_key, data, signature_b64):
    signature = base64.b64decode(signature_b64)
    try:
        ecdsa_public_key.verify(signature, data, ec.ECDSA(hashes.SHA256()))
        return True
    except Exception:
        return False


# Генерирует случайный nonce заданного размера, возвращает в base64 и сырые байты
def generate_nonce(size):
    buf = os.urandom(size)
    return base64.b64encode(buf).decode('utf-8'), buf


# Выполняет начальный этап рукопожатия с сервером
def perform_handshake(init_url, nonce1=None):
    # Генерация ключей клиента
    rsa_priv, rsa_pub, ecdsa_priv, ecdsa_pub = generate_keys()

    # Сериализация публичных ключей в DER
    rsa_pub_der = rsa_pub.public_bytes(
        encoding=serialization.Encoding.DER,
        format=serialization.PublicFormat.SubjectPublicKeyInfo
    )
    ecdsa_pub_der = ecdsa_pub.public_bytes(
        encoding=serialization.Encoding.DER,
        format=serialization.PublicFormat.SubjectPublicKeyInfo
    )

    # Кодирование ключей в base64 для JSON
    rsa_pub_der_b64 = base64.b64encode(rsa_pub_der).decode()
    ecdsa_pub_der_b64 = base64.b64encode(ecdsa_pub_der).decode()

    # Генерация nonce1 или использование переданного (для тестов)
    nonce1_b64, nonce1 = generate_nonce(8) if nonce1 is None else (base64.b64encode(nonce1).decode('utf-8'), nonce1)

    # Подпись данных (RSA pub + ECDSA pub + nonce1)
    data_to_sign = rsa_pub_der + ecdsa_pub_der + nonce1
    signature1 = sign_ecdsa(ecdsa_priv, data_to_sign)

    # Формирование запроса
    payload = {
        "rsa_pub_client": rsa_pub_der_b64,
        "ecdsa_pub_client": ecdsa_pub_der_b64,
        "nonce1": nonce1_b64,
        "signature1": signature1
    }

    # Отправка запроса на /handshake/init
    response = requests.post(init_url, json=payload)
    response.raise_for_status()

    # Обработка ответа сервера
    server_data = response.json()
    client_id = server_data["client_id"]
    rsa_pub_server_der = base64.b64decode(server_data["rsa_pub_server"])
    ecdsa_pub_server_der = base64.b64decode(server_data["ecdsa_pub_server"])
    nonce2 = base64.b64decode(server_data["nonce2"])
    signature2_b64 = server_data["signature2"]

    # Парсинг публичных ключей сервера
    rsa_pub_server = serialization.load_der_public_key(rsa_pub_server_der)
    ecdsa_pub_server = serialization.load_der_public_key(ecdsa_pub_server_der)

    # Проверка подписи сервера (с добавлением client_id)
    data_to_verify = rsa_pub_server_der + ecdsa_pub_server_der + nonce2 + nonce1 + client_id.encode('utf-8')
    if not verify_ecdsa(ecdsa_pub_server, data_to_verify, signature2_b64):
        raise Exception("Ошибка проверки подписи сервера")

    # Возвращаем данные для финализации
    return {
        "client_id": client_id,
        "rsa_priv": rsa_priv,
        "ecdsa_priv": ecdsa_priv,
        "rsa_pub_server": rsa_pub_server,
        "ecdsa_pub_server": ecdsa_pub_server,
        "nonce2": nonce2,
        "nonce1": nonce1  # Сохраняем для тестов
    }


# Выполняет финальный этап рукопожатия, устанавливает сессионный ключ
def perform_finalize(fin_url, handshake_data, nonce3=None):
    # Генерация сессионного ключа (ks) и nonce3
    _, ks = generate_nonce(32)
    nonce3_b64, nonce3 = generate_nonce(8) if nonce3 is None else (base64.b64encode(nonce3).decode('utf-8'), nonce3)

    # Подпись данных (ks + nonce3 + nonce2)
    payload = ks + nonce3 + handshake_data["nonce2"]
    sig3 = sign_ecdsa(handshake_data["ecdsa_priv"], payload)
    sig3_der = base64.b64decode(sig3)

    # Шифрование данных (payload + sig3) с использованием RSA-ключа сервера
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

    # Формирование запроса
    payload = {"encrypted": encrypted_b64}
    headers = {"X-Client-ID": handshake_data["client_id"]}

    # Отправка запроса на /handshake/finalize
    response = requests.post(fin_url, json=payload, headers=headers, timeout=5)
    response.raise_for_status()

    # Проверка signature4
    response_data = response.json()
    signature4_b64 = response_data["signature4"]
    data_to_verify = ks + nonce3 + handshake_data["nonce2"]
    if not verify_ecdsa(handshake_data["ecdsa_pub_server"], data_to_verify, signature4_b64):
        raise Exception("Ошибка проверки подписи сервера (signature4)")

    # Возвращаем сессионный ключ
    return ks


# Деривирует ключи шифрования (K_enc) и MAC (K_mac) из сессионного ключа
def derive_keys(ks):
    hkdf = HKDF(
        algorithm=hashes.SHA256(),
        length=64,
        salt=None,
        info=b"encryption and mac"
    )
    derived_keys = hkdf.derive(ks)
    return derived_keys[:32], derived_keys[32:]  # K_enc, K_mac


# Основной процесс: выполняет рукопожатие, финализацию и деривацию ключей
def main():
    init_url = "http://localhost:8081/handshake/init"  # Прокси меняем когда тестим
    fin_url = "http://localhost:8081/handshake/finalize"  # Прокси меняем когда тестим
    # init_url = "http://localhost:8080/handshake/init"
    # fin_url = "http://localhost:8080/handshake/finalize"

    # Замер общего времени выполнения
    start_total = time.time()

    # Инициализация рукопожатия
    handshake_data = perform_handshake(init_url)
    print(f"Рукопожатие успешно завершено, идентификатор клиента: {handshake_data['client_id']}")

    # Финализация
    ks = perform_finalize(fin_url, handshake_data)
    print("Финализация успешно завершена, сессионный ключ установлен")

    # Деривация ключей
    K_enc, K_mac = derive_keys(ks)
    print(
        f"Ключи сгенерированы: K_enc={base64.b64encode(K_enc).decode('utf-8')[:10]}..., K_mac={base64.b64encode(K_mac).decode('utf-8')[:10]}..."
    )

    # Общее время выполнения
    total_time = (time.time() - start_total) * 1000  # В миллисекундах
    print(f"Общее время выполнения: {total_time:.2f} мс")


if __name__ == "__main__":
    main()