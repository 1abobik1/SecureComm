import {loadDERPub, loadECDSAPriv} from './src/utils/loadKeys';
import {doFinalizeAPI, doInitAPI} from './src/http/http';
import {performance} from 'perf_hooks';

const config = {
    rsaPubPath: '../keys/client_rsa.pub',
    ecdsaPubPath: '../keys/client_ecdsa.pub',
    ecdsaPrivPath: '../keys/client_ecdsa.pem',
    initURL: 'http://localhost:8080/handshake/init',
    finURL: 'http://localhost:8080/handshake/finalize'
};

async function main() {
    try {
        // Загрузка ключей
        const rsaPubDER = loadDERPub(config.rsaPubPath);
        const ecdsaPubDER = loadDERPub(config.ecdsaPubPath);
        const ecdsaPriv = loadECDSAPriv(config.ecdsaPrivPath);


        // Init handshake
        const startInit = performance.now();
        const initResp = await doInitAPI(
            config.initURL,
            rsaPubDER,
            ecdsaPubDER,
            ecdsaPriv
        );
        console.log('Init response:', initResp);
        console.log(`Init handshake time: ${performance.now() - startInit}ms`);

        // Finalize handshake
        const startFin = performance.now();
        const finResp = await doFinalizeAPI(
            config.finURL,
            initResp,
            ecdsaPriv,
            config.rsaPubPath // Путь к RSA публичному ключу для шифрования
        );
        console.log('Finalize response:', finResp);
        console.log(`Finalize handshake time: ${performance.now() - startFin}ms`);

    } catch (err) {
        console.error('Error:', err instanceof Error ? err.message : String(err));
        process.exit(1);
    }
}

// Запуск приложения
main().catch(err => {
    console.error('Unhandled error:', err);
    process.exit(1);
});