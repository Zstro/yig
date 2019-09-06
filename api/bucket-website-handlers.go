package api

import (
	"io"
	"net/http"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/journeymidnight/yig/api/datatype"
	. "github.com/journeymidnight/yig/api/datatype"
	"github.com/journeymidnight/yig/api/datatype/policy"
	. "github.com/journeymidnight/yig/error"
	"github.com/journeymidnight/yig/helper"
	"github.com/journeymidnight/yig/iam/common"
	"github.com/journeymidnight/yig/signature"
)

const (
	MaxBucketWebsiteConfigurationSize = 20 * humanize.KiByte
)

func (api ObjectAPIHandlers) PutBucketWebsiteHandler(w http.ResponseWriter, r *http.Request) {
	helper.Debugln("PutBucketWebsiteHandler", "enter")
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	var credential common.Credential
	var err error
	switch signature.GetRequestAuthType(r) {
	default:
		// For all unknown auth types return error.
		WriteErrorResponse(w, r, ErrAccessDenied)
		return
	case signature.AuthTypeAnonymous:
		break
	case signature.AuthTypePresignedV4, signature.AuthTypeSignedV4,
		signature.AuthTypePresignedV2, signature.AuthTypeSignedV2:
		if credential, err = signature.IsReqAuthenticated(r); err != nil {
			WriteErrorResponse(w, r, err)
			return
		}
	}

	// Error out if Content-Length is missing.
	// PutBucketPolicy always needs Content-Length.
	if r.ContentLength <= 0 {
		WriteErrorResponse(w, r, ErrMissingContentLength)
		return
	}

	// Error out if Content-Length is beyond allowed size.
	if r.ContentLength > MaxBucketWebsiteConfigurationSize {
		WriteErrorResponse(w, r, ErrEntityTooLarge)
		return
	}

	websiteConfig, err := ParseWebsiteConfig(io.LimitReader(r.Body, r.ContentLength))
	if err != nil {
		WriteErrorResponse(w, r, err)
		return
	}

	err = api.ObjectAPI.SetBucketWebsite(credential, bucket, *websiteConfig)
	if err != nil {
		helper.ErrorIf(err, "Unable to set website for bucket.")
		WriteErrorResponse(w, r, err)
		return
	}
	WriteSuccessResponse(w, nil)
}

func (api ObjectAPIHandlers) GetBucketWebsiteHandler(w http.ResponseWriter, r *http.Request) {
	helper.Debugln("GetBucketPolicyHandler", "enter")
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	var credential common.Credential
	var err error
	switch signature.GetRequestAuthType(r) {
	default:
		// For all unknown auth types return error.
		WriteErrorResponse(w, r, ErrAccessDenied)
		return
	case signature.AuthTypeAnonymous:
		break
	case signature.AuthTypePresignedV4, signature.AuthTypeSignedV4,
		signature.AuthTypePresignedV2, signature.AuthTypeSignedV2:
		if credential, err = signature.IsReqAuthenticated(r); err != nil {
			WriteErrorResponse(w, r, err)
			return
		}
	}

	// Read bucket access policy.
	bucketWebsite, err := api.ObjectAPI.GetBucketWebsite(credential, bucket)
	if err != nil {
		WriteErrorResponse(w, r, err)
		return
	}

	encodedSuccessResponse, err := xmlFormat(bucketWebsite)
	if err != nil {
		helper.ErrorIf(err, "Failed to marshal Website XML for bucket", bucket)
		WriteErrorResponse(w, r, ErrInternalError)
		return
	}

	setXmlHeader(w)
	// Write to client.
	WriteSuccessResponse(w, encodedSuccessResponse)
}

func (api ObjectAPIHandlers) DeleteBucketWebsiteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	var credential common.Credential
	var err error
	switch signature.GetRequestAuthType(r) {
	default:
		// For all unknown auth types return error.
		WriteErrorResponse(w, r, ErrAccessDenied)
		return
	case signature.AuthTypeAnonymous:
		break
	case signature.AuthTypePresignedV4, signature.AuthTypeSignedV4,
		signature.AuthTypePresignedV2, signature.AuthTypeSignedV2:
		if credential, err = signature.IsReqAuthenticated(r); err != nil {
			WriteErrorResponse(w, r, err)
			return
		}
	}

	if err := api.ObjectAPI.DeleteBucketWebsite(credential, bucket); err != nil {
		WriteErrorResponse(w, r, err)
		return
	}
	// Success.
	WriteSuccessNoContent(w)
}

func needBeHandledByWebsite(r *http.Request) bool {
	ctx := getRequestContext(r)
	if strings.HasSuffix(ctx.ObjectName, "/") || ctx.ObjectName == "" {
		if ctx.AuthType != signature.AuthTypeAnonymous {
			return false
		}
		if ctx.BucketInfo == nil {
			return false
		}
		return true
	} else {
		return false
	}
}

func (api ObjectAPIHandlers) WebsiteRedirect(w http.ResponseWriter, r *http.Request) (handled bool) {
	w.(*ResponseRecorder).operationName = "GetObject"
	ctx := getRequestContext(r)
	handled = true
	if ctx.BucketInfo == nil {
		WriteErrorResponse(w, r, ErrNoSuchBucket)
		return
	}
	website := ctx.BucketInfo.Website
	if redirect := website.RedirectAllRequestsTo; redirect != nil && redirect.HostName != "" {
		protocol := redirect.Protocol
		if protocol == "" {
			protocol = r.URL.Scheme
		}
		http.Redirect(w, r, protocol+"://"+redirect.HostName, http.StatusFound)
		return true
	} else {
		handled = false
		return
	}
}

func (api ObjectAPIHandlers) ReturnWebsiteIndexDocument(w http.ResponseWriter, r *http.Request) (handled bool) {
	handled = api.WebsiteRedirect(w, r)
	if handled {
		return
	}
	w.(*ResponseRecorder).operationName = "GetObject"
	ctx := getRequestContext(r)
	handled = true
	if ctx.BucketInfo == nil {
		WriteErrorResponse(w, r, ErrNoSuchBucket)
		return
	}
	website := ctx.BucketInfo.Website
	if id := website.IndexDocument; id != nil && id.Suffix != "" {
		indexName := ctx.ObjectName + id.Suffix
		credential := common.Credential{}
		err := IsBucketPolicyAllowed(&credential, ctx.BucketInfo, r, policy.GetObjectAction, indexName)
		if err != nil {
			WriteErrorResponse(w, r, err)
			return
		}
		index, err := api.ObjectAPI.GetObjectInfo(ctx.BucketName, indexName, "", credential)
		if err != nil {
			if err == ErrNoSuchKey {
				api.errAllowableObjectNotFound(w, r, credential)
				return
			}
			WriteErrorResponse(w, r, err)
			return
		}
		writer := newGetObjectResponseWriter(w, r, index, nil, "")
		// Reads the object at startOffset and writes to mw.
		if err := api.ObjectAPI.GetObject(index, 0, index.Size, writer, datatype.SseRequest{}); err != nil {
			helper.ErrorIf(err, "Unable to write to client.")
			if !writer.dataWritten {
				// Error response only if no data has been written to client yet. i.e if
				// partial data has already been written before an error
				// occurred then no point in setting StatusCode and
				// sending error XML.
				WriteErrorResponse(w, r, err)
			}
			return
		}
		if !writer.dataWritten {
			// If ObjectAPI.GetObject did not return error and no data has
			// been written it would mean that it is a 0-byte object.
			// call wrter.Write(nil) to set appropriate headers.
			writer.Write(nil)
		}
		return
	} else {
		handled = false
		return
	}
}

func (api ObjectAPIHandlers) ReturnWebsiteErrorDocument(w http.ResponseWriter, r *http.Request) (handled bool) {
	w.(*ResponseRecorder).operationName = "GetObject"
	ctx := getRequestContext(r)
	if ctx.BucketInfo == nil {
		WriteErrorResponse(w, r, ErrNoSuchBucket)
		return true
	}
	website := ctx.BucketInfo.Website
	if ed := website.ErrorDocument; ed != nil && ed.Key != "" {
		indexName := ctx.ObjectName + ed.Key
		credential := new(common.Credential)
		err := IsBucketPolicyAllowed(credential, ctx.BucketInfo, r, policy.GetObjectAction, indexName)
		if err != nil {
			WriteErrorResponse(w, r, err)
			return true
		}
		index, err := api.ObjectAPI.GetObjectInfo(ctx.BucketName, indexName, "", *credential)
		if err != nil {
			WriteErrorResponse(w, r, err)
			return true
		}
		writer := newGetObjectResponseWriter(w, r, index, nil, "")
		// Reads the object at startOffset and writes to mw.
		if err := api.ObjectAPI.GetObject(index, 0, index.Size, writer, datatype.SseRequest{}); err != nil {
			helper.ErrorIf(err, "Unable to write to client.")
			if !writer.dataWritten {
				// Error response only if no data has been written to client yet. i.e if
				// partial data has already been written before an error
				// occurred then no point in setting StatusCode and
				// sending error XML.
				WriteErrorResponse(w, r, err)
			}
			return true
		}
		if !writer.dataWritten {
			// If ObjectAPI.GetObject did not return error and no data has
			// been written it would mean that it is a 0-byte object.
			// call wrter.Write(nil) to set appropriate headers.
			writer.Write(nil)
		}
		return true
	} else {
		return false
	}
}
