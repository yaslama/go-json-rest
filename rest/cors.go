package rest

import (
	"net/http"
	"strings"
)

// CorsMiddleware provides a configurable CORS implementation.
type CorsMiddleware struct {
	allowedMethods map[string]bool
	allowedHeaders map[string]bool

	// Reject non CORS requests if true. See CorsInfo.IsCors.
	RejectNonCorsRequests bool

	// Function excecuted for every CORS requests to validate the Origin. (Required)
	// Must return true if valid, false if invalid.
	// For instance: simple equality, regexp, DB lookup, ...
	OriginValidator func(origin string, request *Request) bool

	// List of allowed HTTP methods. Note that the comparison will be made in uppercase
	// to avoid common mistakes. And that the Access-Control-Allow-Methods response header
	// also uses uppercase.
	// (see CorsInfo.AccessControlRequestMethod)
	AllowedMethods []string

	// List of allowed HTTP Headers. Note that the comparison will be made with
	// noarmalized names (AccessControlRequestHeaders). And that the response header
	// also uses normalized names.
	// (see CorsInfo.AccessControlRequestHeaders)
	AllowedHeaders []string

	// User to se the Access-Control-Allow-Credentials response header.
	AccessControlAllowCredentials bool

	// Used to set the Access-Control-Max-Age response header, in seconds.
	AccessControlMaxAge int
}

func (mw *CorsMiddleware) MiddlewareFunc(handler HandlerFunc) HandlerFunc {
	return func(writer ResponseWriter, request *Request) {

		corsInfo := request.GetCorsInfo()

		// non CORS requests
		if !corsInfo.IsCors {
			if mw.RejectNonCorsRequests {
				Error(writer, "Non CORS request", http.StatusForbidden)
				return
			}
			// continue, execute the wrapped middleware
			handler(writer, request)
			return
		}

		// Validate the Origin
		if mw.OriginValidator(corsInfo.Origin, request) == false {
			Error(writer, "Invalid Origin", http.StatusForbidden)
			return
		}

		if corsInfo.IsPreflight {

			// check the request methods
			if mw.allowedMethods == nil {
				mw.allowedMethods = map[string]bool{}
				for _, allowedMethod := range mw.AllowedMethods {
					mw.allowedMethods[strings.ToUpper(allowedMethod)] = true
				}

			}
			if mw.allowedMethods[corsInfo.AccessControlRequestMethod] == false {
				Error(writer, "Invalid Preflight Request", http.StatusForbidden)
				return
			}

			// check the request headers
			if mw.allowedHeaders == nil {
				mw.allowedHeaders = map[string]bool{}
				for _, allowedHeader := range mw.AllowedHeaders {
					mw.allowedHeaders[http.CanonicalHeaderKey(allowedHeader)] = true
				}

			}
			for _, requestedHeader := range corsInfo.AccessControlRequestHeaders {
				if mw.allowedHeaders[requestedHeader] == false {
					Error(writer, "Invalid Preflight Request", http.StatusForbidden)
					return
				}
			}

			for allowedMethod, _ := range mw.allowedMethods {
				writer.Header().Add("Access-Control-Allow-Methods", allowedMethod)
			}
			for allowedHeader, _ := range mw.allowedHeaders {
				writer.Header().Add("Access-Control-Allow-Headers", allowedHeader)
			}
			writer.Header().Set("Access-Control-Allow-Origin", corsInfo.Origin)
			if mw.AccessControlAllowCredentials == true {
				writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			writer.Header().Set("Access-Control-Max-Age", string(mw.AccessControlMaxAge))
			writer.WriteHeader(http.StatusOK)
			return
		} else {
			writer.Header().Set("Access-Control-Expose-Headers", "X-Powered-By") // TODO
			writer.Header().Set("Access-Control-Allow-Origin", corsInfo.Origin)
			if mw.AccessControlAllowCredentials == true {
				writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			// continure, execute the wrapped middleware
			handler(writer, request)
			return
		}
	}
}
