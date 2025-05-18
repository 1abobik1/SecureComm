import fs, {readFileSync} from 'fs';
import {createPrivateKey} from 'crypto';
import forge from 'node-forge';

export function loadDERPub(path: string): Buffer {
    const pem = fs.readFileSync(path, 'utf8');
    const pubKey = forge.pki.publicKeyFromPem(pem);
    const der = forge.asn1.toDer(forge.pki.publicKeyToAsn1(pubKey));
    return Buffer.from(der.getBytes(), 'binary');
}

export function loadECDSAPriv(path: string): Buffer{
    const pemContent = readFileSync(path, 'utf8');

    const key = createPrivateKey({
        key: pemContent,
        format: 'pem',
    });
    const derFormat = key.export({
        format: 'der',
        type: 'sec1'
    });

    // 4. Возвращаем как Buffer
    return Buffer.from(derFormat);
}
