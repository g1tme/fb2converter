// +build windows

package config

import (
	"os"
	"path/filepath"
	"strings"
)

// kindlegen provides OS specific part of default kindlegen location
func kindlegen() string {
	return "kindlegen.exe"
}

var toRemove = `.<>":/\|?*` + string(os.PathSeparator) + string(os.PathListSeparator)

// CleanFileName removes not allowed characters form file name.
func CleanFileName(in string) string {
	return strings.Map(func(sym rune) rune {
		if strings.IndexRune(toRemove, sym) != -1 {
			return -1
		}
		return sym
	}, in)
}

// FindConverter attempts to find main conversion engine - myhomelib support.
func FindConverter(expath string) string {

	expath, err := os.Executable()
	if err != nil {
		return ""
	}

	wd := filepath.Dir(expath)

	paths := []string{
		filepath.Join(wd, "fb2c.exe"),                               // `pwd`
		filepath.Join(filepath.Dir(wd), "fb2converter", "fb2c.exe"), // `pwd`/../fb2converter
	}

	for _, p := range paths {
		if _, err = os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
