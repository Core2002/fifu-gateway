// ============================================
// webauthn.js - WebAuthn 前端核心逻辑
// 使用原生 WebAuthn API
// ============================================

// 配置后端 API 基础地址
const API_BASE_URL = 'http://127.0.0.1:5000';

// Token 管理相关函数
function saveToken(token) {
    localStorage.setItem('access_token', token);
}

function getToken() {
    return localStorage.getItem('access_token');
}

function removeToken() {
    localStorage.removeItem('access_token');
}

function saveUserInfo(userInfo) {
    localStorage.setItem('user_info', JSON.stringify(userInfo));
}

function getUserInfo() {
    const info = localStorage.getItem('user_info');
    return info ? JSON.parse(info) : null;
}

function removeUserInfo() {
    localStorage.removeItem('user_info');
}

// 显示消息的工具函数
function showMessage(message, isError = false) {
    const messageDiv = document.getElementById('message');
    messageDiv.textContent = message;
    messageDiv.className = isError ? 'error' : 'success';
    messageDiv.style.display = 'block';
    console.log(`[${isError ? 'ERROR' : 'INFO'}] ${message}`);
}

// Base64URL 编码/解码工具
function base64urlToBuffer(base64url) {
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

function bufferToBase64url(buffer) {
    if (!buffer) return '';
    const binary = String.fromCharCode(...new Uint8Array(buffer));
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}

// 通用 API 调用函数
async function callApi(endpoint, method = 'POST', data = null, useAuth = false) {
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

// 1. 注册流程
async function handleRegister() {
    const username = document.getElementById('regUsername').value.trim();
    if (!username) {
        showMessage('请输入用户名用于注册', true);
        return;
    }

    // 动态检查平台认证器可用性（仅用于提供友好提示）
    if (typeof PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable === 'function') {
        try {
            const available = await PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable();
            if (!available) {
                console.warn('⚠️ 此设备可能没有可用的生物识别传感器（如指纹、面部识别）');
                // 继续执行，因为用户可能使用外部安全密钥
            }
        } catch (e) {
            console.warn('无法检查平台认证器可用性:', e);
        }
    }

    showMessage('正在请求注册挑战...', false);

    try {
        // 1.1 从后端获取注册选项
        const options = await callApi('/webauthn/register/start', 'POST', { username });
        console.log('注册选项:', options);

        // 检查 options 是否有效
        if (!options || typeof options !== 'object') {
            throw new Error('服务器返回的注册选项无效');
        }

        // 检查必要的字段
        if (!options.challenge) {
            throw new Error('服务器返回的 challenge 为空');
        }

        // 转换 challenge 和 id 为 ArrayBuffer
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

        console.log('转换后的 publicKey 选项:', publicKeyOptions);

        // 1.2 调用浏览器原生 WebAuthn API
        showMessage('请在您的设备上验证（如触摸指纹传感器或插入安全密钥）...');
        
        const credential = await navigator.credentials.create({ publicKey: publicKeyOptions });
        
        console.log('浏览器注册响应:', credential);

        if (!credential) {
            throw new Error('浏览器未返回凭证');
        }

        // 1.3 将凭证转换为可发送给后端的格式
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

        // 如果有这些可选字段也加上
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

        console.log('发送给后端的数据:', finishData);
        
        showMessage('正在验证注册信息...');
        const verificationResult = await callApi('/webauthn/register/finish', 'POST', finishData);

        if (verificationResult && verificationResult.status === 'registered') {
            showMessage(`✅ 注册成功！用户 "${username}" 的通行密钥已保存。`);
            document.getElementById('regUsername').value = '';
        } else {
            throw new Error(verificationResult?.error || '服务器验证注册失败');
        }
    } catch (error) {
        console.error('注册流程错误:', error);
        
        if (error.name === 'InvalidStateError') {
            showMessage('⚠️ 此设备可能已为该账户注册过通行密钥，或者该密钥已存在。', true);
        } else if (error.name === 'NotAllowedError') {
            showMessage('❌ 操作被用户取消或超时。', true);
        } else if (error.name === 'SecurityError') {
            showMessage('❌ 安全错误：请确保使用 HTTPS 或 localhost。', true);
        } else {
            showMessage(`注册失败: ${error.message}`, true);
        }
    }
}

// 2. 登录流程
async function handleLogin() {
    const username = document.getElementById('loginUsername').value.trim();
    if (!username) {
        showMessage('请输入用户名用于登录', true);
        return;
    }

    showMessage('正在请求登录挑战...', false);

    try {
        // 2.1 从后端获取认证选项
        const options = await callApi('/webauthn/login/start', 'POST', { username });
        console.log('登录选项:', options);

        if (!options || typeof options !== 'object') {
            throw new Error('服务器返回的登录选项无效');
        }

        if (options.error) {
            throw new Error(options.error);
        }

        if (!options.challenge) {
            throw new Error('服务器返回的 challenge 为空');
        }

        // 转换 challenge 为 ArrayBuffer
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

        console.log('转换后的 publicKey 选项:', publicKeyOptions);

        // 2.2 调用浏览器原生 WebAuthn API
        showMessage('请在您的设备上验证身份...');
        
        const assertion = await navigator.credentials.get({ publicKey: publicKeyOptions });
        
        console.log('浏览器认证响应:', assertion);

        if (!assertion) {
            throw new Error('浏览器未返回凭证');
        }

        // 2.3 将认证响应发送到后端验证
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

        console.log('发送给后端的数据:', finishData);
        
        showMessage('正在验证登录信息...');
        const verificationResult = await callApi('/webauthn/login/finish', 'POST', finishData);

        if (verificationResult && verificationResult.status === 'login ok') {
            // 保存 token 和用户信息
            const { access_token, user } = verificationResult.data;
            saveToken(access_token);
            saveUserInfo(user);
            
            showMessage(`✅ 登录成功！欢迎回来，${user.username}。`);
            document.getElementById('loginUsername').value = '';
            updateLoginStatus(user.username);
            checkAuthStatus(); // 检查认证状态并获取完整用户信息
        } else {
            throw new Error(verificationResult?.error || '服务器验证登录失败');
        }
    } catch (error) {
        console.error('登录流程错误:', error);
        
        if (error.name === 'NotAllowedError') {
            showMessage('❌ 操作被用户取消或超时。', true);
        } else if (error.name === 'SecurityError') {
            showMessage('❌ 安全错误：请确保使用 HTTPS 或 localhost。', true);
        } else {
            showMessage(`登录失败: ${error.message}`, true);
        }
    }
}

// 3. 辅助功能
function updateLoginStatus(username) {
    const statusDiv = document.getElementById('loginStatus');
    // 创建退出登录链接的 HTML，但不使用 inline onclick
    statusDiv.innerHTML = `当前用户：<strong>${username}</strong> | <a href="#" id="logoutLink" style="color: #2563eb; text-decoration: underline;">退出</a>`;
    statusDiv.style.display = 'block';
    
    // 为退出链接添加事件监听器
    const logoutLink = document.getElementById('logoutLink');
    if (logoutLink) {
        logoutLink.addEventListener('click', (e) => {
            e.preventDefault();
            handleLogout();
        });
    }
    
    // 更新用户信息卡片
    const userInfoDiv = document.getElementById('userInfo');
    if (userInfoDiv) {
        const user = getUserInfo();
        if (user) {
            userInfoDiv.innerHTML = `
                <p><strong>ID:</strong> ${user.id}</p>
                <p><strong>用户名:</strong> ${user.username}</p>
                <p><strong>角色:</strong> ${user.role}</p>
            `;
        }
    }
}

function handleLogout() {
    removeToken();
    removeUserInfo();
    const statusDiv = document.getElementById('loginStatus');
    if (statusDiv) statusDiv.style.display = 'none';
    showMessage('已退出登录。');
    // 隐藏受保护的内容
    hideProtectedContent();
}

// 检查认证状态
async function checkAuthStatus() {
    const token = getToken();
    if (!token) {
        hideProtectedContent();
        return;
    }
    
    try {
        const result = await callApi('/profile', 'GET', null, true);
        if (result && result.user) {
            updateLoginStatus(result.user.username);
            showProtectedContent();
        } else {
            removeToken();
            removeUserInfo();
            hideProtectedContent();
        }
    } catch (error) {
        console.error('认证检查失败:', error);
        removeToken();
        removeUserInfo();
        hideProtectedContent();
    }
}

// 显示受保护的内容
function showProtectedContent() {
    const protectedElements = document.querySelectorAll('.protected-content');
    protectedElements.forEach(el => el.style.display = 'block');
}

// 隐藏受保护的内容
function hideProtectedContent() {
    const protectedElements = document.querySelectorAll('.protected-content');
    protectedElements.forEach(el => el.style.display = 'none');
}

// 4. 页面加载时检查浏览器支持
document.addEventListener('DOMContentLoaded', () => {
    console.log('页面加载完成，检查 WebAuthn 支持...');
    
    // 更完善的 WebAuthn 支持检测
    function checkWebAuthnSupport() {
        // 1. 基础检查：PublicKeyCredential API 是否存在
        if (!window.PublicKeyCredential) {
            return {
                supported: false,
                reason: '浏览器不支持 WebAuthn API'
            };
        }
        
        // 2. 检查是否在安全上下文中 (HTTPS 或 localhost)
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
        
        // 3. 检查是否支持平台认证器（可选，用于提供更友好的提示）
        let hasPlatformAuthenticator = true; // 默认为 true，避免误报
        if (typeof PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable === 'function') {
            // 这是一个异步检查，我们只做同步检测
            hasPlatformAuthenticator = true; // 假设支持，实际使用时会动态检测
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
    
    const webauthnCheck = checkWebAuthnSupport();
    
    if (!webauthnCheck.supported) {
        showMessage(`⚠️ ${webauthnCheck.reason}。请使用最新版本的 Chrome、Safari、Edge 或 Firefox，并确保使用 HTTPS 或 localhost 访问。`, true);
        document.querySelectorAll('button').forEach(btn => btn.disabled = true);
        return;
    }
    
    console.log('✅ 浏览器支持 WebAuthn');
    
    // 页面加载时检查认证状态
    checkAuthStatus();

    // 绑定事件监听器
    const registerBtn = document.getElementById('registerBtn');
    const loginBtn = document.getElementById('loginBtn');

    if (registerBtn) {
        registerBtn.addEventListener('click', handleRegister);
    }
    if (loginBtn) {
        loginBtn.addEventListener('click', handleLogin);
    }

    // 允许按回车键触发
    ['regUsername', 'loginUsername'].forEach(id => {
        const input = document.getElementById(id);
        if (input) {
            input.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    if (id === 'regUsername') handleRegister();
                    if (id === 'loginUsername') handleLogin();
                }
            });
        }
    });
});