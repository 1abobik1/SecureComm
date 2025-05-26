'use client';

import React, {useContext, useRef, useState} from 'react';
import {ArrowUpOnSquareIcon} from '@heroicons/react/24/outline';
import {observer} from 'mobx-react-lite';
import {Context} from '@/app/api/store/context';
import {useUsageRefresh} from './UsageRefreshContext';
import {decryptStoredKey} from "@/app/api/utils/EncryptDecryptKey";
import PasswordModal, {PasswordModalRef} from "@/app/ui/PasswordModal";
import {getKs} from "@/app/api/utils/ksInStorage";
import {streamUploadEncryptedFile} from "@/app/api/utils/CryptoHelper";


const FileUploader = observer(() => {
    const inputRef = useRef<HTMLInputElement>(null);
    const modalRef = useRef<PasswordModalRef>(null);
    const store = useContext(Context);
    const [toastMessage, setToastMessage] = useState<string | null>(null);
    const [toastType, setToastType] = useState<'success' | 'error' | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [uploadProgress, setUploadProgress] = useState<number>(0);
    const {triggerRefresh} = useUsageRefresh();

    const handlePasswordSuccess = async (password: string): Promise<boolean> => {
        const encKs: {
            k_enc_iv: string,
            k_enc_data: string,
            k_mac_iv: string,
            k_mac_data: string
        } = JSON.parse(localStorage.getItem('encryptedFileKey') as string);

        try {
            const success = await decryptStoredKey(encKs, password, store);
            if (success) {
                setTimeout(() => inputRef.current?.click(), 100);
                return true;
            }
            return false;
        } catch (error) {
            console.error("Password error:", error);
            return false;
        }
    };

    const handleButtonClick = async () => {
        if (!store.isAuth) return;

        try {
            await store.initializeKey();
            if (store.hasCryptoKey) {
                inputRef.current?.click();
            } else {
                modalRef.current?.open();
            }
        } catch (err) {
            console.error("Key check error:", err);
            setToastMessage('Ошибка проверки ключа');
            setToastType('error');
        }
    };


    const handleFileChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
        const files = event.target.files;
        if (!files || files.length === 0) return;

        const MIN_SPINNER_DURATION = 2000;
        const startTime = Date.now();

        try {
            setIsLoading(true);
            setToastMessage('Загрузка файла...');
            setToastType(null);
            setUploadProgress(0);

            // Получаем ключи из хранилища
            const keys = await getKs();
            if (!keys) {
                modalRef.current?.open();
                setIsLoading(false);
                if (inputRef.current) inputRef.current.value = '';
                return;
            }
            const [kMac, kEnc] = keys;

            // Обрабатываем каждый файл
            for (const file of Array.from(files)) {
                await streamUploadEncryptedFile(file, 'http://localhost:8080/files/one/encrypted', "unknown", kEnc, kMac );
            }

            const elapsedTime = Date.now() - startTime;
            const remainingTime = Math.max(0, MIN_SPINNER_DURATION - elapsedTime);
            await new Promise(resolve => setTimeout(resolve, remainingTime));

            setToastMessage('Файл успешно загружен!');
            triggerRefresh();
            setToastType('success');
        } catch (error) {
            console.error(error);
            setToastMessage(error.message || 'Не удалось загрузить файл');
            setToastType('error');
        } finally {
            setIsLoading(false);
            setUploadProgress(0);
            if (inputRef.current) inputRef.current.value = '';
            setTimeout(() => {
                setToastMessage(null);
                setToastType(null);
            }, 3000);
        }
    };

    const LinkIcon = ArrowUpOnSquareIcon;

    return (
        <>
            <PasswordModal
                ref={modalRef}
                onSubmit={handlePasswordSuccess}
                title="Для загрузки файлов введите пароль"
                description="Сессия истекла. Для продолжения требуется ваш пароль."
            />

            <div className="max-w-md mx-auto text-center">
                <input
                    type="file"
                    multiple
                    ref={inputRef}
                    style={{display: 'none'}}
                    onChange={handleFileChange}
                    disabled={!store.isAuth || isLoading}
                />

                <button
                    onClick={handleButtonClick}
                    disabled={!store.isAuth || isLoading}
                    className="bg-blue-500 hover:bg-blue-600 text-white py-2 px-4 rounded disabled:opacity-50"
                >
                    <div className="flex items-center justify-center">
                        <LinkIcon className="w-6 text-white mr-1"/>
                        <span>{isLoading ? 'Загрузка...' : 'Загрузить файл'}</span>
                    </div>
                </button>

                {isLoading && (
                    <div className="mt-4">
                        <div className="w-full bg-gray-200 h-2 rounded">
                            <div
                                className="h-2 bg-blue-500 rounded transition-all duration-300"
                                style={{width: `${uploadProgress}%`}}
                            />
                        </div>
                        <div className="text-sm text-gray-700 mt-1">{uploadProgress}%</div>
                    </div>
                )}
            </div>

            {toastMessage && (
                <div className={`fixed bottom-4 right-4 w-80 px-4 py-3 rounded shadow-lg z-50 text-white ${
                    toastType === 'success' ? 'bg-green-500' :
                        toastType === 'error' ? 'bg-red-500' : 'bg-blue-500'
                }`}>
                    <div className="flex items-center space-x-3">
                        {isLoading && (
                            <div
                                className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                        )}
                        <div className="text-sm">{toastMessage}</div>
                    </div>
                </div>
            )}
        </>
    );
});

export default FileUploader;