//go:build windows

package main

import (
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	shell32            = syscall.NewLazyDLL("shell32.dll")
	shellExecute       = shell32.NewProc("ShellExecuteW")
	pLoadImageAbout    = u32.NewProc("LoadImageW")
	pDeleteBitmapAbout = g32.NewProc("DeleteObject")
	aboutAvatarBitmap  uintptr
)

func aboutOpenURL(address string) {
	r, _, _ := shellExecute.Call(hwnd, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr(address))), 0, 0, 1)
	if r <= 32 {
		message("Не удалось открыть ссылку: "+address, true)
	}
}
func aboutActions(id int) bool {
	switch id {
	case 510:
		aboutOpenURL("https://band.link/UylkI")
	case 511:
		aboutOpenURL("https://tips.yandex.ru/guest/payment/3749434")
	case 512:
		aboutOpenURL("mailto:mincka@yandex.ru")
	default:
		return false
	}
	return true
}
func aboutAvatar(x, y int) {
	path := filepath.Join(storageRoot(), "Resources", "author.bmp")
	aboutAvatarBitmap, _, _ = pLoadImageAbout.Call(0, uintptr(unsafe.Pointer(ptr(path))), 0, 0, 0, 0x10 /* LR_LOADFROMFILE */)
	if aboutAvatarBitmap == 0 {
		label("[Аватар автора]", x, y, 160, 25)
		return
	}
	h := create("STATIC", "", WS_CHILD|WS_VISIBLE|0x000E /* SS_BITMAP */, x, y, 132, 132, 0)
	pSend.Call(h, 0x0172 /* STM_SETIMAGE */, 0 /* IMAGE_BITMAP */, aboutAvatarBitmap)
}
func aboutCleanUp() {
	if aboutAvatarBitmap != 0 {
		pDeleteBitmapAbout.Call(aboutAvatarBitmap)
		aboutAvatarBitmap = 0
	}
}
