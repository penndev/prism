//go:build !windows && !darwin

package route

func interfaceDNS(int) []string { return nil }

func defaultRoutes(bool) []routeCandidate { return nil }

func listenRouteChanges() (cancel func()) {
	done := make(chan struct{})
	go pollRouteChanges(done)
	return func() { close(done) }
}
