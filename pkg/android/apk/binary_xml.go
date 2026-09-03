package apk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// Binary XML chunk types
const (
	RES_NULL_TYPE                = 0x0000
	RES_STRING_POOL_TYPE         = 0x0001
	RES_TABLE_TYPE               = 0x0002
	RES_XML_TYPE                 = 0x0003
	RES_XML_FIRST_CHUNK_TYPE     = 0x0100
	RES_XML_START_NAMESPACE_TYPE = 0x0100
	RES_XML_END_NAMESPACE_TYPE   = 0x0101
	RES_XML_START_ELEMENT_TYPE   = 0x0102
	RES_XML_END_ELEMENT_TYPE     = 0x0103
	RES_XML_CDATA_TYPE           = 0x0104
	RES_XML_LAST_CHUNK_TYPE      = 0x017f
	RES_XML_RESOURCE_MAP_TYPE    = 0x0180
)

// BinaryXMLParser parses Android binary XML format
type BinaryXMLParser struct {
	data        []byte
	strings     []string
	resourceMap []uint32
}

// ParseBinaryXML parses binary XML data
func ParseBinaryXML(data []byte) (*BinaryXMLParser, error) {
	p := &BinaryXMLParser{data: data}

	if len(data) < 8 {
		return nil, fmt.Errorf("data too short")
	}

	// Read header
	chunkType := binary.LittleEndian.Uint16(data[0:2])
	if chunkType != RES_XML_TYPE {
		return nil, fmt.Errorf("not an XML file: type 0x%04x", chunkType)
	}

	// Parse chunks
	offset := 8 // Skip XML header
	for offset < len(data) {
		if offset+8 > len(data) {
			break
		}

		ctype := binary.LittleEndian.Uint16(data[offset : offset+2])
		csize := binary.LittleEndian.Uint32(data[offset+4 : offset+8])

		if offset+int(csize) > len(data) {
			break
		}

		chunk := data[offset : offset+int(csize)]

		switch ctype {
		case RES_STRING_POOL_TYPE:
			if err := p.parseStringPool(chunk); err != nil {
				return nil, fmt.Errorf("parse string pool: %w", err)
			}
		case RES_XML_RESOURCE_MAP_TYPE:
			p.parseResourceMap(chunk)
		}

		offset += int(csize)
	}

	return p, nil
}

// parseStringPool extracts the string pool
func (p *BinaryXMLParser) parseStringPool(data []byte) error {
	if len(data) < 28 {
		return fmt.Errorf("string pool too short")
	}

	stringCount := binary.LittleEndian.Uint32(data[8:12])
	stringsStart := binary.LittleEndian.Uint32(data[20:24])
	flags := binary.LittleEndian.Uint32(data[16:20])

	isUTF8 := (flags & 0x00000100) != 0

	// Read string offsets
	offsets := make([]uint32, stringCount)
	offsetsStart := 28
	for i := uint32(0); i < stringCount; i++ {
		pos := offsetsStart + int(i)*4
		if pos+4 > len(data) {
			break
		}
		offsets[i] = binary.LittleEndian.Uint32(data[pos : pos+4])
	}

	// Extract strings
	p.strings = make([]string, 0, stringCount)
	for _, offset := range offsets {
		pos := int(stringsStart + offset)
		if pos >= len(data) {
			p.strings = append(p.strings, "")
			continue
		}

		var str string
		if isUTF8 {
			str = p.decodeUTF8String(data[pos:])
		} else {
			str = p.decodeUTF16String(data[pos:])
		}
		p.strings = append(p.strings, str)
	}

	return nil
}

// decodeUTF8String decodes a UTF-8 string from binary XML
func (p *BinaryXMLParser) decodeUTF8String(data []byte) string {
	if len(data) < 2 {
		return ""
	}

	// Skip length encoding
	length := int(data[0])
	start := 1
	if (length & 0x80) != 0 {
		if len(data) < 3 {
			return ""
		}
		length = ((length & 0x7F) << 8) | int(data[1])
		start = 2
	}

	// Another length byte for character count
	if start >= len(data) {
		return ""
	}
	start++

	end := start + length
	if end > len(data) {
		end = len(data)
	}

	// Find null terminator
	nullPos := bytes.IndexByte(data[start:end], 0)
	if nullPos >= 0 {
		end = start + nullPos
	}

	return string(data[start:end])
}

// decodeUTF16String decodes a UTF-16 string from binary XML
func (p *BinaryXMLParser) decodeUTF16String(data []byte) string {
	if len(data) < 2 {
		return ""
	}

	length := int(binary.LittleEndian.Uint16(data[0:2]))
	start := 2

	if start+length*2 > len(data) {
		length = (len(data) - start) / 2
	}

	runes := make([]rune, length)
	for i := 0; i < length; i++ {
		pos := start + i*2
		if pos+2 > len(data) {
			break
		}
		runes[i] = rune(binary.LittleEndian.Uint16(data[pos : pos+2]))
	}

	return string(runes)
}

// parseResourceMap extracts the resource map
func (p *BinaryXMLParser) parseResourceMap(data []byte) {
	if len(data) < 8 {
		return
	}

	count := (len(data) - 8) / 4
	p.resourceMap = make([]uint32, count)

	for i := 0; i < count; i++ {
		pos := 8 + i*4
		if pos+4 <= len(data) {
			p.resourceMap[i] = binary.LittleEndian.Uint32(data[pos : pos+4])
		}
	}
}

// GetString returns a string by index
func (p *BinaryXMLParser) GetString(idx int) string {
	if idx < 0 || idx >= len(p.strings) {
		return ""
	}
	return p.strings[idx]
}

// DecodeManifest parses AndroidManifest.xml and extracts key information
func (p *BinaryXMLParser) DecodeManifest() (*ManifestInfo, error) {
	manifest := &ManifestInfo{
		Permissions: []string{},
		Activities:  []string{},
		Services:    []string{},
		Receivers:   []string{},
		Providers:   []string{},
	}

	// Simple state machine to parse the manifest
	// This is a basic implementation - full parsing requires handling all XML events
	offset := 8 // Skip header

	for offset < len(p.data) {
		if offset+8 > len(p.data) {
			break
		}

		ctype := binary.LittleEndian.Uint16(p.data[offset : offset+2])
		csize := binary.LittleEndian.Uint32(p.data[offset+4 : offset+8])

		if offset+int(csize) > len(p.data) {
			break
		}

		if ctype == RES_XML_START_ELEMENT_TYPE {
			p.parseStartElement(p.data[offset:offset+int(csize)], manifest)
		}

		offset += int(csize)
	}

	return manifest, nil
}

// ManifestInfo holds parsed manifest data
type ManifestInfo struct {
	PackageName          string
	VersionName          string
	VersionCode          int64
	MinSDK               int
	TargetSDK            int
	Debuggable           bool
	AllowBackup          bool
	UsesCleartextTraffic bool
	Permissions          []string
	Activities           []string
	Services             []string
	Receivers            []string
	Providers            []string
}

// Attribute value data types (android.util.TypedValue). Only the subset an
// AndroidManifest commonly uses is enumerated; other types fall back to a
// decimal rendering of the raw data.
const (
	attrTypeReference  = 0x01 // resource reference
	attrTypeString     = 0x03 // string pool index in data
	attrTypeIntDec     = 0x10 // decimal integer
	attrTypeIntHex     = 0x11 // hexadecimal integer
	attrTypeIntBoolean = 0x12 // 0 / 0xFFFFFFFF
)

// parseStartElement extracts the element name and attributes from a start
// element chunk. The chunk carries a 24-byte header followed by the
// ResXMLTree_attrExt structure: ns, name, attributeStart, attributeSize,
// attributeCount, idIndex, classIndex, styleIndex. The name and attribute
// table offsets are read from those fields (the old code read them from the
// namespace slot, which broke parsing of real manifests).
func (p *BinaryXMLParser) parseStartElement(data []byte, manifest *ManifestInfo) {
	if len(data) < 36 {
		return
	}

	nameIdx := int(binary.LittleEndian.Uint32(data[20:24]))
	elementName := p.GetString(nameIdx)

	// attributeStart is relative to the start of the attribute extension,
	// which begins after the 16-byte node header.
	attrStart := binary.LittleEndian.Uint16(data[24:26])
	attrCount := binary.LittleEndian.Uint16(data[28:30])

	attrs := p.parseAttributes(data[16+int(attrStart):], int(attrCount))

	// Process based on element name
	switch elementName {
	case "manifest":
		if pkg, ok := attrs["package"]; ok {
			manifest.PackageName = pkg
		}
		if ver, ok := attrs["versionName"]; ok {
			manifest.VersionName = ver
		}
		if vc, ok := attrs["versionCode"]; ok {
			if n, err := strconv.Atoi(strings.TrimSpace(vc)); err == nil {
				manifest.VersionCode = int64(n)
			}
		}

	case "uses-permission":
		if name, ok := attrs["name"]; ok {
			manifest.Permissions = append(manifest.Permissions, name)
		}

	case "uses-sdk":
		if ms, ok := attrs["minSdkVersion"]; ok {
			if n, err := strconv.Atoi(strings.TrimSpace(ms)); err == nil {
				manifest.MinSDK = n
			}
		}
		if ts, ok := attrs["targetSdkVersion"]; ok {
			if n, err := strconv.Atoi(strings.TrimSpace(ts)); err == nil {
				manifest.TargetSDK = n
			}
		}

	case "application":
		if dbg, ok := attrs["debuggable"]; ok {
			manifest.Debuggable = dbg == "true"
		}
		if bkp, ok := attrs["allowBackup"]; ok {
			manifest.AllowBackup = bkp == "true"
		}
		if clr, ok := attrs["usesCleartextTraffic"]; ok {
			manifest.UsesCleartextTraffic = clr == "true"
		}

	case "activity", "activity-alias":
		if name, ok := attrs["name"]; ok {
			manifest.Activities = append(manifest.Activities, name)
		}

	case "service":
		if name, ok := attrs["name"]; ok {
			manifest.Services = append(manifest.Services, name)
		}

	case "receiver", "receiver-alias":
		if name, ok := attrs["name"]; ok {
			manifest.Receivers = append(manifest.Receivers, name)
		}

	case "provider":
		if name, ok := attrs["name"]; ok {
			manifest.Providers = append(manifest.Providers, name)
		}
	}
}

// parseAttributes extracts attributes from an element's attribute table. Each
// attribute is 20 bytes: ns, name (string pool index), rawValue, then an
// 8-byte Res_value (size, res0, dataType, data). The value is decoded from the
// typed data field rather than assuming every value is a string pool index.
func (p *BinaryXMLParser) parseAttributes(data []byte, count int) map[string]string {
	attrs := make(map[string]string)

	for i := 0; i < count; i++ {
		pos := i * 20
		if pos+20 > len(data) {
			break
		}

		attr := data[pos : pos+20]
		name := p.GetString(int(binary.LittleEndian.Uint32(attr[4:8])))
		if name == "" {
			continue
		}

		dataType := attr[15]
		data := binary.LittleEndian.Uint32(attr[16:20])

		var value string
		switch dataType {
		case attrTypeString:
			value = p.GetString(int(data))
		case attrTypeIntBoolean:
			if data != 0 {
				value = "true"
			} else {
				value = "false"
			}
		case attrTypeIntDec, attrTypeIntHex:
			value = strconv.FormatUint(uint64(data), 10)
		default:
			// A resource reference (or an unhandled type): prefer the raw
			// string index when present, otherwise render the data as decimal.
			if raw := binary.LittleEndian.Uint32(attr[8:12]); raw != 0 {
				if str := p.GetString(int(raw)); str != "" {
					value = str
					break
				}
			}
			value = strconv.FormatUint(uint64(data), 10)
		}

		attrs[name] = value
	}

	return attrs
}
