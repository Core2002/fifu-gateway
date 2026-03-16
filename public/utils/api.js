// ============================================
// utils/api.js - API 通信层
// 封装与后端的 HTTP 请求
// ============================================

import { getToken } from './token.js';

const API_BASE_URL = 'http://localhost:5000';

/**
 * 通用的 API 调用函数
 * @param {string} endpoint - API 端点路径
 * @param {string} method - HTTP 方法
 * @param {Object|null} data - 请求体数据
 * @param {boolean} useAuth - 是否需要认证
 * @returns {Promise<Object>} - API 响应数据
 */
export async function callApi(endpoint, method = 'POST', data = null, useAuth = false) {
    const url = `${API_BASE_URL}${endpoint}`;
    const options = {
        method: method,
        headers: {
            'Content-Type': 'application/json',
        }
    };

    // 如果需要认证，添加 Authorization header
    if (useAuth) {
        const token = getToken();
        if (token) {
            options.headers['Authorization'] = `Bearer ${token}`;
        }
    }

    if (data) {
        options.body = JSON.stringify(data);
    }

    try {
        const response = await fetch(url, options);
        const result = await response.json();

        if (!response.ok) {
            throw new Error(result.error || `HTTP ${response.status}: ${response.statusText}`);
        }

        return result;
    } catch (error) {
        console.error(`API 调用失败 [${endpoint}]:`, error);
        throw error;
    }
}

export { API_BASE_URL };
