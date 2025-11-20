/* ========================================
   火星搜索 - 公共JavaScript函数
   ======================================== */

// API基础配置
const API_BASE = '/api';

// 工具函数集合
const Utils = {
    /**
     * 格式化数字（添加千位分隔符）
     */
    formatNumber(num) {
        return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',');
    },

    /**
     * 格式化日期时间
     */
    formatDateTime(timestamp) {
        const date = new Date(timestamp * 1000);
        return date.toLocaleString('zh-CN');
    },

    /**
     * HTML转义
     */
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    },

    /**
     * 复制文本到剪贴板
     */
    copyToClipboard(text) {
        const textarea = document.createElement('textarea');
        textarea.value = text;
        textarea.style.position = 'fixed';
        textarea.style.opacity = '0';
        document.body.appendChild(textarea);
        textarea.select();
        
        try {
            document.execCommand('copy');
            return true;
        } catch (err) {
            console.error('复制失败:', err);
            return false;
        } finally {
            document.body.removeChild(textarea);
        }
    },

    /**
     * 显示提示消息
     */
    showMessage(message, type = 'info') {
        // 简单的消息提示实现
        const colors = {
            success: '#52c41a',
            error: '#ff4d4f',
            warning: '#faad14',
            info: '#1890ff'
        };

        const messageBox = document.createElement('div');
        messageBox.textContent = message;
        messageBox.style.cssText = `
            position: fixed;
            top: 20px;
            left: 50%;
            transform: translateX(-50%);
            padding: 12px 24px;
            background: ${colors[type] || colors.info};
            color: white;
            border-radius: 4px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
            z-index: 9999;
            animation: slideDown 0.3s ease;
        `;

        document.body.appendChild(messageBox);

        setTimeout(() => {
            messageBox.style.animation = 'slideUp 0.3s ease';
            setTimeout(() => {
                document.body.removeChild(messageBox);
            }, 300);
        }, 3000);
    },

    /**
     * 防抖函数
     */
    debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    },

    /**
     * 节流函数
     */
    throttle(func, limit) {
        let inThrottle;
        return function(...args) {
            if (!inThrottle) {
                func.apply(this, args);
                inThrottle = true;
                setTimeout(() => inThrottle = false, limit);
            }
        };
    }
};

// API请求封装
const API = {
    /**
     * 获取认证token
     */
    getToken() {
        return localStorage.getItem('admin_token');
    },

    /**
     * 设置认证token
     */
    setToken(token) {
        localStorage.setItem('admin_token', token);
    },

    /**
     * 清除认证token
     */
    clearToken() {
        localStorage.removeItem('admin_token');
    },

    /**
     * 通用请求方法
     */
    async request(url, options = {}) {
        const defaultOptions = {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json'
            }
        };

        // 合并选项
        const finalOptions = { ...defaultOptions, ...options };

        // 添加认证token
        const token = this.getToken();
        if (token) {
            finalOptions.headers['Authorization'] = 'Bearer ' + token;
        }

        try {
            const response = await fetch(API_BASE + url, finalOptions);
            const data = await response.json();

            // 处理401未授权
            if (data.code === 401) {
                this.clearToken();
                if (window.location.pathname.startsWith('/admin')) {
                    window.location.href = '/admin/login';
                }
                throw new Error('认证失败，请重新登录');
            }

            return data;
        } catch (error) {
            console.error('API请求失败:', error);
            throw error;
        }
    },

    /**
     * GET请求
     */
    get(url, params = {}) {
        const queryString = new URLSearchParams(params).toString();
        const fullUrl = queryString ? `${url}?${queryString}` : url;
        return this.request(fullUrl);
    },

    /**
     * POST请求
     */
    post(url, data) {
        return this.request(url, {
            method: 'POST',
            body: JSON.stringify(data)
        });
    },

    /**
     * PUT请求
     */
    put(url, data) {
        return this.request(url, {
            method: 'PUT',
            body: JSON.stringify(data)
        });
    },

    /**
     * DELETE请求
     */
    delete(url, data) {
        return this.request(url, {
            method: 'DELETE',
            body: JSON.stringify(data)
        });
    }
};

// 网盘类型映射
const PanTypes = {
    0: { name: '夸克', color: 'primary' },
    2: { name: '百度', color: 'success' },
    3: { name: '阿里', color: 'success' },
    4: { name: 'UC', color: 'warning' },
    5: { name: '迅雷', color: 'danger' }
};

/**
 * 获取网盘类型名称
 */
function getPanTypeName(type) {
    return PanTypes[type]?.name || '未知';
}

/**
 * 获取网盘类型标签颜色
 */
function getPanTypeColor(type) {
    return PanTypes[type]?.color || 'default';
}

// 添加动画样式
const style = document.createElement('style');
style.textContent = `
    @keyframes slideDown {
        from {
            opacity: 0;
            transform: translate(-50%, -20px);
        }
        to {
            opacity: 1;
            transform: translate(-50%, 0);
        }
    }
    
    @keyframes slideUp {
        from {
            opacity: 1;
            transform: translate(-50%, 0);
        }
        to {
            opacity: 0;
            transform: translate(-50%, -20px);
        }
    }
`;
document.head.appendChild(style);

// 导出到全局
window.Utils = Utils;
window.API = API;
window.PanTypes = PanTypes;
window.getPanTypeName = getPanTypeName;
window.getPanTypeColor = getPanTypeColor;