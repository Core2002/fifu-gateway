// ============================================
// webauthn.js - WebAuthn 前端核心逻辑
// 使用原生 WebAuthn API
// ============================================

// 配置后端 API 基础地址
const API_BASE_URL = 'http://127.0.0.1:5000';

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
async function callApi(endpoint, method = 'POST', data = null) {
    const url = `${API_BASE_URL}${endpoint}`;
    const options = {
        method: method,
        headers: {
            'Content-Type': 'application/json',
        }
    };

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
            showMessage(`✅ 登录成功！欢迎回来，${username}。`);
            document.getElementById('loginUsername').value = '';
            updateLoginStatus(username);
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
    statusDiv.innerHTML = `当前用户: <strong>${username}</strong> | <a href="#" onclick="handleLogout(); return false;" style="color: #2563eb; text-decoration: underline;">退出</a>`;
    statusDiv.style.display = 'block';
}

function handleLogout() {
    const statusDiv = document.getElementById('loginStatus');
    if (statusDiv) statusDiv.style.display = 'none';
    showMessage('已退出登录。');
}

// 4. 页面加载时检查浏览器支持
document.addEventListener('DOMContentLoaded', () => {
    console.log('页面加载完成，检查 WebAuthn 支持...');
    
    if (!window.PublicKeyCredential) {
        showMessage('⚠️ 您的浏览器不支持 WebAuthn。请使用 Chrome 67+、Edge 18+、Firefox 60+ 或 Safari 13+。', true);
        document.querySelectorAll('button').forEach(btn => btn.disabled = true);
        return;
    }
    
    console.log('✅ 浏览器支持 WebAuthn');

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