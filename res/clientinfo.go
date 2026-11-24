package res

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

type ClientInfo struct {
	ServiceType string
	Connections []Connection
}

type Connection struct {
	Display         string
	Description     string
	Address         string
	Port            int
	Version         int
	LangType        int
	RegistrationWeb string
	AdminList       []uint32
}

// serviceTypes are the <servicetype> values the classic client accepts
// (SetOption); the comparison is exact.
var serviceTypes = map[string]bool{
	"korea": true, "america": true, "japan": true, "china": true, "taiwan": true,
	"thai": true, "indonesia": true, "philippine": true, "malaysia": true,
	"singapore": true, "germany": true, "india": true, "brazil": true,
	"australia": true, "russia": true, "vietnam": true, "chile": true, "france": true,
}

// Service is the service type the classic client runs as. It starts as
// Korea and keeps that when clientinfo has no or an unknown <servicetype>.
func (c ClientInfo) Service() string {
	if serviceTypes[c.ServiceType] {
		return c.ServiceType
	}
	return "korea"
}

func ParseClientInfo(data []byte) (ClientInfo, error) {
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})

	var raw rawClientInfo
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = clientInfoCharsetReader
	if err := decoder.Decode(&raw); err != nil {
		return ClientInfo{}, err
	}

	out := ClientInfo{ServiceType: strings.TrimSpace(raw.ServiceType)}
	for _, rawConn := range raw.Connections {
		conn := Connection{
			Display:         strings.TrimSpace(rawConn.Display),
			Description:     strings.TrimSpace(rawConn.Description),
			Address:         strings.TrimSpace(rawConn.Address),
			Port:            parseIntDefault(rawConn.Port, 6900),
			Version:         parseIntDefault(rawConn.Version, 55),
			LangType:        parseIntDefault(rawConn.LangType, 0),
			RegistrationWeb: strings.TrimSpace(rawConn.RegistrationWeb),
			AdminList:       parseClientInfoAdminList(rawConn),
		}
		if conn.Address == "" {
			continue
		}
		if conn.Display == "" {
			conn.Display = conn.Address
		}
		out.Connections = append(out.Connections, conn)
	}

	if len(out.Connections) == 0 {
		return out, fmt.Errorf("clientinfo has no usable connection entries")
	}
	return out, nil
}

func clientInfoCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(strings.TrimSpace(charset)) {
	case "utf-8", "utf8":
		return input, nil
	case "euc-kr", "ks_c_5601-1987", "cp949", "uhc":
		return transform.NewReader(input, korean.EUCKR.NewDecoder()), nil
	default:
		return nil, fmt.Errorf("unsupported clientinfo charset %q", charset)
	}
}

func parseIntDefault(value string, fallback int) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return i
}

func parseClientInfoAdminList(raw rawConnection) []uint32 {
	seen := make(map[uint32]struct{})
	out := make([]uint32, 0, len(raw.Yellow.Admins)+len(raw.AID.Admins))
	appendAdmins := func(values []string) {
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			id64, err := strconv.ParseUint(value, 10, 32)
			if err != nil || id64 == 0 {
				continue
			}
			id := uint32(id64)
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	appendAdmins(raw.Yellow.Admins)
	appendAdmins(raw.AID.Admins)
	return out
}

type rawClientInfo struct {
	XMLName     xml.Name        `xml:"clientinfo"`
	ServiceType string          `xml:"servicetype"`
	Connections []rawConnection `xml:"connection"`
}

type rawConnection struct {
	Display         string       `xml:"display"`
	Description     string       `xml:"desc"`
	Address         string       `xml:"address"`
	Port            string       `xml:"port"`
	Version         string       `xml:"version"`
	LangType        string       `xml:"langtype"`
	RegistrationWeb string       `xml:"registrationweb"`
	AID             rawAdminList `xml:"aid"`
	Yellow          rawAdminList `xml:"yellow"`
}

type rawAdminList struct {
	Admins []string `xml:"admin"`
}
