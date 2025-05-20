package sdk

import (
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"io"
)

type PlakarImporterSDK struct {
	scan                       func() (any, error)
	NewReader                  func(pathname string) (io.ReadCloser, error)
	NewExtendedAttributeReader func(pathname string, attribute string) (io.ReadCloser, error)
	GetExtendedAttributes      func(pathname string) ([]importer.ExtendedAttributes, error)
	Close                      func() error
	Root                       func() string
	Origin                     func() string
	Type                       func() string

	importerName string
}

func (p *PlakarImporterSDK) SetScan(scanFunc func() (any, error)) {
	p.scan = scanFunc
}

func (p *PlakarImporterSDK) SetNewReader(readerFunc func(pathname string) (io.ReadCloser, error)) {
	p.NewReader = readerFunc
}

func (p *PlakarImporterSDK) SetNewExtendedAttributeReader(readerFunc func(pathname string, attribute string) (io.ReadCloser, error)) {
	p.NewExtendedAttributeReader = readerFunc
}

func (p *PlakarImporterSDK) SetGetExtendedAttributes(getFunc func(pathname string) ([]importer.ExtendedAttributes, error)) {
	p.GetExtendedAttributes = getFunc
}

func (p *PlakarImporterSDK) SetClose(closeFunc func() error) {
	p.Close = closeFunc
}

func (p *PlakarImporterSDK) SetRoot(rootFunc func() string) {
	p.Root = rootFunc
}

func (p *PlakarImporterSDK) SetOrigin(originFunc func() string) {
	p.Origin = originFunc
}

func (p *PlakarImporterSDK) SetType(typeFunc func() string) {
	p.Type = typeFunc
}
