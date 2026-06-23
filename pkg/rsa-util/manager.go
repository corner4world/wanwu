package rsautil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/UnicomAI/wanwu/pkg/log"
)

// KeyPair 表示一个RSA密钥对
type KeyPair struct {
	KeyID      string
	PublicKey  *rsa.PublicKey
	PrivateKey *rsa.PrivateKey
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

// KeyManager RSA密钥管理器
type KeyManager struct {
	currentKey *KeyPair
	keyCache   map[string]*KeyPair
	mu         sync.RWMutex
	config     Config
}

// Config RSA密钥配置
type Config struct {
	// 私钥文件路径
	PrivateKeyPath string `json:"private_key_path" mapstructure:"private_key_path"`
	// 公钥文件路径
	PublicKeyPath string `json:"public_key_path" mapstructure:"public_key_path"`
	// 密钥轮换周期（天），默认90天
	KeyRotationDays int `json:"key_rotation_days" mapstructure:"key_rotation_days"`
}

var (
	manager *KeyManager
	once    sync.Once
)

// InitKeyManager 初始化密钥管理器（单例）
func InitKeyManager(config Config) error {
	var initErr error
	once.Do(func() {
		manager = &KeyManager{
			keyCache: make(map[string]*KeyPair),
			config:   config,
		}
		initErr = manager.loadFromFile()
	})
	return initErr
}

// GetManager 获取密钥管理器实例
func GetManager() *KeyManager {
	if manager == nil {
		log.Panicf("rsa key manager not initialized, please call InitKeyManager first")
	}
	return manager
}

// loadFromFile 从文件加载密钥对
func (km *KeyManager) loadFromFile() error {
	if km.config.PrivateKeyPath == "" {
		return errors.New("private key path is empty")
	}

	// 读取私钥文件
	privateKeyData, err := os.ReadFile(km.config.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("read private key file failed: %w", err)
	}

	// 解析 PEM 格式
	block, _ := pem.Decode(privateKeyData)
	if block == nil {
		return errors.New("failed to decode PEM block containing private key")
	}

	// 解析 PKCS1 或 PKCS8 格式私钥
	var privateKey *rsa.PrivateKey

	// 尝试 PKCS1
	privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// 尝试 PKCS8
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("parse private key failed: %w", err)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return errors.New("not an RSA private key")
		}
	}

	// 生成密钥ID
	keyID := generateKeyID()

	// 设置过期时间
	rotationDays := km.config.KeyRotationDays
	if rotationDays <= 0 {
		rotationDays = 90 // 默认90天
	}

	keyPair := &KeyPair{
		KeyID:      keyID,
		PublicKey:  &privateKey.PublicKey,
		PrivateKey: privateKey,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().AddDate(0, 0, rotationDays),
	}

	km.mu.Lock()
	defer km.mu.Unlock()
	km.currentKey = keyPair
	km.keyCache[keyID] = keyPair

	log.Infof("RSA key loaded from file, keyID: %s, expires at: %s", keyID, keyPair.ExpiresAt.Format("2006-01-02"))
	return nil
}

// GetPublicKey 获取当前公钥信息（供前端使用）
func (km *KeyManager) GetPublicKey() (keyID string, publicKeyPEM string, expiresIn int64, err error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if km.currentKey == nil {
		return "", "", 0, errors.New("no available RSA key")
	}

	// 检查是否过期
	if time.Now().After(km.currentKey.ExpiresAt) {
		return "", "", 0, errors.New("RSA key has expired")
	}

	// 序列化公钥为 PEM 格式
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(km.currentKey.PublicKey)
	if err != nil {
		return "", "", 0, fmt.Errorf("marshal public key failed: %w", err)
	}

	publicKeyPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}))

	// 建议前端缓存5分钟
	expiresIn = 300

	return km.currentKey.KeyID, publicKeyPEM, expiresIn, nil
}

// Decrypt 使用指定密钥解密数据
func (km *KeyManager) Decrypt(keyID string, ciphertextBase64 string) ([]byte, error) {
	km.mu.RLock()
	keyPair, exists := km.keyCache[keyID]
	km.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("invalid key id: %s", keyID)
	}

	// Base64 解码
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}

	// RSA-OAEP 解密
	hash := sha256.New()
	plaintext, err := rsa.DecryptOAEP(hash, rand.Reader, keyPair.PrivateKey, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("RSA decrypt failed: %w", err)
	}

	return plaintext, nil
}

// DecryptWithCurrentKey 使用当前密钥解密（兼容无keyID的情况）
func (km *KeyManager) DecryptWithCurrentKey(ciphertextBase64 string) ([]byte, error) {
	km.mu.RLock()
	if km.currentKey == nil {
		km.mu.RUnlock()
		return nil, errors.New("no available RSA key")
	}
	keyID := km.currentKey.KeyID
	km.mu.RUnlock()

	return km.Decrypt(keyID, ciphertextBase64)
}

// generateKeyID 生成密钥ID
func generateKeyID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	for i := range b {
		// 使用 crypto/rand 生成随机数
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			// 如果随机数生成失败，使用时间戳作为后备
			b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
			continue
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// RotateKey 手动轮换密钥（用于运维操作）
func (km *KeyManager) RotateKey() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	// 从文件重新加载密钥
	return km.loadFromFile()
}
