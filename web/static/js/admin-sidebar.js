/* ========================================
   火星搜索 - 管理后台侧边栏菜单
   ======================================== */

// 菜单配置
const menuConfig = [
    {
        group: '控制台',
        items: [
            { icon: '📊', text: '控制台', href: '/admin' }
        ]
    },
    {
        group: '资源管理',
        items: [
            { icon: '📁', text: '资源列表', href: '/admin/source/list' },
            { icon: '📤', text: '批量导入', href: '/admin/source/import' },
            { icon: '📂', text: '分类管理', href: '/admin/source/category' }
        ]
    },
    {
        group: '搜索配置',
        items: [
            { icon: '🔗', text: '搜索线路', href: '/admin/search/api' },
            { icon: '🌐', text: '网盘配置', href: '/admin/system/netdisk' }
        ]
    },
    {
        group: '系统管理',
        items: [
            { icon: '⚙️', text: '系统设置', href: '/admin/system/config' },
            { icon: '👥', text: '管理员', href: '/admin/user' },
            { icon: '💬', text: '微信配置', href: '/admin/system/wechat' }
        ]
    }
];

/**
 * 渲染侧边栏菜单
 */
function renderSidebar() {
    const container = document.getElementById('sidebarMenu');
    if (!container) return;

    const currentPath = window.location.pathname;
    let html = '';

    menuConfig.forEach(group => {
        html += `<div class="menu-group">`;
        html += `<div class="menu-group-title">${group.group}</div>`;
        
        group.items.forEach(item => {
            // 修复激活状态判断：精确匹配或子路径匹配（但 /admin 只能精确匹配）
            let isActive = false;
            if (item.href === '/admin') {
                // 控制台页面：只有精确匹配才激活
                isActive = currentPath === '/admin';
            } else {
                // 其他页面：精确匹配或子路径匹配
                isActive = currentPath === item.href || currentPath.startsWith(item.href + '/');
            }
            
            html += `
                <a href="${item.href}" class="menu-item ${isActive ? 'active' : ''}">
                    <span class="menu-icon">${item.icon}</span>
                    <span>${item.text}</span>
                </a>
            `;
        });
        
        html += `</div>`;
    });

    container.innerHTML = html;
}

// 页面加载时渲染侧边栏
// 如果DOM已加载，立即执行；否则等待DOMContentLoaded
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', renderSidebar);
} else {
    renderSidebar();
}