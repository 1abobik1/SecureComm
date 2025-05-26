import {encryptAES256CBC} from "@/app/api/services/HandshakeService/utils/scrypto";

export const buildEncryptedBlob = async (plainData: Uint8Array, kEnc: CryptoKey, kMac: CryptoKey): Promise<Uint8Array> => {
    // 1) Генерируем nonce (16 байт)
    const nonce = crypto.getRandomValues(new Uint8Array(16));

    // 2) Генерируем IV (16 байт)
    const iv = crypto.getRandomValues(new Uint8Array(16));

    // 3) Шифруем данные AES-CBC с PKCS#7 padding
    const ciphertext = await encryptAES256CBC(kEnc, iv, plainData);

    // 4) Создаем HMAC-SHA256(iv || ciphertext)
    const hmacData = new Uint8Array([...iv, ...ciphertext]);
    const tag = await crypto.subtle.sign(
        { name: 'HMAC' },
        kMac,
        hmacData
    );

    // 5) Собираем итоговый blob: nonce||iv||ciphertext||tag
    const result = new Uint8Array(
        nonce.byteLength + iv.byteLength + ciphertext.byteLength + tag.byteLength
    );
    result.set(nonce, 0);
    result.set(iv, nonce.byteLength);
    result.set(new Uint8Array(ciphertext), nonce.byteLength + iv.byteLength);
    result.set(new Uint8Array(tag), nonce.byteLength + iv.byteLength + ciphertext.byteLength);

    return result;
};