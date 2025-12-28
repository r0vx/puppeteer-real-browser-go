package browser

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// UserFingerprintManager 用户指纹管理器
type UserFingerprintManager struct {
	configDir       string                        // 配置文件目录
	cache           map[string]*FingerprintConfig // 内存缓存
	generator       *FingerprintGenerator         // 指纹生成器
	mutex           sync.RWMutex                  // 读写锁
}

// GetInitParamsFromOptions 从 ConnectOptions 提取指纹初始化参数
func GetInitParamsFromOptions(opts *ConnectOptions) *FingerprintInitParams {
	if opts == nil {
		return nil
	}
	// 只有指定了参数才返回
	if opts.Width == 0 && opts.Height == 0 && opts.UserAgent == "" && opts.Language == "" && len(opts.Languages) == 0 && opts.Timezone == "" && opts.TimezoneOffset == 0 {
		return nil
	}
	return &FingerprintInitParams{
		Width:          opts.Width,
		Height:         opts.Height,
		UserAgent:      opts.UserAgent,
		Language:       opts.Language,
		Languages:      opts.Languages,
		Timezone:       opts.Timezone,
		TimezoneOffset: opts.TimezoneOffset,
	}
}

// NewUserFingerprintManager 创建用户指纹管理器
func NewUserFingerprintManager(configDir string) (*UserFingerprintManager, error) {
	// 确保配置目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %v", err)
	}
	
	return &UserFingerprintManager{
		configDir: configDir,
		cache:     make(map[string]*FingerprintConfig),
		generator: NewFingerprintGenerator(),
	}, nil
}

// FingerprintInitParams 指纹初始化参数
type FingerprintInitParams struct {
	Width          int      // 屏幕宽度
	Height         int      // 屏幕高度
	UserAgent      string   // UserAgent
	Language       string   // 主语言，如 "zh-CN"
	Languages      []string // 语言列表，如 ["zh-CN", "zh", "en"]
	Timezone       string   // 时区名称，如 "Asia/Shanghai"
	TimezoneOffset int      // 时区偏移（分钟），如 480 表示 UTC+8，0 表示未设置
}

// GetUserFingerprint 获取用户指纹配置（无初始化参数）
func (ufm *UserFingerprintManager) GetUserFingerprint(userID string) (*FingerprintConfig, error) {
	return ufm.GetOrCreateUserFingerprint(userID, nil)
}

// GetOrCreateUserFingerprint 获取或创建用户指纹配置
// 如果配置已存在，直接返回（不应用 initParams）
// 如果配置不存在，使用 initParams 创建新配置
func (ufm *UserFingerprintManager) GetOrCreateUserFingerprint(userID string, initParams *FingerprintInitParams) (*FingerprintConfig, error) {
	ufm.mutex.RLock()
	
	// 检查缓存
	if config, exists := ufm.cache[userID]; exists {
		ufm.mutex.RUnlock()
		return config, nil
	}
	
	ufm.mutex.RUnlock()
	
	// 尝试从文件加载
	configPath := ufm.getUserConfigPath(userID)
	if _, err := os.Stat(configPath); err == nil {
		config, err := ufm.loadConfigFromFile(configPath)
		if err == nil {
			ufm.mutex.Lock()
			ufm.cache[userID] = config
			ufm.mutex.Unlock()
			return config, nil
		}
	}
	
	// 生成新的指纹配置
	config := ufm.generator.GenerateFingerprint(userID)
	
	// 应用初始化参数（仅在新建时生效）
	if initParams != nil {
		if initParams.Width > 0 {
			config.Screen.Width = initParams.Width
			config.Screen.AvailWidth = initParams.Width
		}
		if initParams.Height > 0 {
			config.Screen.Height = initParams.Height
			config.Screen.AvailHeight = initParams.Height - 72 // 留出任务栏空间
		}
		if initParams.UserAgent != "" {
			config.Browser.UserAgent = initParams.UserAgent
		}
		if initParams.Language != "" {
			config.Browser.Language = initParams.Language
		}
		if len(initParams.Languages) > 0 {
			config.Browser.Languages = initParams.Languages
		}
		// 时区设置
		if initParams.Timezone != "" {
			config.Timezone.Timezone = initParams.Timezone
		}
		if initParams.TimezoneOffset != 0 {
			config.Timezone.Offset = initParams.TimezoneOffset
		}
		// 如果设置了语言但没设置时区，自动匹配时区
		if initParams.Language != "" && initParams.Timezone == "" && initParams.TimezoneOffset == 0 {
			tz, offset := getTimezoneForLanguage(initParams.Language)
			config.Timezone.Timezone = tz
			config.Timezone.Offset = offset
		}
	}
	
	// 保存到文件
	if err := ufm.saveConfigToFile(config, configPath); err != nil {
		return nil, fmt.Errorf("failed to save config: %v", err)
	}
	
	// 添加到缓存
	ufm.mutex.Lock()
	ufm.cache[userID] = config
	ufm.mutex.Unlock()
	
	return config, nil
}

// CreateCustomUserFingerprint 创建自定义用户指纹
func (ufm *UserFingerprintManager) CreateCustomUserFingerprint(userID string, customConfig *FingerprintConfig) error {
	customConfig.UserID = userID
	
	configPath := ufm.getUserConfigPath(userID)
	if err := ufm.saveConfigToFile(customConfig, configPath); err != nil {
		return fmt.Errorf("failed to save custom config: %v", err)
	}
	
	// 更新缓存
	ufm.mutex.Lock()
	ufm.cache[userID] = customConfig
	ufm.mutex.Unlock()
	
	return nil
}

// UpdateUserFingerprint 更新用户指纹配置
func (ufm *UserFingerprintManager) UpdateUserFingerprint(userID string, updates map[string]interface{}) error {
	config, err := ufm.GetUserFingerprint(userID)
	if err != nil {
		return err
	}
	
	// 应用更新（这里简化处理，实际应该更仔细地处理类型转换）
	if userAgent, ok := updates["userAgent"]; ok {
		if ua, ok := userAgent.(string); ok {
			config.Browser.UserAgent = ua
		}
	}
	
	if language, ok := updates["language"]; ok {
		if lang, ok := language.(string); ok {
			config.Browser.Language = lang
		}
	}
	
	if screenWidth, ok := updates["screenWidth"]; ok {
		if width, ok := screenWidth.(int); ok {
			config.Screen.Width = width
		}
	}
	
	if screenHeight, ok := updates["screenHeight"]; ok {
		if height, ok := screenHeight.(int); ok {
			config.Screen.Height = height
		}
	}
	
	// 保存更新后的配置
	configPath := ufm.getUserConfigPath(userID)
	if err := ufm.saveConfigToFile(config, configPath); err != nil {
		return fmt.Errorf("failed to update config: %v", err)
	}
	
	// 更新缓存
	ufm.mutex.Lock()
	ufm.cache[userID] = config
	ufm.mutex.Unlock()
	
	return nil
}

// DeleteUserFingerprint 删除用户指纹配置
func (ufm *UserFingerprintManager) DeleteUserFingerprint(userID string) error {
	// 删除文件
	configPath := ufm.getUserConfigPath(userID)
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete config file: %v", err)
	}
	
	// 从缓存中删除
	ufm.mutex.Lock()
	delete(ufm.cache, userID)
	ufm.mutex.Unlock()
	
	return nil
}

// ListUsers 列出所有用户
func (ufm *UserFingerprintManager) ListUsers() ([]string, error) {
	files, err := ioutil.ReadDir(ufm.configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %v", err)
	}
	
	var users []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			userID := file.Name()[:len(file.Name())-5] // 移除.json后缀
			users = append(users, userID)
		}
	}
	
	return users, nil
}

// GetUserStats 获取用户统计信息
func (ufm *UserFingerprintManager) GetUserStats() (map[string]interface{}, error) {
	users, err := ufm.ListUsers()
	if err != nil {
		return nil, err
	}
	
	stats := make(map[string]interface{})
	stats["total_users"] = len(users)
	stats["cached_users"] = len(ufm.cache)
	
	// 统计不同平台的用户数量
	platformCounts := make(map[string]int)
	languageCounts := make(map[string]int)
	
	for _, userID := range users {
		config, err := ufm.GetUserFingerprint(userID)
		if err != nil {
			continue
		}
		
		platformCounts[config.Browser.Platform]++
		languageCounts[config.Browser.Language]++
	}
	
	stats["platforms"] = platformCounts
	stats["languages"] = languageCounts
	
	return stats, nil
}

// GenerateBatchFingerprints 批量生成指纹
func (ufm *UserFingerprintManager) GenerateBatchFingerprints(userIDs []string) error {
	for _, userID := range userIDs {
		_, err := ufm.GetUserFingerprint(userID)
		if err != nil {
			return fmt.Errorf("failed to generate fingerprint for user %s: %v", userID, err)
		}
	}
	return nil
}

// ExportUserFingerprint 导出用户指纹配置
func (ufm *UserFingerprintManager) ExportUserFingerprint(userID string) (string, error) {
	config, err := ufm.GetUserFingerprint(userID)
	if err != nil {
		return "", err
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %v", err)
	}
	
	return string(data), nil
}

// ImportUserFingerprint 导入用户指纹配置
func (ufm *UserFingerprintManager) ImportUserFingerprint(userID, configJSON string) error {
	var config FingerprintConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %v", err)
	}
	
	config.UserID = userID
	return ufm.CreateCustomUserFingerprint(userID, &config)
}

// CloneUserFingerprint 克隆用户指纹配置
func (ufm *UserFingerprintManager) CloneUserFingerprint(sourceUserID, targetUserID string) error {
	sourceConfig, err := ufm.GetUserFingerprint(sourceUserID)
	if err != nil {
		return fmt.Errorf("failed to get source config: %v", err)
	}
	
	// 深拷贝配置
	data, err := json.Marshal(sourceConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal source config: %v", err)
	}
	
	var targetConfig FingerprintConfig
	if err := json.Unmarshal(data, &targetConfig); err != nil {
		return fmt.Errorf("failed to unmarshal target config: %v", err)
	}
	
	targetConfig.UserID = targetUserID
	return ufm.CreateCustomUserFingerprint(targetUserID, &targetConfig)
}

// 私有方法

// getUserConfigPath 获取用户配置文件路径
func (ufm *UserFingerprintManager) getUserConfigPath(userID string) string {
	return filepath.Join(ufm.configDir, fmt.Sprintf("%s.json", userID))
}

// loadConfigFromFile 从文件加载配置
func (ufm *UserFingerprintManager) loadConfigFromFile(configPath string) (*FingerprintConfig, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}
	
	var config FingerprintConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}
	
	return &config, nil
}

// saveConfigToFile 保存配置到文件
func (ufm *UserFingerprintManager) saveConfigToFile(config *FingerprintConfig, configPath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}
	
	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}
	
	return nil
}

// ConnectOptionsWithFingerprint 扩展连接选项以支持指纹
type ConnectOptionsWithFingerprint struct {
	*ConnectOptions
	UserID                string `json:"user_id,omitempty"`
	EnableFingerprinting  bool   `json:"enable_fingerprinting,omitempty"`
	FingerprintConfigDir  string `json:"fingerprint_config_dir,omitempty"`
	CustomFingerprintPath string `json:"custom_fingerprint_path,omitempty"`
}

// ConnectWithFingerprint 使用指纹配置连接浏览器
func ConnectWithFingerprint(ctx interface{}, opts *ConnectOptionsWithFingerprint) (interface{}, error) {
	if !opts.EnableFingerprinting || opts.UserID == "" {
		// 如果未启用指纹或没有用户ID，使用普通连接
		// 注意：这里需要实际的Connect函数实现
		return nil, fmt.Errorf("Connect function not implemented")
	}
	
	// 创建指纹管理器
	configDir := opts.FingerprintConfigDir
	if configDir == "" {
		configDir = "./fingerprint_configs"
	}
	
	fingerprintManager, err := NewUserFingerprintManager(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create fingerprint manager: %v", err)
	}
	
	// 获取用户指纹配置
	var fingerprintConfig *FingerprintConfig
	if opts.CustomFingerprintPath != "" {
		// 从自定义路径加载
		fingerprintConfig, err = fingerprintManager.loadConfigFromFile(opts.CustomFingerprintPath)
	} else {
		// 获取或生成用户指纹
		fingerprintConfig, err = fingerprintManager.GetUserFingerprint(opts.UserID)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to get fingerprint config: %v", err)
	}
	
	// 创建指纹注入器
	injector := NewFingerprintInjector(fingerprintConfig)
	
	// 修改连接选项以包含指纹相关的Chrome参数
	if opts.Args == nil {
		opts.Args = []string{}
	}
	
	// 添加指纹相关的Chrome参数
	fingerprintFlags := fingerprintConfig.GetChromeFlags()
	opts.Args = append(opts.Args, fingerprintFlags...)
	
	// 添加JavaScript注入参数
	injectionScript := injector.GenerateInjectionScript()
	preloadScript := injector.GetPreloadScript()
	
	// 这里需要实现脚本注入机制
	// 可以通过扩展或者其他方式注入JavaScript
	
	fmt.Printf("🔧 Connecting with fingerprint for user: %s\n", opts.UserID)
	fmt.Printf("📊 User Agent: %s\n", fingerprintConfig.Browser.UserAgent)
	fmt.Printf("🖥️  Screen: %dx%d\n", fingerprintConfig.Screen.Width, fingerprintConfig.Screen.Height)
	fmt.Printf("🌍 Language: %s\n", fingerprintConfig.Browser.Language)
	fmt.Printf("⏰ Timezone: %s\n", fingerprintConfig.Timezone.Timezone)
	
	// 将注入脚本保存到临时文件或通过其他方式传递给浏览器
	_ = injectionScript
	_ = preloadScript
	
	// 使用修改后的选项连接浏览器
	// 注意：这里需要实际的Connect函数实现
	return nil, fmt.Errorf("Connect function not implemented - please use the actual browser connection method")
}

// getTimezoneForLanguage 根据语言返回匹配的时区
// 返回时区名称和偏移量（分钟）
func getTimezoneForLanguage(lang string) (timezone string, offset int) {
	// 语言到时区的映射
	languageTimezones := map[string]struct {
		tz     string
		offset int
	}{
		"zh-CN": {"Asia/Shanghai", 480},      // UTC+8
		"zh-TW": {"Asia/Taipei", 480},        // UTC+8
		"zh-HK": {"Asia/Hong_Kong", 480},     // UTC+8
		"ja":    {"Asia/Tokyo", 540},         // UTC+9
		"ja-JP": {"Asia/Tokyo", 540},         // UTC+9
		"ko":    {"Asia/Seoul", 540},         // UTC+9
		"ko-KR": {"Asia/Seoul", 540},         // UTC+9
		"en-US": {"America/New_York", -300},  // UTC-5 (EST)
		"en-GB": {"Europe/London", 0},        // UTC+0
		"en-AU": {"Australia/Sydney", 600},   // UTC+10
		"de":    {"Europe/Berlin", 60},       // UTC+1
		"de-DE": {"Europe/Berlin", 60},       // UTC+1
		"fr":    {"Europe/Paris", 60},        // UTC+1
		"fr-FR": {"Europe/Paris", 60},        // UTC+1
		"es":    {"Europe/Madrid", 60},       // UTC+1
		"es-ES": {"Europe/Madrid", 60},       // UTC+1
		"pt":    {"Europe/Lisbon", 0},        // UTC+0
		"pt-BR": {"America/Sao_Paulo", -180}, // UTC-3
		"ru":    {"Europe/Moscow", 180},      // UTC+3
		"ru-RU": {"Europe/Moscow", 180},      // UTC+3
		"ar":    {"Asia/Riyadh", 180},        // UTC+3
		"th":    {"Asia/Bangkok", 420},       // UTC+7
		"th-TH": {"Asia/Bangkok", 420},       // UTC+7
		"vi":    {"Asia/Ho_Chi_Minh", 420},   // UTC+7
		"vi-VN": {"Asia/Ho_Chi_Minh", 420},   // UTC+7
		"id":    {"Asia/Jakarta", 420},       // UTC+7
		"id-ID": {"Asia/Jakarta", 420},       // UTC+7
	}

	if tz, ok := languageTimezones[lang]; ok {
		return tz.tz, tz.offset
	}

	// 尝试使用语言前缀匹配
	if len(lang) >= 2 {
		prefix := lang[:2]
		if tz, ok := languageTimezones[prefix]; ok {
			return tz.tz, tz.offset
		}
	}

	// 默认返回 UTC
	return "UTC", 0
}