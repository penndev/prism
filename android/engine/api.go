package engine

// Options is passed to Start. Proxy is a desktop-style URL:
// socks5://user:pass@host:port (also socks5overtls / http / httpovertls).
type Options struct {
	FD      int32
	MTU     int32
	Proxy   string
	Handler Handler
}

// Handler is implemented on the Android side.
// Protect must wrap VpnService.protect.
// UseProxy decides direct vs proxy; Java looks up the IP (leaf → parent) and matches saved area IDs.
// OnProxyRead / OnProxyWrite report proxy-path byte deltas about once a second.
type Handler interface {
	Protect(fd int32) bool
	OnLog(line string)
	UseProxy(address string) bool
	OnProxyRead(n int64)
	OnProxyWrite(n int64)
}

// Area is a node in the IP-region tree. ParentID 0 is a top-level region.
type Area struct {
	ID       int64
	ParentID int64
	Name     string
}

// AreaList is a gomobile-friendly wrapper; slices of pointers are skipped by gobind.
type AreaList struct {
	items []*Area
}

func (l *AreaList) Len() int64 {
	if l == nil {
		return 0
	}
	return int64(len(l.items))
}

func (l *AreaList) Get(i int64) *Area {
	if l == nil || i < 0 || int(i) >= len(l.items) {
		return nil
	}
	return l.items[int(i)]
}

// DbStatus is the current ipregion.db file and metadata.
type DbStatus struct {
	Exists  bool
	Path    string
	Size    int64
	Version string
	Remark  string
	Areas   int64
	V4      int64
	V6      int64
}
