//go:build windows

package fsx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

const (
	// IOReparseTagMountPoint is the reparse tag for directory junctions (mklink /J).
	IOReparseTagMountPoint = 0xA0000003
	ioReparseTagSymlink    = 0xA000000C
)

const (
	fsctlGetReparsePoint = windows.FSCTL_GET_REPARSE_POINT
	fsctlSetReparsePoint = windows.FSCTL_SET_REPARSE_POINT
	maxReparseBuffer     = 16 * 1024
)

type reparseHeader struct {
	ReparseTag           uint32
	ReparseDataLength    uint16
	Reserved             uint16
	SubstituteNameOffset uint16
	SubstituteNameLength uint16
	PrintNameOffset      uint16
	PrintNameLength      uint16
}

// ReparseTag returns the reparse point tag for path, or 0 when absent or unreadable.
func ReparseTag(path string) uint32 {
	_, _, tag, err := readReparsePoint(path)
	if err != nil {
		return 0
	}
	return tag
}

// ReadMountPoint reads substitute and print names from a directory junction.
func ReadMountPoint(path string) (substitute, print string, tag uint32, err error) {
	substitute, print, tag, err = readReparsePoint(path)
	if err != nil {
		return "", "", 0, err
	}
	if tag != IOReparseTagMountPoint {
		return "", "", tag, fmt.Errorf("fsx: not a mount point (tag 0x%08X)", tag)
	}
	return substitute, print, tag, nil
}

// CreateMountPoint creates a directory junction at link using mount-point reparse data.
func CreateMountPoint(link, substitute, print string) error {
	if substitute == "" {
		return fmt.Errorf("fsx: empty mount point substitute name")
	}
	if print == "" {
		print = substitute
		if strings.HasPrefix(substitute, `\??\`) {
			print = substitute[4:]
		}
	}
	if err := os.RemoveAll(link); err != nil {
		return err
	}
	if err := os.Mkdir(link, 0); err != nil {
		return err
	}
	buf := encodeMountPointReparse(substitute, print)
	pathPtr, err := windows.UTF16PtrFromString(link)
	if err != nil {
		if rmErr := os.RemoveAll(link); rmErr != nil {
			return fmt.Errorf("%w (cleanup: %v)", err, rmErr)
		}
		return err
	}
	handle, err := windows.CreateFile(
		pathPtr,
		windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		if rmErr := os.RemoveAll(link); rmErr != nil {
			return fmt.Errorf("%w (cleanup: %v)", err, rmErr)
		}
		return err
	}
	defer func() { _ = windows.CloseHandle(handle) }()
	var returned uint32
	if err := windows.DeviceIoControl(
		handle,
		fsctlSetReparsePoint,
		&buf[0],
		uint32(len(buf)),
		nil,
		0,
		&returned,
		nil,
	); err != nil {
		if rmErr := os.RemoveAll(link); rmErr != nil {
			return fmt.Errorf("%w (cleanup: %v)", err, rmErr)
		}
		return err
	}
	return nil
}

func readReparsePoint(path string) (substitute, print string, tag uint32, err error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", "", 0, err
	}
	handle, err := windows.CreateFile(
		pathPtr,
		0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return "", "", 0, err
	}
	defer func() { _ = windows.CloseHandle(handle) }()

	buf := make([]byte, maxReparseBuffer)
	var returned uint32
	if err := windows.DeviceIoControl(
		handle,
		fsctlGetReparsePoint,
		nil,
		0,
		&buf[0],
		uint32(len(buf)),
		&returned,
		nil,
	); err != nil {
		return "", "", 0, err
	}
	if int(returned) < 8 {
		return "", "", 0, fmt.Errorf("fsx: short reparse buffer")
	}
	tag = binary.LittleEndian.Uint32(buf[0:4])
	dataLen := binary.LittleEndian.Uint16(buf[4:6])
	if int(dataLen)+8 > int(returned) {
		return "", "", tag, fmt.Errorf("fsx: invalid reparse data length")
	}
	switch tag {
	case IOReparseTagMountPoint:
		substitute, print, err = parseMountPointNames(buf[8 : 8+dataLen])
		return substitute, print, tag, err
	case ioReparseTagSymlink:
		substitute, print, err = parseSymlinkNames(buf[8 : 8+dataLen])
		return substitute, print, tag, err
	default:
		return "", "", tag, nil
	}
}

func parseMountPointNames(data []byte) (substitute, print string, err error) {
	if len(data) < 8 {
		return "", "", fmt.Errorf("fsx: short mount point buffer")
	}
	subOff := binary.LittleEndian.Uint16(data[0:2])
	subLen := binary.LittleEndian.Uint16(data[2:4])
	printOff := binary.LittleEndian.Uint16(data[4:6])
	printLen := binary.LittleEndian.Uint16(data[6:8])
	pathBuf := data[8:]
	substitute, err = utf16BytesToString(pathBuf, subOff, subLen)
	if err != nil {
		return "", "", err
	}
	print, err = utf16BytesToString(pathBuf, printOff, printLen)
	return substitute, print, err
}

func parseSymlinkNames(data []byte) (substitute, print string, err error) {
	if len(data) < 12 {
		return "", "", fmt.Errorf("fsx: short symlink buffer")
	}
	printOff := binary.LittleEndian.Uint16(data[4:6])
	printLen := binary.LittleEndian.Uint16(data[6:8])
	pathBuf := data[12:]
	print, err = utf16BytesToString(pathBuf, printOff, printLen)
	return "", print, err
}

func utf16BytesToString(buf []byte, offset, length uint16) (string, error) {
	start := int(offset)
	end := start + int(length)
	if start < 0 || end > len(buf) || start > end {
		return "", fmt.Errorf("fsx: invalid utf16 span")
	}
	if length == 0 {
		return "", nil
	}
	u16 := make([]uint16, 0, length/2)
	for i := start; i+1 < end; i += 2 {
		r := binary.LittleEndian.Uint16(buf[i:])
		if r == 0 {
			break
		}
		u16 = append(u16, r)
	}
	return string(utf16.Decode(u16)), nil
}

func encodeMountPointReparse(substitute, print string) []byte {
	subUTF16 := utf16.Encode([]rune(substitute + "\x00"))
	printUTF16 := utf16.Encode([]rune(print + "\x00"))
	dataLen := 8 + len(subUTF16)*2 + len(printUTF16)*2
	header := reparseHeader{
		ReparseTag:           IOReparseTagMountPoint,
		ReparseDataLength:    uint16(dataLen),
		SubstituteNameOffset: 0,
		SubstituteNameLength: uint16((len(subUTF16) - 1) * 2),
		PrintNameOffset:      uint16(len(subUTF16) * 2),
		PrintNameLength:      uint16((len(printUTF16) - 1) * 2),
	}
	var b bytes.Buffer
	_ = binary.Write(&b, binary.LittleEndian, &header)
	_ = binary.Write(&b, binary.LittleEndian, subUTF16)
	_ = binary.Write(&b, binary.LittleEndian, printUTF16)
	return b.Bytes()
}
