package sanity

import "context"

type fakeCryptMapper struct {
	deviceName string
}

func (s *fakeCryptMapper) CloseCryptDevice(volumeID string) error {
	return nil
}

func (s *fakeCryptMapper) OpenCryptDevice(ctx context.Context, source, volumeID string, integrity bool) (string, error) {
	return "/dev/mapper/" + volumeID, nil
}

func (s *fakeCryptMapper) ResizeCryptDevice(ctx context.Context, volumeID string) (string, error) {
	return s.deviceName, nil
}

func (s *fakeCryptMapper) GetDevicePath(volumeID string) (string, error) {
	return s.deviceName, nil
}

func fakeEvalSymlinks(path string) (string, error) {
	return path, nil
}
