package util

import "os"

// IsExist kiểm tra file/thư mục tồn tại; không tồn tại không phải lỗi I/O.
func IsExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil || os.IsExist(err) {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
