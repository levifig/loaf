//go:build !darwin && !linux && !freebsd && !openbsd && !netbsd && !dragonfly

package devtool

type devLinkLock struct{}

func acquireDevLinkLock(string) (*devLinkLock, error) {
	return &devLinkLock{}, nil
}

func releaseDevLinkLock(*devLinkLock) error {
	return nil
}
