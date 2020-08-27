package gosoap

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/valyala/fasthttp"
	"net/http"
	"strings"
	"time"
)

// Client struct hold all the informations about WSDL,
// request and response of the server
type Helper struct {
	wsdlUrl     string
	url         string
	definitions *wsdlDefinitions
	location    string
}

type RawToken string
const rawTokenPrefix = "##RawToken:"

type Param struct {
	K, V string
}

// Пометить значение как сырое
// (сырое значение будет записано как есть без экранизации)
func (o *Param) SetRaw(raw bool) {
	// Если нужно пометить как сырой xml
	if raw && !o.IsRaw() {
		o.V = rawTokenPrefix + o.V
	}

	// Если нужно убрать признак сырого xml
	if !raw && o.IsRaw() {
		o.V = o.V[len(rawTokenPrefix):]
	}
}

func (o *Param) IsRaw() bool {
	return strings.HasPrefix(o.V, rawTokenPrefix)
}

func (o *Param) GetV() string {
	if o.IsRaw() {
		return o.V[len(rawTokenPrefix):]
	} else {
		return o.V
	}
}

// Params type is used to set the params in soap request
type Params []Param

func (ps *Params) Get(K string) (string, bool) {
	for _, p := range *ps {
		if p.K == K {
			return p.V, true
		}
	}

	return "", false
}

func (ps *Params) Set(K,V string) {
	// Если параметр уже есть в списке, изменяем значение
	for i, p := range *ps {
		if p.K == K {
			(*ps)[i].V = V
			return
		}
	}

	// Если параметра нет, добавляем
	*ps = append(*ps, Param{K, V})
}

// SoapClient return new *Client to handle the requests with the WSDL
func NewHelper(wsdlURL string) (*Helper, error) {
	client := http.Client{
		Timeout: 5*time.Second,
	}
	r, err := client.Get(wsdlURL)
	if r != nil && r.Body != nil {
		defer r.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	def := new(wsdlDefinitions)
	if err := XmlUnmarshal(r.Body, def); err != nil {
		return nil, err
	}

	h := &Helper{
		wsdlUrl:     wsdlURL,
		url:         strings.TrimSuffix(def.TargetNamespace, "/"),
		definitions: def,
	}

	if h.location = h.getLocation(); h.location == "" {
		return nil, errors.New("gosoap.NewHelper: soap address not found")
	}

	return h, nil
}

func (c *Helper) getLocation() string {
	for _, service := range c.definitions.Services {
		for _, ports := range service.Ports {
			for _, addr := range ports.SoapAddresses {
				return addr.Location
			}
		}
	}

	return ""
}

func (c *Helper) getSOAPAction(method string) string {
	for _, binding := range c.definitions.Bindings {
		for _, bOperation := range binding.Operations {
			if bOperation.Name == method {
				for _, sOperation := range bOperation.SoapOperations {
					return sOperation.SoapAction
				}
			}
		}
	}

	return fmt.Sprintf("%s/%s", c.url, method)
}

// Заполняет поля fasthttp.Request
func (c *Helper) FillFastRequest(req *fasthttp.Request, method string, params *Params, headerName string, headerParams *Params) error {
	payload, err := c.encode(method, params, headerName, headerParams)
	if err != nil {
		return fmt.Errorf("gosoap.FillFastRequest: %v", err)
	}

	req.Header.SetMethod("POST")
	req.Header.SetRequestURI(c.location)
	req.Header.Set("Content-Type", "text/xml;charset=UTF-8")
	req.Header.Set("Accept", "text/xml")
	req.Header.Set("SOAPAction", c.getSOAPAction(method))
	req.Header.SetContentLength(len(payload))
	req.SetBody(payload)

	return nil
}

func (c *Helper) CheckError(data []byte) error {
	envelope := &struct {
		Fault struct {
			FaultCode   string `xml:"faultcode"`
			FaultString string `xml:"faultstring"`
		} `xml:"Body>Fault"`
	}{}

	// Разбор XML медленный, поэтому наличие тега с ошибкой
	// проверяем сначала простым поиском
	if bytes.Index(data, []byte(`Fault>`)) > -1 {
		if err := xml.Unmarshal(data, envelope); err != nil {
			return err
		}

		if envelope.Fault.FaultCode != "" || envelope.Fault.FaultString != "" {
			return fmt.Errorf("%s (%s)", envelope.Fault.FaultCode, envelope.Fault.FaultString)
		}
	}

	return nil
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
