function rawECDSAToDER(rawSignature: Uint8Array): Uint8Array {
    const r = rawSignature.slice(0, 32); // Первые 32 байта — R
    const s = rawSignature.slice(32, 64); // Следующие 32 байта — S

    // Преобразуем R и S в формат DER (добавляем ведущий 0x00, если необходимо)
    const encodeInteger = (num: Uint8Array): Uint8Array => {
        // Если старший бит равен 1, добавляем ведущий 0x00, чтобы интерпретировать как положительное число
        const needsPadding = num[0] & 0x80 ? 1 : 0;
        const length = num.length + needsPadding;
        const result = new Uint8Array(length + 2); // 2 байта для тега INTEGER и длины
        result[0] = 0x02; // Тег INTEGER
        result[1] = length; // Длина
        if (needsPadding) {
            result[2] = 0x00;
            result.set(num, 3);
        } else {
            result.set(num, 2);
        }
        return result;
    };

    const rDer = encodeInteger(r);
    const sDer = encodeInteger(s);

    // Объединяем R и S в SEQUENCE
    const totalLength = rDer.length + sDer.length;
    const sequence = new Uint8Array(totalLength + 2); // 2 байта для тега SEQUENCE и длины
    sequence[0] = 0x30; // Тег SEQUENCE
    sequence[1] = totalLength; // Длина содержимого
    sequence.set(rDer, 2);
    sequence.set(sDer, 2 + rDer.length);

    return sequence;
}

export function derToRawECDSA(der: Uint8Array): Uint8Array {
    let pos = 0;
    if (der[pos++] !== 0x30) {
        throw new Error('Invalid DER sequence');
    }
    // читаем длину (1‑ или 2‑байтный вариант)
    let len = der[pos++];
    if (len & 0x80) {
        const n = len & 0x7f;
        len = 0;
        for (let i = 0; i < n; i++) {
            len = (len << 8) | der[pos++];
        }
    }
    // теперь должны идти два INTEGER‑а
    if (der[pos++] !== 0x02) throw new Error('Expected INTEGER for R');
    let rLen = der[pos++];
    let rStart = pos;
    pos += rLen;

    if (der[pos++] !== 0x02) throw new Error('Expected INTEGER for S');
    let sLen = der[pos++];
    let sStart = pos;
    //pos += sLen; // дальше нам не важно

    const rBytes = der.slice(rStart, rStart + rLen);
    const sBytes = der.slice(sStart, sStart + sLen);

    // Приводим к ровно 32‑байтовым, обрезая или дополняя спереди нулями
    function norm(buf: Uint8Array): Uint8Array {
        if (buf.length === 32) return buf;
        if (buf.length > 32) return buf.slice(buf.length - 32);
        const tmp = new Uint8Array(32);
        tmp.set(buf, 32 - buf.length);
        return tmp;
    }

    return new Uint8Array([
        ...norm(rBytes),
        ...norm(sBytes),
    ]);
}

// Функция для подписи данных с помощью ECDSA
export async function signDataWithECDSA(data: Uint8Array, privateKey: CryptoKey): Promise<string> {
    // Подписываем данные
    const signature = await crypto.subtle.sign(
        {
            name: "ECDSA",
            hash: {name: "SHA-256"}
        },
        privateKey,
        data
    );

    // Преобразуем сырую подпись в формат DER
    const derSignature = rawECDSAToDER(new Uint8Array(signature));

    // Кодируем в Base64
    return btoa(String.fromCharCode(...derSignature));
}

export async function createSignature1(
    clientRSAPublicKey: Uint8Array,
    clientECDSAPublicKey: Uint8Array,
    nonce1: Uint8Array,
    ecdsaPrivateKey: CryptoKey
): Promise<string> {
    // Создаем буфер для объединенных данных
    const combinedData = new Uint8Array(
        clientRSAPublicKey.length + clientECDSAPublicKey.length + nonce1.length
    );

    // Копируем данные в буфер
    combinedData.set(clientRSAPublicKey, 0);
    combinedData.set(clientECDSAPublicKey, clientRSAPublicKey.length);
    combinedData.set(nonce1, clientRSAPublicKey.length + clientECDSAPublicKey.length);

    return await signDataWithECDSA(new Uint8Array(combinedData), ecdsaPrivateKey);
}

export function generateNonce(bytes: number): [string, Uint8Array] {
    const nonce = new Uint8Array(bytes);
    crypto.getRandomValues(nonce);
    const base64 = btoa(String.fromCharCode(...nonce));
    return [base64, nonce];
}

export async function encryptRSA(payload: Uint8Array, rsaPubServer: Uint8Array): Promise<string> {
    // Импортируем публичный ключ
    const publicKey = await crypto.subtle.importKey(
        'spki',
        rsaPubServer,
        {
            name: 'RSA-OAEP',
            hash: 'SHA-256'
        },
        false,
        ['encrypt']
    );

    // Шифруем payload
    const encrypted = await crypto.subtle.encrypt(
        {
            name: 'RSA-OAEP'
        },
        publicKey,
        payload
    );

    // Конвертируем ArrayBuffer в Base64
    const encryptedArray = new Uint8Array(encrypted);
    const binary = String.fromCharCode(...encryptedArray);
    return btoa(binary);
}

export async function encryptAES256CBC(key: CryptoKey, iv: Uint8Array, data: Uint8Array): Promise<Uint8Array> {
    // Добавляем PKCS#7 padding
    const blockSize = 16; // AES block size
    const padLength = blockSize - (data.length % blockSize);
    const padded = new Uint8Array(data.length + padLength);
    padded.set(data);
    padded.fill(padLength, data.length);

    const ciphertext = await crypto.subtle.encrypt(
        {
            name: 'AES-CBC',
            iv: iv
        },
        key,
        padded
    );

    return new Uint8Array(ciphertext);
}

export async function deriveAESKey(keyMaterial: Uint8Array): Promise<CryptoKey> {
    return await crypto.subtle.importKey(
        'raw',
        keyMaterial,
        { name: 'AES-CBC', length: 256 },
        false,
        ['encrypt']
    );
}

export async function deriveHMACKey(keyMaterial: Uint8Array): Promise<CryptoKey> {
    return await crypto.subtle.importKey(
        'raw',
        keyMaterial,
        {
            name: 'HMAC',
            hash: { name: 'SHA-256' }
        },
        false,
        ['sign']
    );
}

export async function deriveKeyBytes(keyMaterial: Uint8Array, purpose: string): Promise<Uint8Array> {
    const hmacKey = await crypto.subtle.importKey(
        'raw',
        keyMaterial,
        { name: 'HMAC', hash: 'SHA-256' },
        false,
        ['sign']
    );

    const purposeBytes = new TextEncoder().encode(purpose);
    const signature = await crypto.subtle.sign(
        'HMAC',
        hmacKey,
        purposeBytes
    );

    return new Uint8Array(signature);
}