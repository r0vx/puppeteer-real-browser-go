
// Sample ad blocker content script
console.log('🛡️ Sample Ad Blocker: Content script loaded');

// 简单的广告拦截模拟
const blockAds = () => {
	// 隐藏常见的广告选择器
	const adSelectors = ['.ad', '.ads', '.advertisement', '[id*="ad"]', '[class*="ad"]'];
	adSelectors.forEach(selector => {
		const ads = document.querySelectorAll(selector);
		ads.forEach(ad => {
			ad.style.display = 'none';
		});
	});
};

// 页面加载时运行
if (document.readyState === 'loading') {
	document.addEventListener('DOMContentLoaded', blockAds);
} else {
	blockAds();
}

// 标记扩展存在
window.AdBlockerExtension = true;
