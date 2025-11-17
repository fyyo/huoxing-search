package api

// getInstallHTML 返回安装页面HTML
func getInstallHTML() string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Xinyue-Go 系统安装向导</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            max-width: 800px;
            width: 100%;
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        .header h1 { font-size: 28px; margin-bottom: 10px; }
        .header p { opacity: 0.9; }
        .content { padding: 40px; }
        .step { display: none; }
        .step.active {
            display: block;
            animation: fadeIn 0.3s ease;
        }
        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(10px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .form-group { margin-bottom: 20px; }
        .form-group label {
            display: block;
            margin-bottom: 8px;
            color: #333;
            font-weight: 500;
        }
        .form-group input {
            width: 100%;
            padding: 12px;
            border: 2px solid #e0e0e0;
            border-radius: 6px;
            font-size: 14px;
            transition: border-color 0.3s;
        }
        .form-group input:focus {
            outline: none;
            border-color: #667eea;
        }
        .form-group small {
            display: block;
            margin-top: 5px;
            color: #666;
            font-size: 12px;
        }
        .btn-group {
            display: flex;
            gap: 10px;
            margin-top: 30px;
        }
        .btn {
            flex: 1;
            padding: 12px 24px;
            border: none;
            border-radius: 6px;
            font-size: 16px;
            cursor: pointer;
            transition: all 0.3s;
        }
        .btn-primary {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        .btn-primary:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
        }
        .btn-secondary {
            background: #f5f5f5;
            color: #666;
        }
        .btn-secondary:hover { background: #e0e0e0; }
        .alert {
            padding: 12px 16px;
            border-radius: 6px;
            margin-bottom: 20px;
        }
        .alert-success {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }
        .alert-error {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }
        .check-list { list-style: none; }
        .check-item {
            padding: 12px;
            border-bottom: 1px solid #e0e0e0;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .check-item:last-child { border-bottom: none; }
        .check-status { font-weight: bold; }
        .check-status.pass { color: #28a745; }
        .check-status.fail { color: #dc3545; }
        .progress {
            background: #f5f5f5;
            border-radius: 10px;
            height: 20px;
            overflow: hidden;
            margin: 20px 0;
        }
        .progress-bar {
            background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
            height: 100%;
            transition: width 0.3s;
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-size: 12px;
        }
        .loading {
            display: inline-block;
            width: 20px;
            height: 20px;
            border: 3px solid #f3f3f3;
            border-top: 3px solid #667eea;
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 Xinyue-Go 系统安装向导</h1>
            <p>欢迎使用 Xinyue-Go 网盘搜索系统</p>
        </div>
        <div class="content">
            <!-- 步骤1: 环境检查 -->
            <div class="step active" id="step1">
                <h2>步骤 1: 环境检查</h2>
                <div id="checkResult"></div>
                <div class="btn-group">
                    <button class="btn btn-primary" onclick="checkEnvironment()">开始检查</button>
                </div>
            </div>

            <!-- 步骤2: 数据库配置 -->
            <div class="step" id="step2">
                <h2>步骤 2: 数据库配置</h2>
                <div id="dbTestResult"></div>
                <form id="installForm">
                    <div class="form-group">
                        <label>数据库地址</label>
                        <input type="text" name="db_host" value="127.0.0.1" required>
                        <small>数据库服务器地址，通常为 localhost 或 127.0.0.1</small>
                    </div>
                    <div class="form-group">
                        <label>数据库端口</label>
                        <input type="number" name="db_port" value="3306" required>
                        <small>MySQL 默认端口为 3306</small>
                    </div>
                    <div class="form-group">
                        <label>数据库用户名</label>
                        <input type="text" name="db_user" value="root" required>
                    </div>
                    <div class="form-group">
                        <label>数据库密码</label>
                        <input type="password" name="db_password">
                    </div>
                    <div class="form-group">
                        <label>数据库名</label>
                        <input type="text" name="db_name" value="xinyue" required>
                        <small>如果数据库不存在将自动创建</small>
                    </div>
                    <div class="form-group">
                        <label>表前缀</label>
                        <input type="text" name="db_prefix" value="qf_" required>
                        <small>建议使用默认值</small>
                    </div>
                    <div class="form-group">
                        <label style="display: flex; align-items: center; cursor: pointer;">
                            <input type="checkbox" name="db_ssl_mode" style="width: auto; margin-right: 8px;">
                            <span>启用 SSL 连接（适用于云数据库，本地MySQL建议关闭）</span>
                        </label>
                        <small>云数据库(如TiDB Cloud)需要开启，本地MySQL建议关闭以避免证书验证问题</small>
                    </div>
                    <div class="form-group">
                        <label>网站名称</label>
                        <input type="text" name="site_name" value="Xinyue 网盘搜索" required>
                    </div>
                    <div class="form-group">
                        <label>管理员账号</label>
                        <input type="text" name="admin_user" value="admin" required>
                    </div>
                    <div class="form-group">
                        <label>管理员密码</label>
                        <input type="password" name="admin_pass" required minlength="6">
                        <small>密码长度至少6位</small>
                    </div>
                    <div class="btn-group">
                        <button type="button" class="btn btn-secondary" onclick="testDatabase()">测试连接</button>
                        <button type="button" class="btn btn-primary" onclick="executeInstall()">开始安装</button>
                    </div>
                </form>
            </div>

            <!-- 步骤3: 安装完成 -->
            <div class="step" id="step3">
                <h2>步骤 3: 安装完成</h2>
                <div id="installResult"></div>
            </div>
        </div>
    </div>

    <script>
        let currentStep = 1;

        function showStep(step) {
            document.querySelectorAll('.step').forEach(el => el.classList.remove('active'));
            document.getElementById('step' + step).classList.add('active');
            currentStep = step;
        }

        async function checkEnvironment() {
            const resultDiv = document.getElementById('checkResult');
            resultDiv.innerHTML = '<div class="loading"></div> 正在检查环境...';

            try {
                const response = await fetch('/install/check');
                const data = await response.json();

                if (data.code === 200) {
                    let html = '<ul class="check-list">';
                    data.checks.forEach(check => {
                        const status = check.status ? '✓ 通过' : '✗ 失败';
                        const statusClass = check.status ? 'pass' : 'fail';
                        html += ` + "`" + `
                            <li class="check-item">
                                <span>${check.name}: ${check.value}</span>
                                <span class="check-status ${statusClass}">${status}</span>
                            </li>
                        ` + "`" + `;
                    });
                    html += '</ul>';

                    if (data.all_pass) {
                        html += '<div class="alert alert-success">✓ 环境检查通过，可以继续安装</div>';
                        html += '<div class="btn-group"><button class="btn btn-primary" onclick="showStep(2)">下一步</button></div>';
                    } else {
                        html += '<div class="alert alert-error">✗ 环境检查未通过，请解决上述问题后重试</div>';
                        html += '<div class="btn-group"><button class="btn btn-primary" onclick="checkEnvironment()">重新检查</button></div>';
                    }

                    resultDiv.innerHTML = html;
                }
            } catch (error) {
                resultDiv.innerHTML = '<div class="alert alert-error">检查失败: ' + error.message + '</div>';
            }
        }

        async function testDatabase() {
            const form = document.getElementById('installForm');
            const formData = new FormData(form);
            const data = {
                db_host: formData.get('db_host'),
                db_port: parseInt(formData.get('db_port')),
                db_user: formData.get('db_user'),
                db_password: formData.get('db_password'),
                db_name: formData.get('db_name'),
                db_prefix: formData.get('db_prefix'),
                db_ssl_mode: formData.get('db_ssl_mode') === 'on',
                site_name: formData.get('site_name'),
                admin_user: formData.get('admin_user'),
                admin_pass: formData.get('admin_pass')
            };

            const resultDiv = document.getElementById('dbTestResult');
            resultDiv.innerHTML = '<div class="loading"></div> 正在测试数据库连接...';

            try {
                const response = await fetch('/install/test-db', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(data)
                });
                const result = await response.json();

                if (result.code === 200) {
                    resultDiv.innerHTML = '<div class="alert alert-success">✓ ' + result.message + '</div>';
                } else {
                    resultDiv.innerHTML = '<div class="alert alert-error">✗ ' + result.message + '</div>';
                }
            } catch (error) {
                resultDiv.innerHTML = '<div class="alert alert-error">测试失败: ' + error.message + '</div>';
            }
        }

        async function executeInstall() {
            const form = document.getElementById('installForm');
            
            if (!form.checkValidity()) {
                form.reportValidity();
                return;
            }

            if (!confirm('确认要开始安装吗？')) {
                return;
            }

            const formData = new FormData(form);
            const data = {
                db_host: formData.get('db_host'),
                db_port: parseInt(formData.get('db_port')),
                db_user: formData.get('db_user'),
                db_password: formData.get('db_password'),
                db_name: formData.get('db_name'),
                db_prefix: formData.get('db_prefix'),
                db_ssl_mode: formData.get('db_ssl_mode') === 'on',
                site_name: formData.get('site_name'),
                admin_user: formData.get('admin_user'),
                admin_pass: formData.get('admin_pass')
            };

            showStep(3);
            const resultDiv = document.getElementById('installResult');
            resultDiv.innerHTML = ` + "`" + `
                <div class="progress">
                    <div class="progress-bar" style="width: 50%">正在安装...</div>
                </div>
                <p style="text-align: center; margin-top: 20px;">
                    <span class="loading"></span> 正在创建数据库和配置文件，请稍候...
                </p>
            ` + "`" + `;

            try {
                const response = await fetch('/install/execute', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(data)
                });
                const result = await response.json();

                if (result.code === 200) {
                    resultDiv.innerHTML = ` + "`" + `
                        <div class="progress">
                            <div class="progress-bar" style="width: 100%">安装完成</div>
                        </div>
                        <div class="alert alert-success" style="text-align: center;">
                            <h3 style="margin-bottom: 20px;">🎉 安装成功！</h3>
                            <p style="margin-bottom: 15px;">系统已成功安装并初始化完成</p>
                            <p style="margin-bottom: 20px;"><strong>管理员账号:</strong> ${data.admin_user}</p>
                            <div style="margin-top: 30px;">
                                <a href="/admin/login" class="btn" style="display: inline-block; padding: 12px 30px; background: #4CAF50; color: white; text-decoration: none; border-radius: 4px; font-size: 16px;">
                                    进入管理后台
                                </a>
                            </div>
                            <p style="margin-top: 20px; font-size: 14px; color: #666;">
                                提示：系统已自动完成初始化，无需重启程序
                            </p>
                        </div>
                    ` + "`" + `;
                } else {
                    resultDiv.innerHTML = ` + "`" + `
                        <div class="alert alert-error">
                            <h3>✗ 安装失败</h3>
                            <p>${result.message}</p>
                        </div>
                        <div class="btn-group">
                            <button class="btn btn-primary" onclick="showStep(2)">返回修改</button>
                        </div>
                    ` + "`" + `;
                }
            } catch (error) {
                resultDiv.innerHTML = ` + "`" + `
                    <div class="alert alert-error">
                        <h3>✗ 安装失败</h3>
                        <p>错误信息: ${error.message}</p>
                    </div>
                    <div class="btn-group">
                        <button class="btn btn-primary" onclick="showStep(2)">返回修改</button>
                    </div>
                ` + "`" + `;
            }
        }

        window.addEventListener('load', function() {
            setTimeout(checkEnvironment, 500);
        });
    </script>
</body>
</html>`
}