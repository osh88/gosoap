package gosoap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
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
func (c *Client) Call(m string, p Params) *http.Request {
	c.Method = m
	c.Params = p

	c.payload, _ = xml.MarshalIndent(c, "", "")
	//if err != nil {
	//	fmt.Println(err)
	//}

	req := c.doRequest(c.Definitions.Services[0].Ports[0].SoapAddresses[0].Location)

	//b, err := c.doRequest(c.Definitions.Services[0].Ports[0].SoapAddresses[0].Location)
	//if err != nil {
	//	return err
	//}

	//var soap SoapEnvelope
	//err = xml.Unmarshal(b, &soap)
	//
	//c.Body = soap.Body.Contents
	//c.Header = soap.Header.Contents

	return req
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
func (c *Client) doRequest(url string) *http.Request {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(c.payload))
	if err != nil {
		fmt.Println(err)
	}

	req.ContentLength = int64(len(c.payload))
	req.Header.Add("Content-Type", "text/xml;charset=UTF-8")
	req.Header.Add("Accept", "text/xml")

	soapAction := fmt.Sprintf("%s/%s", c.URL, c.Method)
	for _,oper := range c.Definitions.Bindings[0].Operations {
		if oper.Name == c.Method {
			soapAction = oper.SoapOperations[0].SoapAction
		}
	}
	req.Header.Add("SOAPAction", soapAction)

	return req
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
