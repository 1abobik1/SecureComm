import {LocalStorage} from 'ttl-localstorage';

export function setKs(value: CryptoKey, ttlSeconds: number) {
    LocalStorage.put('ks', value, ttlSeconds);
}


export function getKs() {
    return LocalStorage.get('ks');
}


export function removeKs() {
    LocalStorage.removeKey('ks');
}