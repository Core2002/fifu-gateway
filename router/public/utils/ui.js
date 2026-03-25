// ============================================
// utils/ui.js - UI 工具函数
// 处理 DOM 操作和消息显示
// ============================================

/**
 * 显示消息提示
 * @param {string} message - 要显示的消息
 * @param {boolean} isError - 是否为错误消息
 */
export function showMessage(message, isError = false) {
    const messageDiv = document.getElementById('message');
    if (!messageDiv) {
        console.warn('未找到消息显示元素 (#message)');
        return;
    }
    
    messageDiv.textContent = message;
    messageDiv.className = isError ? 'error' : 'success';
    messageDiv.style.display = 'block';
    console.log(`[${isError ? 'ERROR' : 'INFO'}] ${message}`);
}

/**
 * 更新登录状态显示
 * @param {string} username - 用户名
 */
export function updateLoginStatus(username) {
    const statusDiv = document.getElementById('loginStatus');
    if (!statusDiv) {
        console.warn('未找到登录状态元素 (#loginStatus)');
        return;
    }
    
    // 创建退出登录链接的 HTML
    statusDiv.innerHTML = `当前用户：<strong>${username}</strong> | <a href="#" id="logoutLink" style="color: #2563eb; text-decoration: underline;">退出</a>`;
    statusDiv.style.display = 'block';
    
    // 为退出链接添加事件监听器
    const logoutLink = document.getElementById('logoutLink');
    if (logoutLink) {
        logoutLink.addEventListener('click', (e) => {
            e.preventDefault();
            // 直接调用全局的 handleLogout 函数
            if (window.handleLogout) {
                window.handleLogout();
            }
            // 同时触发登出事件，供其他模块监听
            window.dispatchEvent(new CustomEvent('app:logout'));
        });
    }
    
    // 更新用户信息卡片
    const userInfoDiv = document.getElementById('userInfo');
    if (userInfoDiv) {
        const user = window.appUserInfo;
        if (user) {
            userInfoDiv.innerHTML = `
                <p><strong>ID:</strong> ${user.id}</p>
                <p><strong>用户名:</strong> ${user.username}</p>
                <p><strong>角色:</strong> ${user.role}</p>
            `;
        }
    }
}

/**
 * 显示受保护的内容
 */
export function showProtectedContent() {
    const protectedElements = document.querySelectorAll('.protected-content');
    protectedElements.forEach(el => el.style.display = 'block');
}

/**
 * 隐藏受保护的内容
 */
export function hideProtectedContent() {
    const protectedElements = document.querySelectorAll('.protected-content');
    protectedElements.forEach(el => el.style.display = 'none');
}

/**
 * 清空输入框
 * @param {string} elementId - 输入框的元素 ID
 */
export function clearInput(elementId) {
    const input = document.getElementById(elementId);
    if (input) {
        input.value = '';
    }
}

/**
 * 获取输入框的值
 * @param {string} elementId - 输入框的元素 ID
 * @returns {string} - 输入框的值
 */
export function getInputValue(elementId) {
    const input = document.getElementById(elementId);
    return input ? input.value.trim() : '';
}
