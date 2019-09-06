package datatype

import (
	"encoding/xml"
	"io"
	"io/ioutil"

	. "github.com/journeymidnight/yig/error"
	"github.com/journeymidnight/yig/helper"
)

const MaxBucketWebsiteRulesCount = 100

type WebsiteConfiguration struct {
	XMLName               xml.Name               `xml:"WebsiteConfiguration"`
	Xmlns                 string                 `xml:"xmlns,attr,omitempty"`
	RedirectAllRequestsTo *RedirectAllRequestsTo `xml:"RedirectAllRequestsTo,omitempty"`
	IndexDocument         *IndexDocument         `xml:"IndexDocument,omitempty"`
	ErrorDocument         *ErrorDocument         `xml:"ErrorDocument,omitempty"`
	RoutingRules          []*RoutingRule         `xml:"RoutingRules>RoutingRule,omitempty"`
}

type RedirectAllRequestsTo struct {
	XMLName  xml.Name `xml:"RedirectAllRequestsTo"`
	HostName string   `xml:"HostName"`
	Protocol string   `xml:"Protocol,omitempty"`
}

type IndexDocument struct {
	XMLName xml.Name `xml:"IndexDocument"`
	Suffix  string   `xml:"Suffix"`
}

type ErrorDocument struct {
	XMLName xml.Name `xml:"ErrorDocument"`
	Key     string   `xml:"Key"`
}

type RoutingRule struct {
	XMLName   xml.Name   `xml:"RoutingRule"`
	Condition *Condition `xml:"Condition,omitempty"`
	Redirect  *Redirect  `xml:"Redirect"`
}

type Condition struct {
	XMLName                     xml.Name `xml:"Condition"`
	KeyPrefixEquals             string   `xml:"KeyPrefixEquals,omitempty"`
	HttpErrorCodeReturnedEquals string   `xml:"HttpErrorCodeReturnedEquals,omitempty"`
}

type Redirect struct {
	XMLName              xml.Name `xml:"Redirect"`
	Protocol             string   `xml:"Protocol,omitempty"`
	HostName             string   `xml:"HostName,omitempty"`
	ReplaceKeyPrefixWith string   `xml:"ReplaceKeyPrefixWith,omitempty"`
	ReplaceKeyWith       string   `xml:"ReplaceKeyWith,omitempty"`
	HttpRedirectCode     string   `xml:"HttpRedirectCode,omitempty"`
}

func (w *WebsiteConfiguration) Validate() (error error) {
	if w.RedirectAllRequestsTo != nil {
		protocol := w.RedirectAllRequestsTo.Protocol
		if protocol != "" && protocol != "http" && protocol != "https" {
			return ErrInvalidWebsiteRedirectProtocol
		}
	}
	if w.RoutingRules != nil {
		if len(w.RoutingRules) > MaxBucketWebsiteRulesCount {
			return ErrExceededWebsiteRoutingRulesLimit
		}
		for _, r := range w.RoutingRules {
			protocol := r.Redirect.Protocol
			if protocol != "" && protocol != "http" && protocol != "https" {
				return ErrInvalidWebsiteRedirectProtocol
			}
		}
	}
	return
}

func ParseWebsiteConfig(reader io.Reader) (*WebsiteConfiguration, error) {
	websiteConfig := new(WebsiteConfiguration)
	websiteBuffer, err := ioutil.ReadAll(reader)
	if err != nil {
		helper.ErrorIf(err, "Unable to read website config body")
		return nil, ErrInvalidWebsiteConfiguration
	}
	err = xml.Unmarshal(websiteBuffer, websiteConfig)
	if err != nil {
		helper.ErrorIf(err, "Unable to parse website config xml body")
		return nil, ErrMalformedWebsiteConfiguration
	}
	err = websiteConfig.Validate()
	if err != nil {
		return nil, err
	}
	return websiteConfig, nil
}
