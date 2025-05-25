import {makeAutoObservable} from "mobx";
import AuthService from "@/app/api/services/AuthServices";
import axios from 'axios';
import {AuthResponse} from "../models/response/AuthResponse";
import {AUTH_API_URL} from "@/app/api/http/urls";
import {decryptKsDataLogin, encryptAndStoreKey, importKeyFromBase64} from '@/app/api/utils/EncryptDecryptKey';
import {doHandshake} from "@/app/api/services/HandshakeService/HandshakeService";
import {removeKs, setKs, setKsLogin} from "@/app/api/utils/ksInStorage";


export default class Store {

    isAuth = false;
    isLoading = false;
    platform: 'web' = 'web';

    constructor() {
        makeAutoObservable(this);
    }


    setAuth(bool: boolean) {
        this.isAuth = bool;
    }

    setLoading(bool: boolean) {
        this.isLoading = bool;
    }


    async login(email: string, password: string) {
            try {
                const response = await AuthService.login(email, password, this.platform);
                const encryptedKs = response.data.ks;
                localStorage.setItem('encryptedFileKey', JSON.stringify(encryptedKs));
                const decryptedKs = await decryptKsDataLogin(encryptedKs, password);
                setKsLogin(decryptedKs, 60 * 8);
                localStorage.setItem('token', response.data.access_token);
                this.setAuth(true);
            } catch (e) {

            }
        }

    async signup(email: string, password: string) {
        try {
            const response = await AuthService.signup(email, password, this.platform);
            localStorage.setItem('token', response.data.access_token);
            const ks64 = await doHandshake();
            if (ks64!==undefined){
                const cryptoKey = await importKeyFromBase64(ks64);
                await encryptAndStoreKey(cryptoKey, password);
                setKs(cryptoKey, 60 * 8)
                this.setAuth(true);
            }

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


    async logout() {
        try {
            await AuthService.logout(this.platform);
            localStorage.removeItem('token');
            removeKs();
            localStorage.removeItem('encryptedFileKey')
            localStorage.removeItem('lastPath');
            this.setAuth(false);
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
                console.log('Не авторизован.');
            } else {
                console.log('Ошибка авторизации:', e.response?.data?.message);
            }
            return Promise.reject(e);
        } finally {
            this.setLoading(false);
        }
    }

}