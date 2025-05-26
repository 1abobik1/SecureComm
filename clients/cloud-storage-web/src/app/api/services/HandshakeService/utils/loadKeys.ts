export async function generateRSAPublicKeyDER(): Promise<Uint8Array> {
    const keyPair = await crypto.subtle.generateKey(
        {
            name: "RSA-PSS",
            modulusLength: 3072, // можно изменить на нужную длину
            publicExponent: new Uint8Array([0x01, 0x00, 0x01]), // 65537
            hash: "SHA-256",
        },
        true, // ключи можно экспортировать
        ["sign", "verify"]
    );

    // Экспортируем публичный ключ в формате SPKI (DER)
    const spkiDer = await crypto.subtle.exportKey("spki", keyPair.publicKey);

    return new Uint8Array(spkiDer);
}

export async function generateECDSAKeys(): Promise<[Uint8Array, CryptoKey]> {
    // Генерация ключевой пары
    const keyPair = await crypto.subtle.generateKey(
        {
            name: "ECDSA",
            namedCurve: "P-256",
        },
        true,
        ["sign", "verify"]
    );

    // Экспорт публичного ключа в формате SPKI (DER)
    const publicKeyDer = await crypto.subtle.exportKey("spki", keyPair.publicKey);

    // Экспорт приватного ключа в формате PKCS8 (DER)
    const privateKeyDer = await crypto.subtle.exportKey("pkcs8", keyPair.privateKey);

    return [new Uint8Array(publicKeyDer), keyPair.privateKey];
}
