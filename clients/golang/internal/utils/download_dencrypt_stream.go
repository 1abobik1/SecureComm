package utils

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/hmac"
    "crypto/sha256"
    "errors"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "time"
)

// DownloadAndDecryptStream скачивает «encrypted_blob» по presignedURL,
// проверяет HMAC, расшифровывает AES-CBC блок за блоком, снимает PKCS#7-padding
// у последнего блока, проверяет timestamp и сохраняет чистый файл в outPath.
//
// Формат потока на сервере:
//   [ timestamp (8 байт) | nonce (16 байт) | IV (16 байт) | ciphertext (N*16 байт) | tag (32 байта) ]
//
// 1) timestamp и nonce передаются «в открыть»;
// 2) далее идёт IV; затем каждый блок ciphertext дешифруется CBC;
// 3) Нельзя записывать последний блок ciphertext в файл до тех пор, пока не проверили padding.
//    Поэтому мы храним в памяти ровно по одному блоку «prevPlainBlock»;
// 4) Как только читаем новый блок ciphertext, первое, что делаем — пишем «prevPlainBlock» в файл;
//    а затем перескакиваем: что было «current» → становится «prevPlainBlock», читаем след. ciphertext;
// 5) После того как считали и дешифровали все N блоков ciphertext, у нас остаётся один последний
//    расшифрованный блок в prevPlainBlock. Именно в нём лежит PKCS#7-padding. Снимаем его и пишем
//    «остаток блока без padding» в файл.
// 6) Читаем tag (32 байта), сравниваем с HMAC(iv∥ciphertext). Если не совпадает — возвращаем ошибку.
//
// В итоге на диске остаётся только `outPath`, без каких-либо .tmp_decrypt.
func DownloadAndDecryptStream(presignedURL, outPath string, kEnc, kMac []byte) error {
    time.Sleep(time.Minute)
    // 0) Создать родительские папки, если их нет
    if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
        return fmt.Errorf("не удалось создать папку для %q: %w", outPath, err)
    }

    // 1) HTTP GET
    resp, err := http.Get(presignedURL)
    if err != nil {
        return fmt.Errorf("ошибка при GET: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("download вернул HTTP %d", resp.StatusCode)
    }

    // 2) Считать Content-Length, чтобы знать, сколько байт ciphertext ожидается
    clHeader := resp.Header.Get("Content-Length")
    if clHeader == "" {
        return errors.New("Content-Length отсутствует, не могу работать в стриме")
    }
    contentLen, err := strconv.ParseInt(clHeader, 10, 64)
    if err != nil {
        return fmt.Errorf("невалидный Content-Length: %w", err)
    }
    // Минимальная возможная длина: 8 (timestamp) + 16 (nonce) + 16 (IV) + 16 (минимум 1 блок ciphertext) + 32 (tag) = 88 байт
    if contentLen < int64(8+16+aes.BlockSize+aes.BlockSize+sha256.Size) {
        return fmt.Errorf("слишком короткий зашифрованный blob: %d байт", contentLen)
    }

    // 3a) Сначала читаем 8 байт = timestamp
    tsBytes := make([]byte, 8)
    if _, err := io.ReadFull(resp.Body, tsBytes); err != nil {
        return fmt.Errorf("не удалось прочитать timestamp: %w", err)
    }
    // 3b) Затем 16 байт = nonce
    nonceBytes := make([]byte, 16)
    if _, err := io.ReadFull(resp.Body, nonceBytes); err != nil {
        return fmt.Errorf("не удалось прочитать nonce: %w", err)
    }
    // Проверяем timestamp (±30 секунд)
    tsVal := int64(0)
    for i := 0; i < 8; i++ {
        tsVal = (tsVal << 8) | int64(tsBytes[i])
    }
    nowMs := time.Now().UnixMilli()
    if diff := nowMs - tsVal; diff > 30_000 || diff < -30_000 {
        return fmt.Errorf("stale/future timestamp (diff=%d ms)", diff)
    }
    // (nonce можно дополнительно проверить, если нужно хранить все «использованные», но обычно не обязательно)

    // 4) Читаем 16 байт = IV
    iv := make([]byte, aes.BlockSize)
    if _, err := io.ReadFull(resp.Body, iv); err != nil {
        return fmt.Errorf("не удалось прочитать IV: %w", err)
    }

    // 5) Длина ciphertext = contentLen − (8+16+16+32)
    cipherLen := contentLen - int64(8+16+aes.BlockSize+sha256.Size)
    if cipherLen < aes.BlockSize {
        return fmt.Errorf("недостаточно байт для ciphertext: %d", cipherLen)
    }
    if cipherLen%aes.BlockSize != 0 {
        return fmt.Errorf("ciphertext длина не кратна AES-блоку: %d", cipherLen)
    }

    // 6) Настраиваем HMAC: mac = HMAC_SHA256(iv || ciphertext)
    macHasher := hmac.New(sha256.New, kMac)
    macHasher.Write(iv)

    // 7) Инициализируем AES-CBC-дешифратор
    blockCipher, err := aes.NewCipher(kEnc)
    if err != nil {
        return fmt.Errorf("AES-инициализация не удалась: %w", err)
    }
    cbc := cipher.NewCBCDecrypter(blockCipher, iv)

    // 8) Открываем итоговый файл на запись
    outFile, err := os.Create(outPath)
    if err != nil {
        return fmt.Errorf("не удалось создать выходной файл %q: %w", outPath, err)
    }
    defer outFile.Close()

    // 9) Читаем ciphertext по одному блоку (16 байт) за раз.
    //    Чтобы не хранить весь файл в памяти, но при этом правильно убрать PKCS#7
    //    для последнего блока, мы будем:
    //      - читать _первый_ блок ciphertext и не писать его сразу,
    //      - на каждой итерации читать _следующий_ блок ciphertext, дешифровать _предыдущий_ и писать его,
    //      - в конце, когда следующий блока нет, дешифровать последний и снимать padding.

    buf := make([]byte, aes.BlockSize)
    // Сначала — читаем ровно один блок ciphertext
    if _, err := io.ReadFull(resp.Body, buf); err != nil {
        return fmt.Errorf("не удалось прочитать первый блок ciphertext: %w", err)
    }
    macHasher.Write(buf) // учитываем его в HMAC
    prevCipherBlock := make([]byte, aes.BlockSize)
    copy(prevCipherBlock, buf)

    // Количество блоков ciphertext
    blocksCount := int(cipherLen / int64(aes.BlockSize))

    // Декодим вперед (blocksCount − 1) блоков: каждый раз
    // читаем новый блок, HMAC, дешифруем prevCipherBlock и пишем в файл,
    // затем сдвигаем: prevCipherBlock = newCipherBlock
    for i := 1; i < blocksCount; i++ {
        if _, err := io.ReadFull(resp.Body, buf); err != nil {
            return fmt.Errorf("ошибка чтения ciphertext, блок %d: %w", i, err)
        }
        macHasher.Write(buf) // обновляем HMAC(iv ∥ все прочитанные ciphertext)

        // Дешифруем PREV блок:
        plainPrev := make([]byte, aes.BlockSize)
        cbc.CryptBlocks(plainPrev, prevCipherBlock)
        // Пишем расшифрованные байты (они без padding, потому что не последний блок)
        if _, err := outFile.Write(plainPrev); err != nil {
            return fmt.Errorf("ошибка записи decrypted блока %d: %w", i-1, err)
        }

        // Теперь сохраняем новый как prev
        copy(prevCipherBlock, buf)
    }

    // 10) После цикла prevCipherBlock содержит _последний_ блок ciphertext.
    //     Дешифруем его. В нём лежит PKCS#7-padding.
    lastPlain := make([]byte, aes.BlockSize)
    cbc.CryptBlocks(lastPlain, prevCipherBlock)

    // 11) Снимаем PKCS#7 padding
    padLen := int(lastPlain[aes.BlockSize-1])
    if padLen < 1 || padLen > aes.BlockSize {
        return fmt.Errorf("некорректная длина padding-байтов: %d", padLen)
    }
    // Проверяем, что все padding-байты одинаковы
    for i := 0; i < padLen; i++ {
        if int(lastPlain[aes.BlockSize-1-i]) != padLen {
            return fmt.Errorf("паддинг испорчен, byte[%d]=%d", aes.BlockSize-1-i, lastPlain[aes.BlockSize-1-i])
        }
    }
    // Записываем все байты из последнего блока, кроме padding
    if _, err := outFile.Write(lastPlain[:aes.BlockSize-padLen]); err != nil {
        return fmt.Errorf("ошибка записи последнего decrypted блока без padding: %w", err)
    }

    // 12) Читаем tag (32 байта) и сверяем с HMAC
    tag := make([]byte, sha256.Size)
    if _, err := io.ReadFull(resp.Body, tag); err != nil {
        return fmt.Errorf("не удалось прочитать HMAC-tag: %w", err)
    }
    computed := macHasher.Sum(nil)
    if !hmac.Equal(computed, tag) {
        return fmt.Errorf("HMAC mismatch: данные повреждены или неверный KMac")
    }

    // Всё успешно, файл outPath готов
    return nil
}
