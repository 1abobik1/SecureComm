export function getEncryptedKeyFromToken(): string {
    const token = localStorage.getItem('token');
    if (!token) throw new Error("Не авторизован");

    const payload = JSON.parse(atob(token.split('.')[1]));
    return payload.user_key;
}
