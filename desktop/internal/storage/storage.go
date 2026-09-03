// 封装类似 sqlite 的 bbolt 数据库给前端持久化数据使用。
package storage

import (
	"desktop/internal/lang"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"

	"go.etcd.io/bbolt"
)

const (
	bucketName     = "data"
	IpregionDBName = "ipregion.db"
)

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

func IpregionDBPath() (string, error) {
	dir, err := AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, IpregionDBName), nil
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
	normalized := normalizeServers(servers)
	seen := make(map[string]struct{}, len(normalized))
	out := make([]ServerEntry, 0, len(normalized))
	for _, v := range normalized {
		// 和前端 _id 同一套身份：协议+账号+地址相同就是同一条，后写的丢掉。
		key := v.Protocol + "://" + v.Username + ":" + v.Password + "@" + v.Host
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	return s.putJSON(KeyServers, out)
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
	return normalizeServers(out), nil
}

// ruleCache 让 CachedRuleConfig 不必每次都开 bbolt 事务，写入时由 SetRuleConfig 刷新。
var ruleCache atomic.Pointer[RuleConfig]

func (s *Storage) SetRuleConfig(v RuleConfig) error {
	if err := s.putJSON(KeyRule, v); err != nil {
		return err
	}
	ruleCache.Store(&v)
	return nil
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

// CachedRuleConfig 供连接路径使用：每条连接都要判地域规则，
// 走 GetRuleConfig 就是每条连接一次 bbolt 只读事务。
// 返回的 *RuleConfig 是共享只读副本，调用方不要改。
func (s *Storage) CachedRuleConfig() *RuleConfig {
	if c := ruleCache.Load(); c != nil {
		return c
	}
	cfg, err := s.GetRuleConfig()
	if err != nil {
		return nil
	}
	if cfg == nil {
		// 没存过配置时也要缓存一个空值，否则每条连接还是会去查库
		cfg = &RuleConfig{}
	}
	ruleCache.Store(cfg)
	return cfg
}

func (s *Storage) SetSelectedServer(v ServerEntry) error {
	return s.putJSON(KeySelectedServer, normalizeServer(v))
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
	out = normalizeServer(out)
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
