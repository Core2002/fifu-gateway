// ============================================
// webauthn-core.js - WebAuthn 核心业务逻辑
// 处理注册和登录的业务流程，与 DOM 解耦
// ============================================

import { base64urlToBuffer, bufferToBase64url } from './utils/base64.js';
import { callApi } from './utils/api.js';

/**
 * WebAuthn 核心类
 * 提供注册和登录的核心功能，不直接操作 DOM
 */
export class WebAuthnCore {
    constructor() {
        this.API_BASE_URL = 'http://localhost:5000';
    }

    /**
     * 检查 WebAuthn 支持情况
     * @returns {Object} - 检查结果对象
     */
    checkWebAuthnSupport() {
        // 基础检查：PublicKeyCredential API 是否存在
        if (!window.PublicKeyCredential) {
            return {
                supported: false,
                reason: '浏览器不支持 WebAuthn API'
            };
        }
        
        // 检查是否在安全上下文中
        const isSecureContext = window.isSecureContext || 
                                location.protocol === 'https:' || 
                                location.hostname === 'localhost' || 
                                location.hostname === '127.0.0.1';
        
        if (!isSecureContext) {
            return {
                supported: false,
                reason: 'WebAuthn 需要 HTTPS 或 localhost 环境'
            };
        }
        
        // 检查平台认证器（可选）
        let hasPlatformAuthenticator = true;
        if (typeof PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable === 'function') {
            hasPlatformAuthenticator = true;
        }
        
        console.log('✅ WebAuthn 支持检测通过');
        console.log('   - PublicKeyCredential: 支持');
        console.log('   - 安全上下文:', isSecureContext ? '是' : '否');
        console.log('   - 平台认证器:', hasPlatformAuthenticator ? '可能支持' : '未知');
        
        return {
            supported: true,
            hasPlatformAuthenticator
        };
    }

    /**
     * 执行注册流程
     * @param {string} username - 用户名
     * @param {Function} onProgress - 进度回调函数
     * @returns {Promise<Object>} - 注册结果
     */
    async register(username, onProgress = () => {}) {
        try {
            // 动态检查平台认证器可用性
            if (typeof PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable === 'function') {
                try {
                    const available = await PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable();
                    if (!available) {
                        console.warn('⚠️ 此设备可能没有可用的生物识别传感器（如指纹、面部识别）');
                    }
                } catch (e) {
                    console.warn('无法检查平台认证器可用性:', e);
                }
            }

            onProgress('正在请求注册挑战...', false);

            // 从后端获取注册选项
            const options = await callApi('/webauthn/register/start', 'POST', { username });
            console.log('注册选项:', options);

            // 验证选项有效性
            this._validateOptions(options);

            // 转换为 WebAuthn API 需要的格式
            const publicKeyOptions = this._convertRegistrationOptions(options);

            console.log('转换后的 publicKey 选项:', publicKeyOptions);

            // 调用浏览器原生 WebAuthn API
            onProgress('请在您的设备上验证（如触摸指纹传感器或插入安全密钥）...');
            
            const credential = await navigator.credentials.create({ publicKey: publicKeyOptions });
            
            console.log('浏览器注册响应:', credential);

            if (!credential) {
                throw new Error('浏览器未返回凭证');
            }

            // 转换为可发送给后端的格式
            const finishData = this._convertAttestationResponse(credential, username);

            console.log('发送给后端的数据:', finishData);
            
            onProgress('正在验证注册信息...');
            const verificationResult = await callApi('/webauthn/register/finish', 'POST', finishData);

            if (verificationResult && verificationResult.status === 'registered') {
                onProgress(`✅ 注册成功！用户 "${username}" 的通行密钥已保存。`, false);
                return { success: true, status: 'registered', username };
            } else {
                throw new Error(verificationResult?.error || '服务器验证注册失败');
            }
        } catch (error) {
            console.error('注册流程错误:', error);
            const errorMessage = this._handleRegistrationError(error);
            onProgress(errorMessage, true);
            return { success: false, error };
        }
    }

    /**
     * 执行登录流程
     * @param {string} username - 用户名
     * @param {Function} onProgress - 进度回调函数
     * @returns {Promise<Object>} - 登录结果
     */
    async login(username, onProgress = () => {}) {
        try {
            onProgress('正在请求登录挑战...', false);

            // 从后端获取认证选项
            const options = await callApi('/webauthn/login/start', 'POST', { username });
            console.log('登录选项:', options);

            // 验证选项有效性
            this._validateOptions(options);

            // 检查是否有错误
            if (options.error) {
                throw new Error(options.error);
            }

            // 转换为 WebAuthn API 需要的格式
            const publicKeyOptions = this._convertAssertionOptions(options);

            console.log('转换后的 publicKey 选项:', publicKeyOptions);

            // 调用浏览器原生 WebAuthn API
            onProgress('请在您的设备上验证身份...');
            
            const assertion = await navigator.credentials.get({ publicKey: publicKeyOptions });
            
            console.log('浏览器认证响应:', assertion);

            if (!assertion) {
                throw new Error('浏览器未返回凭证');
            }

            // 将认证响应发送到后端验证
            const finishData = this._convertAssertionResponse(assertion, username);

            console.log('发送给后端的数据:', finishData);
            
            onProgress('正在验证登录信息...');
            const verificationResult = await callApi('/webauthn/login/finish', 'POST', finishData);

            if (verificationResult && verificationResult.status === 'login ok') {
                onProgress(`✅ 登录成功！欢迎回来，${username}。`, false);
                return { 
                    success: true, 
                    status: 'login ok', 
                    data: verificationResult.data,
                    username 
                };
            } else {
                throw new Error(verificationResult?.error || '服务器验证登录失败');
            }
        } catch (error) {
            console.error('登录流程错误:', error);
            const errorMessage = this._handleLoginError(error);
            onProgress(errorMessage, true);
            return { success: false, error };
        }
    }

    /**
     * 验证选项对象的有效性
     * @private
     */
    _validateOptions(options) {
        if (!options || typeof options !== 'object') {
            throw new Error('服务器返回的选项无效');
        }
        if (!options.challenge) {
            throw new Error('服务器返回的 challenge 为空');
        }
    }

    /**
     * 转换注册选项为 WebAuthn API 格式
     * @private
     */
    _convertRegistrationOptions(options) {
        const publicKeyOptions = {
            rp: options.rp,
            user: {
                id: base64urlToBuffer(options.user.id),
                name: options.user.name,
                displayName: options.user.displayName
            },
            challenge: base64urlToBuffer(options.challenge),
            pubKeyCredParams: options.pubKeyCredParams,
            timeout: options.timeout,
            attestation: options.attestation || 'none'
        };

        // 添加可选字段
        if (options.excludeCredentials) {
            publicKeyOptions.excludeCredentials = options.excludeCredentials.map(cred => ({
                id: base64urlToBuffer(cred.id),
                type: cred.type || 'public-key',
                transports: cred.transports || []
            }));
        }

        if (options.authenticatorSelection) {
            publicKeyOptions.authenticatorSelection = options.authenticatorSelection;
        }

        return publicKeyOptions;
    }

    /**
     * 转换登录选项为 WebAuthn API 格式
     * @private
     */
    _convertAssertionOptions(options) {
        const publicKeyOptions = {
            challenge: base64urlToBuffer(options.challenge),
            rpId: options.rpId,
            timeout: options.timeout,
            userVerification: options.userVerification || 'preferred'
        };

        if (options.allowCredentials) {
            publicKeyOptions.allowCredentials = options.allowCredentials.map(cred => ({
                id: base64urlToBuffer(cred.id),
                type: cred.type || 'public-key',
                transports: cred.transports || []
            }));
        }

        return publicKeyOptions;
    }

    /**
     * 转换注册响应为后端需要的格式
     * @private
     */
    _convertAttestationResponse(credential, username) {
        const attestationResponse = credential.response;
        const finishData = {
            username: username,
            id: credential.id,
            rawId: bufferToBase64url(credential.rawId),
            type: credential.type,
            response: {
                clientDataJSON: bufferToBase64url(attestationResponse.clientDataJSON),
                attestationObject: bufferToBase64url(attestationResponse.attestationObject),
            },
            clientExtensionResults: credential.getClientExtensionResults()
        };

        // 添加可选字段
        if (attestationResponse.authenticatorData) {
            finishData.response.authenticatorData = bufferToBase64url(attestationResponse.authenticatorData);
        }
        if (attestationResponse.publicKey) {
            finishData.response.publicKey = bufferToBase64url(attestationResponse.publicKey);
        }
        if (attestationResponse.publicKeyAlgorithm !== undefined) {
            finishData.response.publicKeyAlgorithm = attestationResponse.publicKeyAlgorithm;
        }
        if (attestationResponse.transports) {
            finishData.response.transports = attestationResponse.transports;
        }

        return finishData;
    }

    /**
     * 转换登录响应为后端需要的格式
     * @private
     */
    _convertAssertionResponse(assertion, username) {
        const authenticatorData = assertion.response.authenticatorData;
        const clientDataJSON = assertion.response.clientDataJSON;
        const signature = assertion.response.signature;

        const finishData = {
            username: username,
            id: assertion.id,
            rawId: bufferToBase64url(assertion.rawId),
            type: assertion.type,
            response: {
                authenticatorData: bufferToBase64url(authenticatorData),
                clientDataJSON: bufferToBase64url(clientDataJSON),
                signature: bufferToBase64url(signature),
                userHandle: assertion.response.userHandle ? bufferToBase64url(assertion.response.userHandle) : undefined
            },
            clientExtensionResults: assertion.getClientExtensionResults()
        };

        return finishData;
    }

    /**
     * 处理注册错误
     * @private
     */
    _handleRegistrationError(error) {
        if (error.name === 'InvalidStateError') {
            return '⚠️ 此设备可能已为该账户注册过通行密钥，或者该密钥已存在。';
        } else if (error.name === 'NotAllowedError') {
            return '❌ 操作被用户取消或超时。';
        } else if (error.name === 'SecurityError') {
            return '❌ 安全错误：请确保使用 HTTPS 或 localhost。';
        } else {
            return `注册失败：${error.message}`;
        }
    }

    /**
     * 处理登录错误
     * @private
     */
    _handleLoginError(error) {
        if (error.name === 'NotAllowedError') {
            return '❌ 操作被用户取消或超时。';
        } else if (error.name === 'SecurityError') {
            return '❌ 安全错误：请确保使用 HTTPS 或 localhost。';
        } else {
            return `登录失败：${error.message}`;
        }
    }
}

// 导出单例实例
export const webAuthn = new WebAuthnCore();
