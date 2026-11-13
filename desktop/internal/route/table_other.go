//go:build !windows && !darwin

package route

func interfaceDNS(int) []string { return nil }

func defaultRoutes(bool) []routeCandidate { return nil }

func listenRouteChanges() (cancel func()) {
	return startRoutePoll()
}
