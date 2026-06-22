package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

type FileConflictStrategy int

const (
	OVERWRITE = iota
	SKIP
)

type CopyResult int

const (
	CopyResultUnknown = iota
	CopyResultCreated
	CopyResultSkipped
	CopyResultOverwritten
)

func copyFile(src, dest string, strategy FileConflictStrategy) (CopyResult, error) {
	srcExists, err := fileExists(src)
	if err != nil {
		return 0, err
	}
	if !srcExists {
		return 0, fmt.Errorf("source file does not exist: %s", src)
	}

	destExists, err := fileExists(dest)
	if err != nil {
		return 0, err
	}

	if destExists {
		switch strategy {
		case OVERWRITE:
			if err := writeCopy(src, dest); err != nil {
				return 0, err
			}
			return CopyResultOverwritten, nil
		case SKIP:
			return CopyResultSkipped, nil
		}
	}

	if err := writeCopy(src, dest); err != nil {
		return 0, err
	}
	return CopyResultCreated, nil
}

func writeCopy(src, dest string) error {
	fin, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fin.Close()

	buf := make([]byte, 1024)
	fout, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer fin.Close()

	for {
		n, err := fin.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		if _, err := fout.Write(buf[:n]); err != nil {
			return err
		}
	}
	return nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err == nil {
		return true, nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, err
}
