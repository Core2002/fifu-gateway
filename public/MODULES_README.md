# WebAuthn 前端模块说明

## 📁 文件结构

```
public/
├── main.js              # 应用主入口，整合所有模块
├── webauthn-core.js     # WebAuthn 核心业务逻辑（与 DOM 解耦）
└── utils/
    ├── index.js         # 工具函数统一导出
    ├── base64.js        # Base64URL 编码/解码工具
    ├── token.js         # Token 和用户信息管理
    ├── api.js           # API 通信层
    └── ui.js            # UI 工具函数
```

## 🎯 各模块职责

### 1. `main.js` - 应用主入口
- **职责**: 整合所有模块，初始化应用
- **功能**:
  - 页面加载时检查 WebAuthn 支持
  - 绑定按钮事件监听器
  - 处理用户交互（注册、登录、登出）
  - 协调各模块工作

### 2. `webauthn-core.js` - WebAuthn 核心类
- **职责**: 纯业务逻辑，**不操作 DOM**
- **功能**:
  - 检查 WebAuthn 浏览器支持
  - 执行注册流程（获取挑战、调用 WebAuthn API、验证响应）
  - 执行登录流程（获取挑战、调用 WebAuthn API、验证响应）
  - 错误处理和消息格式化
- **特点**: 
  - 使用回调函数传递进度消息
  - 返回 Promise 对象
  - 完全与 DOM 解耦，可复用

### 3. `utils/base64.js` - Base64 工具
- **功能**:
  - `base64urlToBuffer()`: Base64URL → ArrayBuffer
  - `bufferToBase64url()`: ArrayBuffer → Base64URL

### 4. `utils/token.js` - Token 管理
- **功能**:
  - `saveToken()`: 保存访问令牌
  - `getToken()`: 获取访问令牌
  - `removeToken()`: 移除访问令牌
  - `saveUserInfo()`: 保存用户信息
  - `getUserInfo()`: 获取用户信息
  - `removeUserInfo()`: 移除用户信息

### 5. `utils/api.js` - API 通信
- **功能**:
  - `callApi()`: 通用 HTTP 请求封装
  - 自动添加 Authorization header
  - 统一的错误处理

### 6. `utils/ui.js` - UI 工具
- **功能**:
  - `showMessage()`: 显示消息提示
  - `updateLoginStatus()`: 更新登录状态显示
  - `showProtectedContent()`: 显示受保护内容
  - `hideProtectedContent()`: 隐藏受保护内容
  - `clearInput()`: 清空输入框
  - `getInputValue()`: 获取输入值

## 🔧 使用方式

### 在 HTML 中引入
```html
<script type="module" src="/app/main.js"></script>
```

### 在其他模块中使用
```javascript
// 导入核心类
import { webAuthn } from './webauthn-core.js';

// 导入工具函数
import { showMessage, clearInput } from './utils/ui.js';
import { getToken, saveToken } from './utils/token.js';
import { callApi } from './utils/api.js';
```

## 📝 设计原则

### 1. **关注点分离**
- **业务逻辑** (`webauthn-core.js`): 纯粹的 WebAuthn 流程，不依赖 DOM
- **UI 交互** (`main.js`, `ui.js`): 负责 DOM 操作和用户界面
- **工具函数** (`utils/*.js`): 可复用的辅助功能

### 2. **解耦设计**
- `webAuthnCore` 类通过回调函数传递进度，不直接操作 DOM
- UI 模块监听自定义事件或调用核心方法
- 各模块之间通过明确的接口通信

### 3. **单一职责**
每个文件只负责一个明确的功能领域：
- Base64 转换只在 `base64.js`
- Token 管理只在 `token.js`
- WebAuthn 核心逻辑只在 `webauthn-core.js`

## 🚀 优势

✅ **可维护性**: 代码结构清晰，易于理解和修改  
✅ **可测试性**: 核心逻辑与 UI 分离，便于单元测试  
✅ **可复用性**: 工具函数和核心类可在其他项目中重用  
✅ **可扩展性**: 新增功能时只需修改对应模块  

## 📖 示例代码

### 执行注册
```javascript
import { webAuthn } from './webauthn-core.js';
import { showMessage } from './utils/ui.js';

const result = await webAuthn.register('username', (message, isError) => {
    showMessage(message, isError);
});

if (result.success) {
    console.log('注册成功');
}
```

### 执行登录
```javascript
import { webAuthn } from './webauthn-core.js';
import { showMessage } from './utils/ui.js';

const result = await webAuthn.login('username', async (message, isError) => {
    showMessage(message, isError);
    
    if (!isError && message.includes('登录成功')) {
        // 处理登录后的逻辑
    }
});
```
