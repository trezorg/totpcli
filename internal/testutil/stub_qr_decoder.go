package testutil

// StubQRDecoder is a test double for importer.QRDecoder.
type StubQRDecoder struct {
	URI string
	Err error
}

func (s StubQRDecoder) Decode(string) (string, error) {
	if s.Err != nil {
		return "", s.Err
	}
	return s.URI, nil
}
