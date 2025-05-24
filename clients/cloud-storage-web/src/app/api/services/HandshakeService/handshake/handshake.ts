import {IInitResponse} from "../models/response/IInitResponse";
import {createSignature1, derToRawECDSA, encryptRSA, generateNonce, signDataWithECDSA} from "../utils/scrypto";
import {IInitRequest} from "../models/request/IInitRequest";
import {IFinalizeResponse} from "../models/response/IFinalizeResponse";
import {IFinalizeRequest} from "../models/request/IFinalizeRequest";
import {postJSON} from "../utils/postJSON";

// Основные функции
export const doInitAPI = async (
    url: string,
    rsaPubDER: Uint8Array,
    ecdsaPubDER: Uint8Array,
    ecdsaPriv:  CryptoKey
): Promise<IInitResponse> => {

    // Генерируем nonce1
    const [nonce1b64, nonce1] = generateNonce(8);


    const signature1 = await createSignature1(
        rsaPubDER,
        ecdsaPubDER,
        nonce1,
        ecdsaPriv
    );

    // Формируем запрос
    const reqBody: IInitRequest = {
        ecdsa_pub_client: btoa(String.fromCharCode(...ecdsaPubDER)),
        rsa_pub_client: btoa(String.fromCharCode(...rsaPubDER)),
        nonce1: nonce1b64,
        signature1
    };


    const response = await postJSON(url, reqBody);
    const initResponse = response.data as IInitResponse;

    // --- Верификация ответа сервера ---

    // 1. Декодируем Base64 данные из ответа
    const serverRSAPubDER = Uint8Array.from(atob(initResponse.rsa_pub_server), c => c.charCodeAt(0));
    const serverECDSAPubDER = Uint8Array.from(atob(initResponse.ecdsa_pub_server), c => c.charCodeAt(0));
    const nonce2 = Uint8Array.from(atob(initResponse.nonce2), c => c.charCodeAt(0));
    const signature2 = Uint8Array.from(atob(initResponse.signature2), c => c.charCodeAt(0));

    // 2. Импортируем ECDSA публичный ключ сервера
    const serverECDSAPublicKey = await crypto.subtle.importKey(
        "spki",
        serverECDSAPubDER,
        { name: "ECDSA", namedCurve: "P-256" },
        true,
        ["verify"]
    );

    // 3. Подготавливаем данные для верификации
    // Формат: rsaServer || ecdsaServer || nonce2 || nonce1 || clientID
    const verifyData = new Uint8Array([
        ...serverRSAPubDER,
        ...serverECDSAPubDER,
        ...nonce2,
        ...nonce1,
        ...new TextEncoder().encode(initResponse.client_id)
    ]);


    const derSig2 = Uint8Array.from(atob(initResponse.signature2), c => c.charCodeAt(0));
    const rawSig2 = derToRawECDSA(derSig2);

    const isValid = await crypto.subtle.verify(
        { name: "ECDSA", hash: "SHA-256" },
        serverECDSAPublicKey,
        rawSig2,           //  передаём именно R||S, не DER
        verifyData
    );

    if (!isValid) {
        throw new Error('handshake/init: server signature verification failed');
    }

    // Если все проверки пройдены, возвращаем ответ
    return initResponse;
};


export const doFinalizeAPI = async (
    url: string,
    initResponse: IInitResponse,
    ecdsaPriv: CryptoKey
): Promise<[IFinalizeResponse, string]> => {
    const rsaPubServer = Uint8Array.from(atob(initResponse.rsa_pub_server), c => c.charCodeAt(0));
    const nonce2 = Uint8Array.from(atob(initResponse.nonce2), c => c.charCodeAt(0));

    // Генерируем ks и nonce3
    const [ksb64, ks] = generateNonce(32);
    const [__, nonce3] = generateNonce(8);

    //создаем payload
    const payload = new Uint8Array(ks.length + nonce3.length + nonce2.length);
    payload.set(ks, 0);
    payload.set(nonce3, ks.length);
    payload.set(nonce2, ks.length + nonce3.length);

    // подписываем payload, и получаем signature3
    const signature3 = await signDataWithECDSA(new Uint8Array(payload), ecdsaPriv);

    // Шифруем данные
    const encrypted = await encryptRSA(payload, rsaPubServer);

    // Формируем запрос
    const reqBody: IFinalizeRequest = {encrypted, signature3};
    const headers = { 'X-Client-ID': initResponse.client_id };

    const response = await postJSON(url, reqBody, headers);
    if (response.status !== 200) {
        throw new Error(`handshake/finalize failed: status ${response.status}, body: ${JSON.stringify(response.data)}`);
    }

    const finalizeResponse = response.data as IFinalizeResponse;

    // Верификация подписи сервера
    const signature4 = Uint8Array.from(atob(finalizeResponse.signature4), c => c.charCodeAt(0));

    const serverECDSAPub = Uint8Array.from(atob(initResponse.ecdsa_pub_server), c => c.charCodeAt(0));
    // Импортируем ECDSA публичный ключ сервера
    const serverECDSAPublicKey = await crypto.subtle.importKey(
        "spki",
        serverECDSAPub,
        { name: "ECDSA", namedCurve: "P-256" },
        true,
        ["verify"]
    );

    // Подготавливаем данные для верификации: ks || nonce3 || nonce2
    const verifyData = new Uint8Array([
        ...ks,
        ...nonce3,
        ...nonce2
    ]);

    // Конвертируем подпись из DER в raw формат (R||S)
    const rawSig4 = derToRawECDSA(signature4);

    // Проверяем подпись
    const isValid = await crypto.subtle.verify(
        { name: "ECDSA", hash: "SHA-256" },
        serverECDSAPublicKey,
        rawSig4,
        verifyData
    );

    if (!isValid) {
        throw new Error('handshake/finalize: server signature verification failed');
    }


    return [finalizeResponse,ksb64];

};