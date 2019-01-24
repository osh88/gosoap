package gosoap

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/valyala/fasthttp"
	"net/url"
	"strings"
)

// Params type is used to set the params in soap request
type Params map[string]string

// SoapClient return new *Client to handle the requests with the WSDL
func SoapClient(wsdl string) (*Client, error) {
	_, err := url.Parse(wsdl)
	if err != nil {
		return nil, err
	}

	d, err := getWsdlDefinitions(wsdl)
	if err != nil {
		return nil, err
	}

	c := &Client{
		WSDL:        wsdl,
		URL:         strings.TrimSuffix(d.TargetNamespace, "/"),
		Definitions: d,
	}

	return c, nil
}

// Client struct hold all the informations about WSDL,
// request and response of the server
type Client struct {
	WSDL         string
	URL          string
	Method       string
	Params       Params
	ParamsOrder  []string
	HeaderName   string
	HeaderParams Params
	Definitions  *wsdlDefinitions
	Body         []byte
	Header       []byte
	Cookies      map[string]string

	payload []byte
}

func (c *Client) GetLastRequest() []byte {
	return c.payload
}

// Call call's the method m with Params p
func (c *Client) FillFastRequest(req *fasthttp.Request, m string, p Params) error {
	c.Method = m
	c.Params = p
	var err error
	c.payload, err = xml.Marshal(c)
	if err != nil {
		return fmt.Errorf("gosoap.FillFastRequest: %v", err)
	}

	for _, service := range c.Definitions.Services {
		for _, ports := range service.Ports {
			for _, addr := range ports.SoapAddresses {
				c.fillRequest(req, addr.Location)
				return nil
			}
		}
	}

	return errors.New("gosoap.FillFastRequest: soap address not found")
}

// Unmarshal get the body and unmarshal into the interface
func (c *Client) Unmarshal(v interface{}) error {
	if len(c.Body) == 0 {
		return fmt.Errorf("Body is empty")
	}

	var f Fault
	err := XmlUnmarshal(bytes.NewReader(c.Body), &f)
	if err != nil {
		return err
	}

	if f.Code != "" {
		return fmt.Errorf("[%s]: %s", f.Code, f.Description)
	}

	return XmlUnmarshal(bytes.NewReader(c.Body), v)
}

// doRequest makes new request to the server using the c.Method, c.URL and the body.
// body is enveloped in Call method
func (c *Client) fillRequest(req *fasthttp.Request, url string) {
	req.Header.SetMethod("POST")
	req.Header.SetRequestURI(url)
	req.Header.Set("Content-Type", "text/xml;charset=UTF-8")
	req.Header.Set("Accept", "text/xml")

	soapAction := fmt.Sprintf("%s/%s", c.URL, c.Method)
	if len(c.Definitions.Bindings) > 0 {
		for _, oper := range c.Definitions.Bindings[0].Operations {
			if oper.Name == c.Method && len(oper.SoapOperations) > 0 {
				soapAction = oper.SoapOperations[0].SoapAction
			}
		}
	}
	req.Header.Set("SOAPAction", soapAction)

	req.Header.SetContentLength(len(c.payload))
	req.SetBody(c.payload)
	return
}

// SoapEnvelope struct
type SoapEnvelope struct {
	XMLName struct{} `xml:"Envelope"`
	Header  SoapHeader
	Body    SoapBody
}

// SoapHeader struct
type SoapHeader struct {
	XMLName  struct{} `xml:"Header"`
	Contents []byte   `xml:",innerxml"`
}

// SoapBody struct
type SoapBody struct {
	XMLName  struct{} `xml:"Body"`
	Contents []byte   `xml:",innerxml"`
}
