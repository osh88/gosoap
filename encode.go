package gosoap

import (
	"encoding/xml"
	"fmt"
	"bytes"
)

// MarshalXML envelope the body and encode to xml
func (c *Helper) encode(method string, params *Params, headerName string, headerParams *Params) ([]byte, error) {
	//start envelope
	if c.definitions == nil {
		return nil, fmt.Errorf("definitions is nil")
	}

	var err error
	var buf bytes.Buffer
	e := xml.NewEncoder(&buf)

	tokens := []xml.Token{}

	targetNamespace := c.definitions.TargetNamespace
	//if targetNamespace == "" && len(c.Definitions.Types) > 0 && len(c.Definitions.Types[0].XsdSchema) > 0 {
	//	targetNamespace = c.Definitions.Types[0].XsdSchema[0].TargetNamespace
	//}

	tokens = startEnvelope(tokens)

	if headerParams != nil {
		tokens = startHeader(tokens, headerName, targetNamespace)
		for _, p := range *headerParams {
			t := xml.StartElement{
				Name: xml.Name{
					Space: "",
					Local: p.K,
				},
			}

			if p.IsRaw() {
				tokens = append(tokens, t, RawToken(p.GetV()), xml.EndElement{Name: t.Name})
			} else {
				tokens = append(tokens, t, xml.CharData(p.GetV()), xml.EndElement{Name: t.Name})
			}
		}
		tokens = endHeader(tokens, headerName)
	}

	if tokens, err = startBody(tokens, method, targetNamespace); err != nil {
		return nil, err
	}

	if params != nil {
		for _, p := range *params {
			t := xml.StartElement{
				Name: xml.Name{
					Space: "",
					Local: p.K,
				},
			}

			if p.IsRaw() {
				tokens = append(tokens, t, RawToken(p.GetV()), xml.EndElement{Name: t.Name})
			} else {
				tokens = append(tokens, t, xml.CharData(p.GetV()), xml.EndElement{Name: t.Name})
			}
		}
	}

	tokens = endBody(tokens, method)
	tokens = endEnvelope(tokens)

	for _, t := range tokens {
		switch v := t.(type) {
		case RawToken:
			if err := e.Flush(); err != nil {
				return nil, err
			}
			if _, err := buf.WriteString(string(v)); err != nil {
				return nil, err
			}
		default:
			if err := e.EncodeToken(t); err != nil {
				return nil, err
			}
		}
	}

	if err := e.Flush(); err != nil {
		return nil, err
	} else {
		return buf.Bytes(), nil
	}
}

func startEnvelope(tokens []xml.Token) []xml.Token {
	e := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: "soap:Envelope",
		},
		Attr: []xml.Attr{
			{Name: xml.Name{Space: "", Local: "xmlns:xsi"}, Value: "http://www.w3.org/2001/XMLSchema-instance"},
			{Name: xml.Name{Space: "", Local: "xmlns:xsd"}, Value: "http://www.w3.org/2001/XMLSchema"},
			{Name: xml.Name{Space: "", Local: "xmlns:soap"}, Value: "http://schemas.xmlsoap.org/soap/envelope/"},
		},
	}

	return append(tokens, e)
}

func endEnvelope(tokens []xml.Token) []xml.Token {
	e := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: "soap:Envelope",
		},
	}

	return append(tokens, e)
}

func startHeader(tokens []xml.Token, m, n string) []xml.Token {
	h := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: "soap:Header",
		},
	}

	if m == "" || n == "" {
		return append(tokens, h)
	}

	r := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: m,
		},
		Attr: []xml.Attr{
			{Name: xml.Name{Space: "", Local: "xmlns"}, Value: n},
		},
	}

	return append(tokens, h, r)
}

func endHeader(tokens []xml.Token, m string) []xml.Token {
	h := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: "soap:Header",
		},
	}

	if m == "" {
		return append(tokens, h)
	}

	r := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: m,
		},
	}

	return append(tokens, r, h)
}

// startToken initiate body of the envelope
func startBody(tokens []xml.Token, m, n string) ([]xml.Token, error) {
	b := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: "soap:Body",
		},
	}

	if m == "" || n == "" {
		return nil, fmt.Errorf("method or namespace is empty")
	}

	r := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: m,
		},
		Attr: []xml.Attr{
			{Name: xml.Name{Space: "", Local: "xmlns"}, Value: n},
		},
	}

	return append(tokens, b, r), nil
}

// endToken close body of the envelope
func endBody(tokens []xml.Token, m string) []xml.Token {
	b := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: "soap:Body",
		},
	}

	r := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: m,
		},
	}

	return append(tokens, r, b)
}
