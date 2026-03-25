// ============================================
// utils/token.js - Token 和用户信息管理
// ============================================

/**
 * 保存访问令牌到 localStorage
 * @param {string} token - JWT 或其他访问令牌
 */
export function saveToken(token) {
    localStorage.setItem('access_token', token);
}

/**
 * 从 localStorage 获取访问令牌
 * @returns {string|null} - 访问令牌或 null
 */
export function getToken() {
    return localStorage.getItem('access_token');
}

/**
 * 从 localStorage 移除访问令牌
 */
export function removeToken() {
    localStorage.removeItem('access_token');
}

/**
 * 保存用户信息到 localStorage
 * @param {Object} userInfo - 用户信息对象
 */
export function saveUserInfo(userInfo) {
    localStorage.setItem('user_info', JSON.stringify(userInfo));
}

/**
 * 从 localStorage 获取用户信息
 * @returns {Object|null} - 用户信息对象或 null
 */
export function getUserInfo() {
    const info = localStorage.getItem('user_info');
    return info ? JSON.parse(info) : null;
}

/**
 * 从 localStorage 移除用户信息
 */
export function removeUserInfo() {
    localStorage.removeItem('user_info');
}
