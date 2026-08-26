// 封装类似 sqlite 的 bbolt 数据库给前端持久化数据使用。
package storage

import (
	"desktop/internal/lang"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"go.etcd.io/bbolt"
)

const bucketName = "data"

type Storage struct {
	db *bbolt.DB
}

func AppDir() (string, error) {
	upath, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dbDir := filepath.Join(upath, appDirName)
	if err := os.MkdirAll(dbDir, 0700); err != nil {
		return "", err
	}
	return dbDir, nil
}

func (s *Storage) SetSettings(v Settings) error {
	return s.putJSON(KeySettings, v)
}

// GetSettings 无记录时返回 nil, nil。
func (s *Storage) GetSettings() (*Settings, error) {
	var out Settings
	ok, err := s.getJSON(KeySettings, &out)
	if err != nil || !ok {
		return nil, err
	}
	if out.System.Language == "" {
		out.System.Language = lang.DefaultLang.CurrentLocale()
	}
	return &out, nil
}

func (s *Storage) SetServers(servers []ServerEntry) error {
	if servers == nil {
		servers = []ServerEntry{}
	}
	return s.putJSON(KeyServers, servers)
}

// GetServers 无记录时返回空切片。
func (s *Storage) GetServers() ([]ServerEntry, error) {
	var out []ServerEntry
	ok, err := s.getJSON(KeyServers, &out)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []ServerEntry{}, nil
	}
	return out, nil
}

func (s *Storage) SetRuleConfig(v RuleConfig) error {
	return s.putJSON(KeyRule, v)
}

// GetRuleConfig 无记录时返回 nil, nil。
func (s *Storage) GetRuleConfig() (*RuleConfig, error) {
	var out RuleConfig
	ok, err := s.getJSON(KeyRule, &out)
	if err != nil || !ok {
		return nil, err
	}
	return &out, nil
}

var ruleAreaNames func(ids []uint32) []string

// SetRuleAreaNames 由 ipregion 注册展示名解析，避免 storage 反向依赖 ipregion。
func SetRuleAreaNames(fn func(ids []uint32) []string) {
	ruleAreaNames = fn
}

func (s *Storage) GetRuleStatus() RuleStatus {
	out := RuleStatus{
		AreaMode:  "global",
		AreaNames: []string{},
		Domains:   []string{},
	}
	cfg, err := s.GetRuleConfig()
	if err != nil || cfg == nil {
		return out
	}
	switch cfg.AreaMode {
	case "proxy", "bypass":
		out.AreaMode = cfg.AreaMode
		if ruleAreaNames != nil {
			out.AreaNames = ruleAreaNames(cfg.AreaIDs)
		}
	case "none":
		out.AreaMode = cfg.AreaMode
	}
	if len(cfg.Domains) > 0 {
		out.Domains = cfg.Domains
	}
	return out
}

func (s *Storage) SetSelectedServer(v ServerEntry) error {
	return s.putJSON(KeySelectedServer, v)
}

// ClearSelectedServer 清除已保存的当前连接节点。
func (s *Storage) ClearSelectedServer() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		return b.Delete([]byte(KeySelectedServer))
	})
}

// GetSelectedServer 无记录时返回 nil, nil。
func (s *Storage) GetSelectedServer() (*ServerEntry, error) {
	var out ServerEntry
	ok, err := s.getJSON(KeySelectedServer, &out)
	if err != nil || !ok {
		return nil, err
	}
	return &out, nil
}

func (s *Storage) putJSON(key string, v any) error {
	if key == "" {
		return errors.New("key不能为空")
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		return b.Put([]byte(key), data)
	})
}

func (s *Storage) getJSON(key string, dest any) (found bool, err error) {
	if key == "" {
		return false, errors.New("key不能为空")
	}
	err = s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		v := b.Get([]byte(key))
		if v == nil {
			return nil
		}
		found = true
		data := make([]byte, len(v))
		copy(data, v)
		return json.Unmarshal(data, dest)
	})
	return found, err
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func New() (*Storage, error) {
	dbDir, err := AppDir()
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dbDir, "data.db")
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Storage{db: db}, nil
}

var DefaultStorage *Storage
