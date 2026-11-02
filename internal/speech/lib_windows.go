//go:build windows

package speech

import "syscall"

// openLibrary loads the pre-bundled DLL from the app directory.
func openLibrary(path string) (uintptr, error) {
	h, err := syscall.LoadLibrary(path)
	return uintptr(h), err
}

func librarySymbol(h uintptr, name string) (uintptr, error) {
	return syscall.GetProcAddress(syscall.Handle(h), name)
}
