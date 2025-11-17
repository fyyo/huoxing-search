/**
 * 后台侧边栏管理 - 统一的菜单配置
 * 修改侧边栏只需要修改这一个文件
 */

// 侧边栏菜单配置
const SIDEBAR_MENU = [
    {
        type: 'item',
        icon: '📊',
        title: '控制台',
        path: '/admin'
    },
    {
        type: 'group',
        title: '资源管理',
        items: [
            { icon: '📁', title: '资源列表', path: '/admin/source/list' },
            { icon: '🏷️', title: '资源分类', path: '/admin/source/category' },
            { icon: '📤', title: '批量导入', path: '/admin/source/import' }
        ]
    },
    {
        type: 'group',
        title: '搜索配置',
        items: [
            { icon: '🔗', title: '搜索线路', path: '/admin/search/api' }
        ]
    },
    {
        type: 'item',
        icon: '👤',
        title: '用户管理',
        path: '/admin/user'
    },
    {
        type: 'group',
        title: '系统设置',
        items: [
            { icon: '⚙️', title: '基本配置', path: '/admin/system/config' },
            { icon: '☁️', title: '网盘配置', path: '/admin/system/netdisk' }
        ]
    },
    {
        type: 'group',
        title: '微信配置',
        items: [
            { icon: '🤖', title: '微信配置', path: '/admin/system/wechat' }
        ]
    }
];

/**
 * 渲染侧边栏菜单
 * @param {string} containerId - 侧边栏容器ID
 * @param {string} currentPath - 当前页面路径（用于高亮当前菜单）
 */
function renderSidebar(containerId, currentPath) {
    const container = document.getElementById(containerId);
    if (!container) {
        console.error('侧边栏容器不存在:', containerId);
        return;
    }

    let html = '';

    SIDEBAR_MENU.forEach(item => {
        if (item.type === 'item') {
            // 单个菜单项
            const isActive = currentPath === item.path ? 'active' : '';
            html += `<a href="${item.path}" class="menu-item ${isActive}">
                ${item.icon} ${item.title}
            </a>`;
        } else if (item.type === 'group') {
            // 菜单组
            html += `<div class="menu-group">
                <div class="menu-group-title">${item.title}</div>`;
            
            item.items.forEach(subItem => {
                const isActive = currentPath === subItem.path ? 'active' : '';
                html += `<a href="${subItem.path}" class="menu-item ${isActive}">
                    ${subItem.icon} ${subItem.title}
                </a>`;
            });
            
            html += `</div>`;
        }
    });

    container.innerHTML = html;
}

/**
 * 初始化侧边栏
 * 自动检测当前页面路径并高亮对应菜单
 */
function initSidebar() {
    const currentPath = window.location.pathname;
    renderSidebar('sidebarMenu', currentPath);
}

// 页面加载完成后自动初始化
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initSidebar);
} else {
    initSidebar();
}