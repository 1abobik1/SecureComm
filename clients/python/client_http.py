import base64
import os
import requests
import time
from cryptography.hazmat.primitives.asymmetric import rsa, ec, padding
from cryptography.hazmat.primitives import serialization, hashes
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.primitives.hmac import HMAC
from cryptography.hazmat.primitives.padding import PKCS7

# Генерирует пару RSA и ECDSA ключей для клиента
def generate_keys():
    rsa_private_key = rsa.generate_private_key(public_exponent=65537, key_size=3072)
    rsa_public_key = rsa_private_key.public_key()
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
    except Exception as e:
        print(f"Verification failed: {e}")
        return False

# Генерирует случайный nonce заданного размера, возвращает в base64 и сырые байты
def generate_nonce(size):
    buf = os.urandom(size)
    return base64.b64encode(buf).decode('utf-8'), buf

# Выполняет начальный этап обмена ключами с сервером
def perform_handshake(init_url, nonce1=None):
    rsa_priv, rsa_pub, ecdsa_priv, ecdsa_pub = generate_keys()
    rsa_pub_der = rsa_pub.public_bytes(
        encoding=serialization.Encoding.DER,
        format=serialization.PublicFormat.SubjectPublicKeyInfo
    )
    ecdsa_pub_der = ecdsa_pub.public_bytes(
        encoding=serialization.Encoding.DER,
        format=serialization.PublicFormat.SubjectPublicKeyInfo
    )
    rsa_pub_der_b64 = base64.b64encode(rsa_pub_der).decode()
    ecdsa_pub_der_b64 = base64.b64encode(ecdsa_pub_der).decode()
    nonce1_b64, nonce1 = generate_nonce(8) if nonce1 is None else (base64.b64encode(nonce1).decode('utf-8'), nonce1)
    data_to_sign = base64.b64decode(rsa_pub_der_b64) + base64.b64decode(ecdsa_pub_der_b64) + base64.b64decode(nonce1_b64)
    signature1 = sign_ecdsa(ecdsa_priv, data_to_sign)
    payload = {
        "rsa_pub_client": rsa_pub_der_b64,
        "ecdsa_pub_client": ecdsa_pub_der_b64,
        "nonce1": nonce1_b64,
        "signature1": signature1
    }
    response = requests.post(init_url, json=payload)
    response.raise_for_status()
    server_data = response.json()
    client_id = server_data["client_id"]
    rsa_pub_server_der = base64.b64decode(server_data["rsa_pub_server"])
    ecdsa_pub_server_der = base64.b64decode(server_data["ecdsa_pub_server"])
    nonce2 = base64.b64decode(server_data["nonce2"])
    signature2_b64 = server_data["signature2"]
    rsa_pub_server = serialization.load_der_public_key(rsa_pub_server_der)
    ecdsa_pub_server = serialization.load_der_public_key(ecdsa_pub_server_der)
    data_to_verify = rsa_pub_server_der + ecdsa_pub_server_der + nonce2 + nonce1 + client_id.encode('utf-8')
    if not verify_ecdsa(ecdsa_pub_server, data_to_verify, signature2_b64):
        raise Exception("Ошибка проверки подписи сервера")
    return {
        "client_id": client_id,
        "rsa_priv": rsa_priv,
        "ecdsa_priv": ecdsa_priv,
        "rsa_pub_server": rsa_pub_server,
        "ecdsa_pub_server": ecdsa_pub_server,
        "nonce2": nonce2,
        "nonce1": nonce1,
        "nonce1_b64": nonce1_b64
    }

# Выполняет финальный этап обмена ключами, устанавливает сессионный ключ
def perform_finalize(fin_url, handshake_data, nonce3=None):
    _, ks = generate_nonce(32)
    nonce3_b64, nonce3 = generate_nonce(8) if nonce3 is None else (base64.b64encode(nonce3).decode('utf-8'), nonce3)
    payload = ks + nonce3 + handshake_data["nonce2"]
    data_to_sign = payload
    sig3 = sign_ecdsa(handshake_data["ecdsa_priv"], data_to_sign)
    to_encrypt = payload
    encrypted = handshake_data["rsa_pub_server"].encrypt(
        to_encrypt,
        padding.OAEP(
            mgf=padding.MGF1(algorithm=hashes.SHA256()),
            algorithm=hashes.SHA256(),
            label=None
        )
    )
    encrypted_b64 = base64.b64encode(encrypted).decode('utf-8')
    payload = {
        "encrypted": encrypted_b64,
        "signature3": sig3
    }
    headers = {"X-Client-ID": handshake_data["client_id"]}
    response = requests.post(fin_url, json=payload, headers=headers, timeout=5)
    response.raise_for_status()
    response_data = response.json()
    signature4_b64 = response_data["signature4"]
    data_to_verify = ks + nonce3 + handshake_data["nonce2"]
    if not verify_ecdsa(handshake_data["ecdsa_pub_server"], data_to_verify, signature4_b64):
        raise Exception("Ошибка проверки подписи сервера (signature4)")
    return ks

# Деривирует ключи шифрования (K_enc) и MAC (K_mac) из сессионного ключа
def derive_keys(ks):
    h_enc = HMAC(ks, hashes.SHA256())
    h_enc.update(b"enc")
    k_enc = h_enc.finalize()

    h_mac = HMAC(ks, hashes.SHA256())
    h_mac.update(b"mac")
    k_mac = h_mac.finalize()

    return k_enc, k_mac

# Выполняет тест сессии, отправляет зашифрованное сообщение
def perform_session_test(test_url, session, plaintext="Hello, Secure World!"):
    start = time.time()
    ts = int(time.time() * 1000)  # Timestamp в миллисекундах
    timestamp = ts.to_bytes(8, byteorder='big')
    nonce = os.urandom(16)  # 16 байт для nonce
    # Принимаем plaintext как str или bytes
    if isinstance(plaintext, str):
        plaintext_bytes = plaintext.encode('utf-8')
    else:
        plaintext_bytes = plaintext
    blob = timestamp + nonce + plaintext_bytes
    padder = PKCS7(128).padder()
    padded = padder.update(blob) + padder.finalize()
    iv = os.urandom(16)
    cipher = Cipher(algorithms.AES(session["k_enc"]), modes.CBC(iv))
    encryptor = cipher.encryptor()
    ciphertext = encryptor.update(padded) + encryptor.finalize()
    h = HMAC(session["k_mac"], hashes.SHA256())
    h.update(iv)
    h.update(ciphertext)
    tag = h.finalize()
    pkg = iv + ciphertext + tag
    b64msg = base64.b64encode(pkg).decode('utf-8')
    signature = sign_ecdsa(session["ecdsa_priv"], pkg)
    payload = {
        "encrypted_message": b64msg,
        "client_signature": signature
    }
    headers = {"X-Client-ID": session["client_id"], "Content-Type": "application/json"}
    response = requests.post(test_url, json=payload, headers=headers, timeout=30)
    response.raise_for_status()
    response_data = response.json()
    duration = (time.time() - start) * 1000
    print(f"Время шифрования и отправки (/session/test, {len(plaintext_bytes)/1024/1024:.2f} МБ): {duration:.2f} мс")
    print("Тест сессии успешно завершён!")
    print(f"Сервер расшифровал: {len(response_data['plaintext'])} символов")
    if len(response_data['plaintext']) != len(plaintext_bytes):
        print(f"Предупреждение: Сервер вернул {len(response_data['plaintext'])} байт, ожидалось {len(plaintext_bytes)}")

# Основной процесс
def main():
    # init_url = "http://localhost:8080/handshake/init"
    # fin_url = "http://localhost:8080/handshake/finalize"
    # test_url = "http://localhost:8080/session/test"
    init_url = "http://localhost:8081/handshake/init"
    fin_url = "http://localhost:8081/handshake/finalize"
    test_url = "http://localhost:8081/session/test"
    start_total = time.time()
    handshake_data = perform_handshake(init_url)
    print(f"Обмен ключами успешно завершен, идентификатор клиента: {handshake_data['client_id']}")
    ks = perform_finalize(fin_url, handshake_data)
    print(f"Время финального рукопожатия: {(time.time() - start_total) * 1000:.2f} мс")
    print("Финализация успешно завершена, сессионный ключ установлен")
    K_enc, K_mac = derive_keys(ks)
    session = {
        "client_id": handshake_data["client_id"],
        "k_enc": K_enc,
        "k_mac": K_mac,
        "ecdsa_priv": handshake_data["ecdsa_priv"]
    }
    print(f"Ключи сгенерированы: K_enc={base64.b64encode(K_enc).decode('utf-8')[:10]}..., K_mac={base64.b64encode(K_mac).decode('utf-8')[:10]}...")
    perform_session_test(test_url, session)
    total_time = (time.time() - start_total) * 1000
    print(f"Общее время выполнения: {total_time:.2f} мс")

if __name__ == "__main__":
    main()