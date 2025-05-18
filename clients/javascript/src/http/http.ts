import axios, {AxiosResponse} from 'axios';
import {IInitResponse} from "../models/response/IInitResponse";
import {
    DerSig,
    derToSignature,
    encryptRSA,
    generateRandomBytes,
    loadRSAPubDER,
    parseECDSAPriv,
    signPayloadECDSA
} from "../utils/scrypto";
import {IInitRequest} from "../models/request/IInitRequest";
import crypto from "node:crypto";
import {ec as EC} from 'elliptic';
import {IFinalizeResponse} from "../models/response/IFinalizeResponse";
import {IFinalizeRequest} from "../models/request/IFinalizeRequest";


const verifyECDSASignature = (
    pubKey: EC.KeyPair,
    data: Buffer,
    signature: DerSig
): boolean => {
    const hash = crypto.createHash('sha256').update(data).digest('hex');
    const sigObj = {
        r: signature.R.toString(16).padStart(64, '0'), // Дополняем до 64 символов
        s: signature.S.toString(16).padStart(64, '0')  // для корректной работы
    };
    return pubKey.verify(hash, sigObj);
};

// Основные функции
export const postJSON = async (
    url: string,
    payload: any,
    headers: Record<string, string> = {}
): Promise<AxiosResponse> => {
    try {
        return await axios.post(url, payload, {
            headers: {
                'Content-Type': 'application/json',
                ...headers
            }
        });
    } catch (error) {
        if (axios.isAxiosError(error)) {
            throw new Error(
                `HTTP request failed: ${error.response?.status} ${JSON.stringify(error.response?.data)}`
            );
        }
        throw error;
    }
};

export const doInitAPI = async (
    url: string,
    rsaPubPEM: Buffer,
    ecdsaPubPEM: Buffer,
    ecdsaPrivPEM:  Buffer| Uint8Array
): Promise<IInitResponse> => {
    const ec = new EC('p256');
    const ecdsaPriv = parseECDSAPriv(Buffer.from(ecdsaPrivPEM));

    // Конвертируем PEM в DER
    const rsaPubDER = loadRSAPubDER(rsaPubPEM);
    const ecdsaPubDER = Buffer.from(ecdsaPriv.getPublic('hex'), 'hex');

    // Генерируем nonce1
    const [nonce1b64, nonce1] = generateRandomBytes(8);

    // Подписываем payload
    const toSign1 = Buffer.concat([rsaPubDER, ecdsaPubDER, nonce1]);
    const signature1 = signPayloadECDSA(ecdsaPriv, toSign1);

    // Формируем запрос
    const reqBody: IInitRequest = {
        ecdsa_pub_client: ecdsaPubDER.toString('base64'),
        rsa_pub_client: rsaPubDER.toString('base64'),
        nonce1: nonce1b64,
        signature1
    };

    const response = await postJSON(url, reqBody);
    if (response.status !== 200) {
        throw new Error(`handshake/init failed: status ${response.status}, body: ${JSON.stringify(response.data)}`);
    }

    const initResponse = response.data as IInitResponse;

    // Верификация подписи сервера
    const nonce2 = Buffer.from(initResponse.nonce2, 'base64');
    const sig2DER = Buffer.from(initResponse.signature2, 'base64');
    const sig2 = derToSignature(sig2DER);

    const serverECDSAPub = ec.keyFromPublic(
        Buffer.from(initResponse.ecdsa_pub_server, 'base64').toString('hex'),
        'hex'
    );

    const clientId = Buffer.from(initResponse.client_id);
    const verifyData = Buffer.concat([
        Buffer.from(initResponse.rsa_pub_server, 'base64'),
        Buffer.from(initResponse.ecdsa_pub_server, 'base64'),
        nonce2,
        nonce1,
        clientId
    ]);

    if (!verifyECDSASignature(serverECDSAPub, verifyData, sig2)) {
        throw new Error('handshake/init: server signature verification failed');
    }

    return initResponse;
};


export const doFinalizeAPI = async (
    url: string,
    initResponse: IInitResponse,
    ecdsaPrivPEM: Buffer | Uint8Array,
    rsaPubPemPath: string
): Promise<IFinalizeResponse> => {
    const ecdsaPriv = parseECDSAPriv(Buffer.from(ecdsaPrivPEM));
    const nonce2 = Buffer.from(initResponse.nonce2, 'base64');

    // Генерируем ks и nonce3
    const [_, ks] = generateRandomBytes(32);
    const [__, nonce3] = generateRandomBytes(8);

    // Подписываем payload
    const payload = Buffer.concat([ks, nonce3, nonce2]);
    const signature3 = signPayloadECDSA(ecdsaPriv, payload);
    const sig3DER = Buffer.from(signature3, 'base64');

    // Шифруем данные
    const toEncrypt = Buffer.concat([payload, sig3DER]);
    const encrypted = encryptRSA(rsaPubPemPath, toEncrypt);

    // Формируем запрос
    const reqBody: IFinalizeRequest = { encrypted };
    const headers = { 'X-Client-ID': initResponse.client_id };

    const response = await postJSON(url, reqBody, headers);
    if (response.status !== 200) {
        throw new Error(`handshake/finalize failed: status ${response.status}, body: ${JSON.stringify(response.data)}`);
    }

    const finalizeResponse = response.data as IFinalizeResponse;

    // Верификация подписи сервера
    const sig4DER = Buffer.from(finalizeResponse.signature4, 'base64');
    const sig4 = derToSignature(sig4DER);

    const ec = new EC('p256');
    const serverECDSAPub = ec.keyFromPublic(
        Buffer.from(initResponse.ecdsa_pub_server, 'base64').toString('hex'),
        'hex'
    );

    const verifyData = Buffer.concat([ks, nonce3, nonce2]);
    if (!verifyECDSASignature(serverECDSAPub, verifyData, sig4)) {
        throw new Error('finalize: bad server signature4');
    }

    console.log('Finalize OK, server signature verified');
    return finalizeResponse;
};