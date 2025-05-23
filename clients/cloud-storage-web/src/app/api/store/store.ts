import {makeAutoObservable} from "mobx";
import AuthService from "@/app/api/services/AuthServices";
import axios from 'axios';
import {AuthResponse} from "../models/response/AuthResponse";
import {AUTH_API_URL} from "@/app/api/http/urls";
import {
    decryptKeyWithPassword,
    encryptKeyWithPassword,
    generateFileEncryptionKey
} from '@/app/api/utils/EncryptDecryptKey';
import {clearKey, getStoredKey, storeKey} from "@/app/api/utils/KeyStorage";
import {getEncryptedKeyFromToken} from "@/app/api/utils/KeyFromToken";


export default class Store {

    isAuth = false;
    isLoading = false;
    hasCryptoKey = false;

    constructor() {
        makeAutoObservable(this);
        this.initializeKey();
    }

    async initializeKey() {
        try {
            await getStoredKey();
            this.hasCryptoKey = true;
        } catch {
            this.hasCryptoKey = false;
        }
    }


    setAuth(bool: boolean) {
        this.isAuth = bool;
    }

    setLoading(bool: boolean) {
        this.isLoading = bool;
    }


    async login(email: string, password: string, platform: string) {
            try {
                const response = await AuthService.login(email, password, platform);
                localStorage.setItem('token', response.data.access_token);
                this.setAuth(true);

            } catch (e) {
                console.log(e);
            }
        }

    async signup(email: string, password: string, platform: string) {
        try {
            const fileKey = await generateFileEncryptionKey();
            storeKey(fileKey);
            this.hasCryptoKey = true;
            const user_key = await encryptKeyWithPassword(fileKey, password);
            const response = await AuthService.signup(email, password, user_key, platform);
            localStorage.setItem('token', response.data.access_token);
            this.setAuth(true);


        } catch (e) {
            console.log(e.response?.data?.message);

            if (e.response?.status === 409) {
                alert('Аккаунт на эту почту уже зарегистрирован.');
            } else {
                console.error(e);
                alert('Произошла ошибка при регистрации');
            }
        }
    }

    async decryptAndStoreKey(password: string): Promise<boolean> {
        try {
            const encryptedKey = getEncryptedKeyFromToken();
            await decryptKeyWithPassword(encryptedKey, password);
            this.hasCryptoKey = true;
            return true;
        } catch (err) {
            console.error("Decryption error:", err);
            return false;
        }
    }



    async logout() {
        try {
            await AuthService.logout();
            localStorage.removeItem('token');
            localStorage.removeItem('lastPath');
            clearKey();
            this.setAuth(false);
            this.hasCryptoKey = false;
        } catch (e) {
            console.log(e.response?.data?.message);
        }
    }

    async checkAuth() {
        this.setLoading(true);
        try {
            const response = await axios.post<AuthResponse>(
                `${AUTH_API_URL}/token/update`,
                {},
                {
                    withCredentials: true
                }
            );
            localStorage.setItem('token', response.data.access_token);
            this.setAuth(true);
            return Promise.resolve();

        } catch (e) {
            if (e.response?.status === 401) {
                this.setAuth(false);
                localStorage.removeItem('token');
                console.log('Не удалось обновить токен: пользователь не авторизован.');
            } else {
                console.log('Ошибка авторизации:', e.response?.data?.message);
            }
            return Promise.reject(e);
        } finally {
            this.setLoading(false);
        }
    }

}