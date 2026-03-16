// ============================================
// utils/base64.js - Base64URL 编码/解码工具
// ============================================

/**
 * 将 Base64URL 字符串转换为 ArrayBuffer
 * @param {string} base64url - Base64URL 编码的字符串
 * @returns {ArrayBuffer} - 转换后的 ArrayBuffer
 */
export function base64urlToBuffer(base64url) {
    if (!base64url) {
        throw new Error('base64url 数据为空');
    }
    // 将 base64url 转换为标准 base64
    let base64 = base64url.replace(/-/g, '+').replace(/_/g, '/');
    // 添加填充
    while (base64.length % 4) {
        base64 += '=';
    }
    // 解码为二进制
    const binary = atob(base64);
    const buffer = new ArrayBuffer(binary.length);
    const view = new Uint8Array(buffer);
    for (let i = 0; i < binary.length; i++) {
        view[i] = binary.charCodeAt(i);
    }
    return buffer;
}

/**
 * 将 ArrayBuffer 转换为 Base64URL 字符串
 * @param {ArrayBuffer} buffer - 要转换的 ArrayBuffer
 * @returns {string} - Base64URL 编码的字符串
 */
export function bufferToBase64url(buffer) {
    if (!buffer) return '';
    const binary = String.fromCharCode(...new Uint8Array(buffer));
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}
