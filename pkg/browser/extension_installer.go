package browser

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtensionInstaller handles automatic extension installation
type ExtensionInstaller struct {
	userDataDir string
}

// ExtensionManifest represents the structure of manifest.json
type ExtensionManifest struct {
	ManifestVersion int    `json:"manifest_version"`
	Name            string `json:"name"`
	Version         string `json:"version"`
	Description     string `json:"description"`
}

// NewExtensionInstaller creates a new extension installer
func NewExtensionInstaller(userDataDir string) *ExtensionInstaller {
	return &ExtensionInstaller{
		userDataDir: userDataDir,
	}
}

// PreInstallExtensions automatically installs extensions to user data directory
func (ei *ExtensionInstaller) PreInstallExtensions(extensionPaths []string) error {
	if len(extensionPaths) == 0 {
		return nil
	}

	extensionsDir := filepath.Join(ei.userDataDir, "Default", "Extensions")
	fmt.Printf("  🔧 创建扩展目录: %s\n", extensionsDir)
	if err := os.MkdirAll(extensionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create extensions directory: %w", err)
	}

	for i, extPath := range extensionPaths {
		fmt.Printf("  📦 正在安装扩展 %d/%d: %s\n", i+1, len(extensionPaths), extPath)
		
		// Check if source path exists
		if _, err := os.Stat(extPath); err != nil {
			fmt.Printf("      ❌ 源路径不存在: %v\n", err)
			return fmt.Errorf("source extension path does not exist %s: %w", extPath, err)
		}
		
		if err := ei.installExtension(extPath, extensionsDir); err != nil {
			fmt.Printf("      ❌ 安装失败: %v\n", err)
			return fmt.Errorf("failed to install extension %s: %w", extPath, err)
		}
		
		fmt.Printf("      ✅ 安装成功\n")
	}

	return nil
}

// installExtension installs a single extension
func (ei *ExtensionInstaller) installExtension(extensionPath, extensionsDir string) error {
	// Check if it's a .crx file or directory
	if strings.HasSuffix(extensionPath, ".crx") {
		return ei.installCRXExtension(extensionPath, extensionsDir)
	} else {
		return ei.installDirectoryExtension(extensionPath, extensionsDir)
	}
}

// installCRXExtension installs a .crx extension file
func (ei *ExtensionInstaller) installCRXExtension(crxPath, extensionsDir string) error {
	// For .crx files, we need to extract them and install
	// CRX format: header + zip data
	// For now, we'll treat them as zip files (simplified approach)
	
	// Read the .crx file
	crxFile, err := os.Open(crxPath)
	if err != nil {
		return fmt.Errorf("failed to open CRX file: %w", err)
	}
	defer crxFile.Close()

	// CRX files start with "Cr24" magic number, followed by version and lengths
	// Skip the CRX header to get to the ZIP data
	header := make([]byte, 16)
	if _, err := crxFile.Read(header); err != nil {
		return fmt.Errorf("failed to read CRX header: %w", err)
	}

	// Check if it's a valid CRX file
	if string(header[:4]) != "Cr24" {
		return fmt.Errorf("invalid CRX file format")
	}

	// Get CRX version
	version := uint32(header[4]) | uint32(header[5])<<8 | uint32(header[6])<<16 | uint32(header[7])<<24

	var skipBytes int64
	if version == 2 {
		// CRX version 2 format
		pubKeyLen := uint32(header[8]) | uint32(header[9])<<8 | uint32(header[10])<<16 | uint32(header[11])<<24
		sigLen := uint32(header[12]) | uint32(header[13])<<8 | uint32(header[14])<<16 | uint32(header[15])<<24
		skipBytes = int64(pubKeyLen + sigLen)
	} else if version == 3 {
		// CRX version 3 format - has additional header length field
		headerLen := uint32(header[8]) | uint32(header[9])<<8 | uint32(header[10])<<16 | uint32(header[11])<<24
		skipBytes = int64(headerLen)
	} else {
		return fmt.Errorf("unsupported CRX version: %d", version)
	}

	// Skip header data to get to ZIP data
	if _, err := crxFile.Seek(skipBytes, io.SeekCurrent); err != nil {
		return fmt.Errorf("failed to skip CRX metadata: %w", err)
	}

	// Create temporary file for ZIP extraction
	tempDir := filepath.Join(os.TempDir(), "crx_extract_"+filepath.Base(crxPath))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Read the rest as ZIP data
	zipData, err := io.ReadAll(crxFile)
	if err != nil {
		return fmt.Errorf("failed to read ZIP data from CRX: %w", err)
	}

	// Write ZIP data to temporary file
	tempZipPath := filepath.Join(tempDir, "extension.zip")
	if err := os.WriteFile(tempZipPath, zipData, 0644); err != nil {
		return fmt.Errorf("failed to write temp ZIP file: %w", err)
	}

	// Extract ZIP to temporary directory
	extractDir := filepath.Join(tempDir, "extracted")
	if err := ei.extractZip(tempZipPath, extractDir); err != nil {
		return fmt.Errorf("failed to extract CRX: %w", err)
	}

	// Now install the extracted directory
	return ei.installDirectoryExtension(extractDir, extensionsDir)
}

// installDirectoryExtension installs an unpacked extension directory
func (ei *ExtensionInstaller) installDirectoryExtension(sourcePath, extensionsDir string) error {
	// Read manifest to get extension ID and version
	manifestPath := filepath.Join(sourcePath, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest.json: %w", err)
	}

	var manifest ExtensionManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest.json: %w", err)
	}

	// Generate extension ID (simplified - in reality Chrome uses the public key)
	extensionID := ei.generateExtensionID(sourcePath)
	
	// Create extension directory: Extensions/[ID]/[Version]
	targetDir := filepath.Join(extensionsDir, extensionID, manifest.Version)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Copy all extension files
	if err := ei.copyDirectory(sourcePath, targetDir); err != nil {
		return fmt.Errorf("failed to copy extension files: %w", err)
	}

	fmt.Printf("  ✅ Pre-installed extension: %s (ID: %s, Version: %s)\n", 
		manifest.Name, extensionID, manifest.Version)

	return nil
}

// generateExtensionID generates a Chrome extension ID
func (ei *ExtensionInstaller) generateExtensionID(extensionPath string) string {
	// In real Chrome, ID is generated from the public key
	// For simplicity, we'll use a hash of the path + manifest
	// This creates a consistent ID for the same extension
	
	manifestPath := filepath.Join(extensionPath, "manifest.json")
	manifestData, _ := os.ReadFile(manifestPath)
	
	// Simple hash based on path and manifest content
	hashSource := extensionPath + string(manifestData)
	return ei.simpleHash(hashSource)
}

// simpleHash creates a simple hash for extension ID
func (ei *ExtensionInstaller) simpleHash(input string) string {
	// Create a 32-character ID using a simple hash
	const chars = "abcdefghijklmnopqrstuvwxyz"
	
	// Ensure input is not empty
	if len(input) == 0 {
		input = "default"
	}
	
	hash := 0
	for _, c := range input {
		hash = hash*31 + int(c)
	}
	
	// Ensure hash is positive
	if hash < 0 {
		hash = -hash
	}
	
	result := make([]byte, 32)
	for i := 0; i < 32; i++ {
		result[i] = chars[hash%len(chars)]
		hash = (hash / len(chars)) + i + 1
		if hash == 0 {
			hash = int(input[i%len(input)]) + i + 1
		}
	}
	
	return string(result)
}

// extractZip extracts a ZIP file to a directory
func (ei *ExtensionInstaller) extractZip(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {
		path := filepath.Join(destDir, file.Name)
		
		// Security check: prevent zip slip
		if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.FileInfo().Mode())
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()

		_, err = io.Copy(targetFile, fileReader)
		if err != nil {
			return err
		}
	}

	return nil
}

// copyDirectory recursively copies a directory
func (ei *ExtensionInstaller) copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate the target path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			// Create directory but don't return - continue walking
			return os.MkdirAll(targetPath, info.Mode())
		}

		// Copy file
		return ei.copyFile(path, targetPath)
	})
}

// copyFile copies a single file
func (ei *ExtensionInstaller) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// CreateExtensionsPreferences updates the extensions preference file
func (ei *ExtensionInstaller) CreateExtensionsPreferences(extensionPaths []string) error {
	if len(extensionPaths) == 0 {
		return nil
	}

	prefsDir := filepath.Join(ei.userDataDir, "Default")
	if err := os.MkdirAll(prefsDir, 0755); err != nil {
		return err
	}

	prefsPath := filepath.Join(prefsDir, "Preferences")
	
	// First, check if extensions are actually installed in the Extensions directory
	extensionsDir := filepath.Join(ei.userDataDir, "Default", "Extensions")
	if _, err := os.Stat(extensionsDir); err != nil {
		return fmt.Errorf("Extensions directory does not exist: %w", err)
	}
	
	// List what extensions are actually installed
	entries, err := os.ReadDir(extensionsDir)
	if err != nil {
		return fmt.Errorf("Failed to read Extensions directory: %w", err)
	}
	
	if len(entries) == 0 {
		return fmt.Errorf("No extensions found in Extensions directory")
	}
	
	fmt.Printf("  🔍 发现 %d 个已安装扩展在目录中\n", len(entries))
	
	// 读取现有的Preferences文件（如果存在）
	var preferences map[string]interface{}
	if data, err := os.ReadFile(prefsPath); err == nil {
		// 如果文件存在，解析它
		if err := json.Unmarshal(data, &preferences); err != nil {
			fmt.Printf("警告: 无法解析现有Preferences文件: %v\n", err)
			preferences = make(map[string]interface{})
		}
	} else {
		// 如果文件不存在，创建新的
		preferences = make(map[string]interface{})
	}

	// 确保extensions部分存在
	if _, exists := preferences["extensions"]; !exists {
		preferences["extensions"] = map[string]interface{}{
			"settings": map[string]interface{}{},
		}
	}

	extensionsMap := preferences["extensions"].(map[string]interface{})
	if _, exists := extensionsMap["settings"]; !exists {
		extensionsMap["settings"] = map[string]interface{}{}
	}

	settings := extensionsMap["settings"].(map[string]interface{})

	// Process each installed extension directory
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		extensionID := entry.Name()
		extensionDir := filepath.Join(extensionsDir, extensionID)
		
		// Find version directories within this extension
		versionEntries, err := os.ReadDir(extensionDir)
		if err != nil {
			continue
		}
		
		for _, versionEntry := range versionEntries {
			if !versionEntry.IsDir() {
				continue
			}
			
			// Read manifest from the version directory
			manifestPath := filepath.Join(extensionDir, versionEntry.Name(), "manifest.json")
			if _, err := os.Stat(manifestPath); err != nil {
				continue
			}
			
			manifestData, err := os.ReadFile(manifestPath)
			if err != nil {
				continue
			}
			
			var manifest ExtensionManifest
			if err := json.Unmarshal(manifestData, &manifest); err != nil {
				continue
			}
			
			// Add extension to preferences using the actual directory name as ID
			settings[extensionID] = map[string]interface{}{
				"state":                    1, // ENABLED
				"was_installed_by_default": false,
				"was_installed_by_oem":     false,
				"install_time":             "13000000000000000", // Chrome timestamp
				"from_webstore":           false,
				"was_installed_by_custodian": false,
				"path":                     filepath.Join(extensionDir, versionEntry.Name()),
			}
			
			fmt.Printf("  📝 已注册扩展到Preferences: %s (ID: %s)\n", manifest.Name, extensionID)
			break // Only process the first version found
		}
	}

	// Write preferences file back
	prefsData, err := json.MarshalIndent(preferences, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(prefsPath, prefsData, 0644)
}