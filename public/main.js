// ============================================
// main.js - WebAuthn 应用主入口
// 整合所有模块，处理用户交互和界面更新
// ============================================

import { showMessage, updateLoginStatus, showProtectedContent, hideProtectedContent, clearInput, getInputValue } from './utils/ui.js';
import { saveToken, saveUserInfo, removeToken, removeUserInfo, getToken } from './utils/token.js';
import { callApi } from './utils/api.js';
import { webAuthn } from './webauthn-core.js';

/**
 * 处理注册按钮点击
 */
async function handleRegister() {
    const username = getInputValue('regUsername');
    if (!username) {
        showMessage('请输入用户名用于注册', true);
        return;
    }

    // 使用 WebAuthn 核心类执行注册
    const result = await webAuthn.register(username, (message, isError) => {
        showMessage(message, isError);
    });

    if (result.success) {
        clearInput('regUsername');
    }
}

/**
 * 处理登录按钮点击
 */
async function handleLogin() {
    const username = getInputValue('loginUsername');
    if (!username) {
        showMessage('请输入用户名用于登录', true);
        return;
    }

    // 使用 WebAuthn 核心类执行登录
    const result = await webAuthn.login(username, async (message, isError) => {
        showMessage(message, isError);
    });

    // 如果登录成功，保存 token 和用户信息并更新 UI
    if (result.success && result.data) {
        const { access_token, user } = result.data;
        saveToken(access_token);
        saveUserInfo(user);
        window.appUserInfo = { ...user, access_token };
        updateLoginStatus(user.username);
        showProtectedContent();
        clearInput('loginUsername');
        
        // 触发登录成功事件
        window.dispatchEvent(new CustomEvent('app:login', { 
            detail: { user, token: access_token } 
        }));
        
        // 检查认证状态并获取完整用户信息
        await checkAuthStatus();
    }
}

/**
 * 处理退出登录
 */
function handleLogout() {
    removeToken();
    removeUserInfo();
    window.appUserInfo = null;
    
    const statusDiv = document.getElementById('loginStatus');
    if (statusDiv) {
        statusDiv.style.display = 'none';
    }
    
    showMessage('已退出登录。');
    hideProtectedContent();
    
    // 触发登出事件
    window.dispatchEvent(new CustomEvent('app:logout'));
}

/**
 * 检查认证状态
 */
async function checkAuthStatus() {
    const token = getToken();
    if (!token) {
        hideProtectedContent();
        return;
    }
    
    try {
        const result = await callApi('/profile', 'GET', null, true);
        if (result && result.user) {
            window.appUserInfo = result.user;
            updateLoginStatus(result.user.username);
            showProtectedContent();
        } else {
            removeToken();
            removeUserInfo();
            window.appUserInfo = null;
            hideProtectedContent();
        }
    } catch (error) {
        console.error('认证检查失败:', error);
        removeToken();
        removeUserInfo();
        window.appUserInfo = null;
        hideProtectedContent();
    }
}

/**
 * 初始化应用
 */
function initApp() {
    console.log('初始化 WebAuthn 应用...');
    
    // 1. 检查 WebAuthn 支持
    const webAuthnCheck = webAuthn.checkWebAuthnSupport();
    
    if (!webAuthnCheck.supported) {
        showMessage(`⚠️ ${webAuthnCheck.reason}。请使用最新版本的 Chrome、Safari、Edge 或 Firefox，并确保使用 HTTPS 或 localhost 访问。`, true);
        document.querySelectorAll('button').forEach(btn => btn.disabled = true);
        return;
    }
    
    console.log('✅ 浏览器支持 WebAuthn');
    
    // 2. 检查认证状态
    checkAuthStatus();

    // 3. 绑定事件监听器
    const registerBtn = document.getElementById('registerBtn');
    const loginBtn = document.getElementById('loginBtn');

    if (registerBtn) {
        registerBtn.addEventListener('click', handleRegister);
    }
    
    if (loginBtn) {
        loginBtn.addEventListener('click', handleLogin);
    }

    // 4. 允许按回车键触发
    ['regUsername', 'loginUsername'].forEach(id => {
        const input = document.getElementById(id);
        if (input) {
            input.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    if (id === 'regUsername') {
                        handleRegister();
                    }
                    if (id === 'loginUsername') {
                        handleLogin();
                    }
                }
            });
        }
    });

    // 5. 监听全局登出事件
    window.addEventListener('app:logout', () => {
        console.log('收到登出事件');
    });

    // 6. 将 handleLogout 挂载到全局供 UI 模块调用
    window.handleLogout = handleLogout;
    
    console.log('✅ WebAuthn 应用初始化完成');
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', initApp);

// 导出函数供外部使用
export { handleRegister, handleLogin, handleLogout, checkAuthStatus };
