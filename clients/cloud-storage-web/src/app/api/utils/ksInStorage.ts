import {LocalStorage} from 'ttl-localstorage';

export function setKs(value: CryptoKey, ttlSeconds: number) {
    LocalStorage.put('ks', value, ttlSeconds);
}

export function setKsLogin(value: {
    k_enc_iv: Uint8Array;
    k_enc_data: Uint8Array;
    k_mac_iv: Uint8Array;
    k_mac_data: Uint8Array;
}, ttlSeconds: number) {
    LocalStorage.put('ks', value, ttlSeconds);
}


export function getKs(): CryptoKey | {
    k_enc_iv: Uint8Array;
    k_enc_data: Uint8Array;
    k_mac_iv: Uint8Array;
    k_mac_data: Uint8Array;
}{
    return LocalStorage.get('ks');
}


export function removeKs() {
    LocalStorage.removeKey('ks');
}