
// Sample password manager content script
console.log('🔐 Sample Password Manager: Content script loaded');

// 检测密码字段
const detectPasswordFields = () => {
	const passwordFields = document.querySelectorAll('input[type="password"]');
	const emailFields = document.querySelectorAll('input[type="email"], input[name*="email"], input[name*="username"]');
	
	if (passwordFields.length > 0) {
		console.log('🔐 Password fields detected:', passwordFields.length);
		
		// 添加自动填充提示
		passwordFields.forEach(field => {
			field.addEventListener('focus', () => {
				console.log('🔐 Password field focused - auto-fill available');
			});
		});
	}
};

// 页面加载时检测
if (document.readyState === 'loading') {
	document.addEventListener('DOMContentLoaded', detectPasswordFields);
} else {
	detectPasswordFields();
}

// 标记扩展存在
window.PasswordManagerExtension = true;
